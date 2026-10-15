package asset

// การไล่สาย (AST-57): ตัวเอง → ชั้นบนจากใกล้ → ไกล → GLOBAL · เลือกค่าที่ใกล้ที่สุดที่มีอยู่จริง

// Source — ที่มาของค่า (AST-25)
type Source string

const (
	SourceNone   Source = ""       // ไม่มีค่าเลย
	SourceSelf   Source = "self"   // ตัวเองตั้ง
	SourceUpline Source = "upline" // ได้มาจากชั้นบน
	SourceGlobal Source = "global" // ค่ากลาง
	SourceMixed  Source = "mixed"  // ทั้งชุดมาจากหลายที่
)

// ChainIDs — สายของบัญชี selfID: ตัวเอง แล้วชั้นบนจากใกล้ → ไกล
// parents = `user_agents.cnf.parent` เรียงจาก Superadmin ลงมาถึงผู้สร้างตรง ไม่รวมตัวเอง (MGMT-61)
func ChainIDs(selfID uint, parents []uint) []uint {
	chain := make([]uint, 0, len(parents)+1)
	chain = append(chain, selfID)
	for i := len(parents) - 1; i >= 0; i-- {
		chain = append(chain, parents[i])
	}
	return chain
}

// rank — ลำดับในสาย (0 = ตัวเอง) · GLOBAL (ownerID nil) = ไกลสุด · ไม่อยู่ในสาย = -1
func rank(chain []uint, ownerID *uint) int {
	if ownerID == nil {
		return len(chain)
	}
	for i, id := range chain {
		if id == *ownerID {
			return i
		}
	}
	return -1
}

// Nearest — index ของแถวที่ใกล้ที่สุดในสาย (nil = GLOBAL ไกลสุด) · แถวของบัญชีนอกสายถูกข้าม · ไม่พบ = -1
func Nearest(chain []uint, ownerIDs []*uint) int {
	best, bestRank := -1, -1
	for i, o := range ownerIDs {
		r := rank(chain, o)
		if r < 0 {
			continue
		}
		if best == -1 || r < bestRank {
			best, bestRank = i, r
		}
	}
	return best
}

// SourceOf — ที่มาของแถวที่เจ้าของคือ ownerID เมื่อมองจากบัญชี chain[0]
// คืน owner = id ของชั้นบนที่ให้ค่ามา เมื่อ source = upline (ไม่เช่นนั้น 0)
func SourceOf(chain []uint, ownerID *uint) (src Source, owner uint) {
	switch {
	case ownerID == nil:
		return SourceGlobal, 0
	case len(chain) > 0 && *ownerID == chain[0]:
		return SourceSelf, 0
	}
	return SourceUpline, *ownerID
}

// OverallSource — ที่มาของทั้งชุด: ว่าง = ไม่มี · ที่เดียวกันหมด = ค่านั้น · ต่างกัน = mixed (AST-25)
func OverallSource(sources []Source) Source {
	var first Source
	for _, s := range sources {
		if s == SourceNone {
			continue
		}
		switch {
		case first == SourceNone:
			first = s
		case first != s:
			return SourceMixed
		}
	}
	return first
}

// UsedCandidate — แถว asset_used ที่เป็นไปได้ของช่องหนึ่ง (เจ้าของอยู่ในสายหรือเป็น GLOBAL)
type UsedCandidate struct {
	SlotID     uint
	OwnerID    *uint // nil = GLOBAL
	ConfigID   uint
	StorageKey string
}

// ResolvedUsed — ค่าที่ใช้งานจริงของช่อง
type ResolvedUsed struct {
	UsedCandidate
	Source        Source
	InheritedFrom uint // id ชั้นบนที่ให้ค่ามา เมื่อ Source = upline
}

// ResolveUsed — เลือก used ที่ใกล้ที่สุดต่อช่อง (AST-25) · ช่องที่ไม่มีแถวในสายและ GLOBAL ไม่อยู่ในผล
func ResolveUsed(chain []uint, cands []UsedCandidate) map[uint]ResolvedUsed {
	bySlot := map[uint][]UsedCandidate{}
	for _, c := range cands {
		bySlot[c.SlotID] = append(bySlot[c.SlotID], c)
	}
	out := make(map[uint]ResolvedUsed, len(bySlot))
	for slot, list := range bySlot {
		owners := make([]*uint, len(list))
		for i, c := range list {
			owners[i] = c.OwnerID
		}
		idx := Nearest(chain, owners)
		if idx < 0 {
			continue
		}
		src, from := SourceOf(chain, list[idx].OwnerID)
		out[slot] = ResolvedUsed{UsedCandidate: list[idx], Source: src, InheritedFrom: from}
	}
	return out
}

// ThemeCandidate — แถว game_themes ที่เป็นไปได้ (เจ้าของอยู่ในสายหรือเป็น GLOBAL)
type ThemeCandidate struct {
	OwnerID *uint // nil = GLOBAL
	Theme   Theme
}

// ResolvedTheme — theme ที่ใช้งานจริง (ทั้ง 5 ค่าจากแถวเดียว — AST-37)
type ResolvedTheme struct {
	Theme         Theme
	Source        Source
	InheritedFrom uint
}

// ResolveTheme — เลือกแถวที่ใกล้ที่สุด ใช้ทั้งชุด ไม่ผสมทีละค่า (AST-37) · ไม่พบเลย ok = false
func ResolveTheme(chain []uint, cands []ThemeCandidate) (ResolvedTheme, bool) {
	owners := make([]*uint, len(cands))
	for i, c := range cands {
		owners[i] = c.OwnerID
	}
	idx := Nearest(chain, owners)
	if idx < 0 {
		return ResolvedTheme{}, false
	}
	src, from := SourceOf(chain, cands[idx].OwnerID)
	return ResolvedTheme{Theme: cands[idx].Theme, Source: src, InheritedFrom: from}, true
}

// Branding — ชื่อเว็บ + โลโก้ (MVP) · favicon / site_theme เลื่อน (AST-58, AST-59)
type Branding struct {
	SiteName string
	LogoKey  string
}

// HasValue — แถว "มี branding" = มี field ใดไม่ว่าง (AST-47)
func (b Branding) HasValue() bool { return b.SiteName != "" || b.LogoKey != "" }

// BrandingCandidate — แถว site_brandings ของบัญชีหนึ่งในสาย
type BrandingCandidate struct {
	OwnerID  uint
	Branding Branding
}

// ResolvedBranding — branding ที่ใช้งานจริง · Source = none เมื่อไม่มีใครตั้งเลย (ทุก field ว่าง)
type ResolvedBranding struct {
	Branding      Branding
	Source        Source
	InheritedFrom uint
}

// ResolveBranding — บัญชีใกล้สุดในสายที่ "มี branding" ใช้ทั้งแถว ห้ามผสม field จากคนละชั้น (AST-47)
// branding ไม่มี GLOBAL — ไม่เจอ = ว่างทุก field
func ResolveBranding(chain []uint, cands []BrandingCandidate) ResolvedBranding {
	var best *BrandingCandidate
	bestRank := -1
	for i := range cands {
		c := &cands[i]
		if !c.Branding.HasValue() {
			continue
		}
		id := c.OwnerID
		r := rank(chain, &id)
		if r < 0 {
			continue
		}
		if best == nil || r < bestRank {
			best, bestRank = c, r
		}
	}
	if best == nil {
		return ResolvedBranding{Source: SourceNone}
	}
	id := best.OwnerID
	src, from := SourceOf(chain, &id)
	return ResolvedBranding{Branding: best.Branding, Source: src, InheritedFrom: from}
}
