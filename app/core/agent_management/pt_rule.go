package agentmanagement

// กฎค่าหุ้นส่วน (MGMT-16 – MGMT-25) · ทุกค่าเป็น bp (1% = 100 bp)

const (
	FullPTBP         = 10000 // 100% — Superadmin ได้รับ (MGMT-22)
	PTStepBP         = 50    // 0.5% (MGMT-18)
	MaxCommissionBP  = 100   // 1% (MGMT-18)
	CommissionStepBP = 10    // 0.1% (MGMT-18)
)

// PTGroup — กลุ่ม PT ใน API (MGMT-16)
type PTGroup string

const PTGroupGame PTGroup = "game"

// GroupGame — เกมหนึ่งในกลุ่ม
type GroupGame struct {
	Category string
	GameCode string
}

// groupGames — กลุ่ม → หมวด → เกม (MGMT-16) · เพิ่มเกม / กลุ่มที่นี่โดยไม่เปลี่ยน API / ตาราง
var groupGames = map[PTGroup][]GroupGame{
	PTGroupGame: {
		{"minigame", "coin_toss"},
		{"minigame", "rock_paper_scissors"},
		{"minigame", "scratch_card"},
	},
}

// Groups — กลุ่มทั้งหมดเรียงตามชื่อ
func Groups() []PTGroup { return []PTGroup{PTGroupGame} }

// GamesOf — เกมในกลุ่ม · กลุ่มที่ไม่มี = nil
func GamesOf(g PTGroup) []GroupGame { return groupGames[g] }

// GroupOfGame — กลุ่มของรหัสเกม · ไม่พบ = ok false
func GroupOfGame(gameCode string) (PTGroup, bool) {
	for g, games := range groupGames {
		for _, gg := range games {
			if gg.GameCode == gameCode {
				return g, true
			}
		}
	}
	return "", false
}

// IsPTStep — 0 ถึง max ทีละ 0.5%
func IsPTStep(bp int) bool { return bp >= 0 && bp <= FullPTBP && bp%PTStepBP == 0 }

// IsCommissionStep — 0 ถึง 1% ทีละ 0.1%
func IsCommissionStep(bp int) bool {
	return bp >= 0 && bp <= MaxCommissionBP && bp%CommissionStepBP == 0
}

// PTViolation — ผลการตรวจค่าหุ้นส่วน · service แปลงเป็น apperr
type PTViolation int

const (
	PTOK                  PTViolation = iota
	PTInvalidStep                     // 422 — ไม่ลงตาม step / เกินช่วง (MGMT-18)
	PTExceedsReceived                 // 402305 (MGMT-18, MGMT-22)
	PTSeamlessMasterLock              // 402307 (MGMT-19)
	PTForceRemainExceeded             // 402308 (MGMT-25)
	PTCommissionExceeded              // 402309 (MGMT-18)
)

// ChildPT — ค่าที่ผู้สร้างตั้งให้ลูก 1 กลุ่ม (MGMT-16)
type ChildPT struct {
	PTFromParentBP int
	ForceBP        int
	RemainBP       int
	CommissionBP   int
}

// ValidateChildPT — ตรวจค่าที่ผู้สร้างตั้งให้ลูก (MGMT-18, MGMT-19, MGMT-25)
// creatorReceivedBP = ค่าที่ผู้สร้างได้รับในกลุ่มนี้ · creatorIsSeamlessMaster = ผู้สร้างเป็น Company Seamless Master
func ValidateChildPT(v ChildPT, creatorReceivedBP int, creatorIsSeamlessMaster bool) PTViolation {
	if !IsPTStep(v.PTFromParentBP) || !IsPTStep(v.ForceBP) || !IsPTStep(v.RemainBP) {
		return PTInvalidStep
	}
	if v.CommissionBP < 0 || v.CommissionBP%CommissionStepBP != 0 {
		return PTInvalidStep
	}
	if v.CommissionBP > MaxCommissionBP {
		return PTCommissionExceeded
	}
	if creatorIsSeamlessMaster && (v.PTFromParentBP != creatorReceivedBP || v.ForceBP != 0 || v.RemainBP != 0) {
		return PTSeamlessMasterLock
	}
	if v.PTFromParentBP > creatorReceivedBP {
		return PTExceedsReceived
	}
	if v.ForceBP > v.PTFromParentBP || v.RemainBP > v.PTFromParentBP {
		return PTForceRemainExceeded
	}
	return PTOK
}

// ValidateMemberCommission — Member มีแค่ Commission (MGMT-21)
func ValidateMemberCommission(bp int) PTViolation {
	if bp < 0 || bp%CommissionStepBP != 0 {
		return PTInvalidStep
	}
	if bp > MaxCommissionBP {
		return PTCommissionExceeded
	}
	return PTOK
}

// ValidateOwnPT — ค่าถือ pt ที่บัญชีตั้งเอง (MGMT-22) · Company Seamless Master ล็อกที่ 0 (MGMT-19)
func ValidateOwnPT(ptBP, receivedBP int, isSeamlessMaster bool) PTViolation {
	if !IsPTStep(ptBP) {
		return PTInvalidStep
	}
	if isSeamlessMaster && ptBP != 0 {
		return PTSeamlessMasterLock
	}
	if ptBP > receivedBP {
		return PTExceedsReceived
	}
	return PTOK
}

// InitialOwnPT — บัญชีใหม่เริ่มถือทั้งหมดที่ได้รับ (MGMT-22) · Company Seamless Master = 0 (MGMT-19)
func InitialOwnPT(receivedBP int, isSeamlessMaster bool) int {
	if isSeamlessMaster {
		return 0
	}
	return receivedBP
}

// MinPTFromParent — ค่าต่ำสุดที่ผู้สร้างตั้งให้ลูกได้ (MGMT-24)
// = ค่าที่มากที่สุดระหว่าง pt ของลูก และค่าที่ลูกให้ลูกของมันแต่ละคน
func MinPTFromParent(childOwnPTBP int, grandchildrenPTFromParentBP []int) int {
	m := childOwnPTBP
	for _, v := range grandchildrenPTFromParentBP {
		if v > m {
			m = v
		}
	}
	return m
}

// GameSwitch — เปิด / ปิดเกมของบัญชี 1 ชั้น (MGMT-20)
type GameSwitch struct {
	Status     bool
	StatusGame bool
}

// IsGameOpen — เกมเล่นได้เมื่อ status ของกลุ่มและ status_game เป็น true ทุกชั้นตั้งแต่บัญชีนี้ถึงหัวสาย (MGMT-20)
func IsGameOpen(chain []GameSwitch) bool {
	for _, s := range chain {
		if !s.Status || !s.StatusGame {
			return false
		}
	}
	return true
}
