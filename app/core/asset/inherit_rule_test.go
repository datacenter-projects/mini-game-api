package asset

import (
	"reflect"
	"testing"
)

func idp(v uint) *uint { return &v }

func TestChainIDs(t *testing.T) { // AST-57: ตัวเอง → ชั้นบนจากใกล้ → ไกล
	tests := []struct {
		name    string
		self    uint
		parents []uint
		want    []uint
	}{
		{"ไม่มีชั้นบน (superadmin)", 1, nil, []uint{1}},
		{"agent01 ใต้ share01 ใต้ comp01 ใต้ superadmin", 15, []uint{1, 10, 12}, []uint{15, 12, 10, 1}},
		{"สายลึก 6 ชั้น", 60, []uint{1, 10, 20, 30, 40, 50}, []uint{60, 50, 40, 30, 20, 10, 1}},
	}
	for _, tt := range tests {
		if got := ChainIDs(tt.self, tt.parents); !reflect.DeepEqual(got, tt.want) {
			t.Errorf("%s: got %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestChainIDsDoesNotMutateParents(t *testing.T) {
	parents := []uint{1, 10, 12}
	_ = ChainIDs(15, parents)
	if !reflect.DeepEqual(parents, []uint{1, 10, 12}) {
		t.Errorf("ChainIDs แก้ slice ที่รับมา: %v", parents)
	}
}

func TestNearest(t *testing.T) { // AST-57
	chain := []uint{15, 12, 10, 1} // agent01 → share01 → comp01 → superadmin
	tests := []struct {
		name   string
		owners []*uint
		want   int
	}{
		{"ว่าง", nil, -1},
		{"ตัวเองใกล้สุด", []*uint{idp(12), idp(15), nil}, 1},
		{"ชั้นบนใกล้กว่าไกล", []*uint{idp(10), idp(12)}, 1},
		{"GLOBAL ไกลสุด", []*uint{nil, idp(10)}, 1},
		{"GLOBAL อย่างเดียว", []*uint{nil}, 0},
		{"บัญชีนอกสายถูกข้าม", []*uint{idp(99)}, -1},
		{"นอกสายปนกับชั้นบน", []*uint{idp(99), idp(10)}, 1},
	}
	for _, tt := range tests {
		if got := Nearest(chain, tt.owners); got != tt.want {
			t.Errorf("%s: got %d, want %d", tt.name, got, tt.want)
		}
	}
}

func TestSourceOf(t *testing.T) { // AST-25 · AST-53
	chain := []uint{15, 12, 10}
	tests := []struct {
		name      string
		owner     *uint
		wantSrc   Source
		wantOwner uint
	}{
		{"GLOBAL", nil, SourceGlobal, 0},
		{"ตัวเอง", idp(15), SourceSelf, 0},
		{"ชั้นบน", idp(12), SourceUpline, 12},
	}
	for _, tt := range tests {
		src, owner := SourceOf(chain, tt.owner)
		if src != tt.wantSrc || owner != tt.wantOwner {
			t.Errorf("%s: got (%s,%d), want (%s,%d)", tt.name, src, owner, tt.wantSrc, tt.wantOwner)
		}
	}
}

func TestOverallSource(t *testing.T) { // AST-25
	tests := []struct {
		name string
		in   []Source
		want Source
	}{
		{"ไม่มีเลย", nil, SourceNone},
		{"ว่างทุกช่อง", []Source{SourceNone, SourceNone}, SourceNone},
		{"global ทั้งหมด", []Source{SourceGlobal, SourceGlobal}, SourceGlobal},
		{"self ทั้งหมด (ช่องว่างไม่นับ)", []Source{SourceSelf, SourceNone, SourceSelf}, SourceSelf},
		{"upline + global", []Source{SourceUpline, SourceGlobal}, SourceMixed},
		{"self + upline + global", []Source{SourceSelf, SourceUpline, SourceGlobal}, SourceMixed},
	}
	for _, tt := range tests {
		if got := OverallSource(tt.in); got != tt.want {
			t.Errorf("%s: got %q, want %q", tt.name, got, tt.want)
		}
	}
}

func TestResolveUsed(t *testing.T) { // AST-24 · AST-25
	// agent01 (15) → share01 (12) → comp01 (10) → superadmin (1)
	chain := []uint{15, 12, 10, 1}
	cands := []UsedCandidate{
		{SlotID: 1, OwnerID: nil, ConfigID: 100, StorageKey: "g1"},     // ช่อง 1: GLOBAL
		{SlotID: 1, OwnerID: idp(12), ConfigID: 120, StorageKey: "s1"}, // ช่อง 1: share01
		{SlotID: 2, OwnerID: nil, ConfigID: 101, StorageKey: "g2"},     // ช่อง 2: GLOBAL อย่างเดียว
		{SlotID: 3, OwnerID: idp(15), ConfigID: 130, StorageKey: "a3"}, // ช่อง 3: ตัวเอง
		{SlotID: 3, OwnerID: idp(12), ConfigID: 121, StorageKey: "s3"}, // ช่อง 3: share01 (ไกลกว่า)
		{SlotID: 3, OwnerID: nil, ConfigID: 102, StorageKey: "g3"},     // ช่อง 3: GLOBAL
		{SlotID: 4, OwnerID: idp(99), ConfigID: 140, StorageKey: "x4"}, // ช่อง 4: นอกสายเท่านั้น → ไม่มีผล
	}
	got := ResolveUsed(chain, cands)
	if len(got) != 3 {
		t.Fatalf("ต้องได้ 3 ช่อง (ช่อง 4 ไม่มีค่า) ได้ %d: %+v", len(got), got)
	}
	check := func(slot uint, cfg uint, src Source, from uint) {
		t.Helper()
		r, ok := got[slot]
		if !ok {
			t.Fatalf("ช่อง %d ไม่มีในผล", slot)
		}
		if r.ConfigID != cfg || r.Source != src || r.InheritedFrom != from {
			t.Errorf("ช่อง %d: got cfg=%d src=%s from=%d, want cfg=%d src=%s from=%d",
				slot, r.ConfigID, r.Source, r.InheritedFrom, cfg, src, from)
		}
	}
	check(1, 120, SourceUpline, 12) // share01 ชนะ GLOBAL
	check(2, 101, SourceGlobal, 0)
	check(3, 130, SourceSelf, 0) // ตัวเองชนะชั้นบนและ GLOBAL
	if _, ok := got[4]; ok {
		t.Error("ช่อง 4 มีแต่ของบัญชีนอกสาย ต้องไม่มีผล")
	}
}

func TestResolveUsedEmpty(t *testing.T) {
	if got := ResolveUsed([]uint{15}, nil); len(got) != 0 {
		t.Errorf("ไม่มีแถว ต้องไม่มีผล: %+v", got)
	}
}

func TestResolveTheme(t *testing.T) { // AST-37: ไล่ทั้งชุด ไม่ผสมทีละค่า
	chain := []uint{15, 12, 10, 1}
	global := Theme{FontFamily: "noto", MainColor1: "#f7447e", MainColor2: "#ffffff", StatsBarColor: StatsBarColorMain1, StatsBarFontFamily: "prompt"}
	share := Theme{FontFamily: "kanit", MainColor1: "#1e90ff", MainColor2: "#000000", StatsBarColor: StatsBarColorMain2, StatsBarFontFamily: "mitr"}
	agent := Theme{FontFamily: "sarabun", MainColor1: "#00ff00", MainColor2: "#111111", StatsBarColor: StatsBarColorMain1, StatsBarFontFamily: "noto"}

	t.Run("GLOBAL อย่างเดียว", func(t *testing.T) {
		r, ok := ResolveTheme(chain, []ThemeCandidate{{OwnerID: nil, Theme: global}})
		if !ok || r.Source != SourceGlobal || r.Theme != global || r.InheritedFrom != 0 {
			t.Errorf("got %+v ok=%v", r, ok)
		}
	})
	t.Run("ชั้นบนชนะ GLOBAL ใช้ทั้ง 5 ค่า", func(t *testing.T) {
		r, ok := ResolveTheme(chain, []ThemeCandidate{{nil, global}, {idp(12), share}})
		if !ok || r.Source != SourceUpline || r.Theme != share || r.InheritedFrom != 12 {
			t.Errorf("got %+v ok=%v", r, ok)
		}
	})
	t.Run("ตัวเองชนะชั้นบน ไม่ผสมกับ share", func(t *testing.T) {
		r, ok := ResolveTheme(chain, []ThemeCandidate{{nil, global}, {idp(12), share}, {idp(15), agent}})
		if !ok || r.Source != SourceSelf || r.Theme != agent {
			t.Errorf("got %+v ok=%v", r, ok)
		}
	})
	t.Run("ไม่มีเลย", func(t *testing.T) {
		if _, ok := ResolveTheme(chain, nil); ok {
			t.Error("ไม่มีแถว ต้อง ok = false")
		}
	})
}

func TestResolveBranding(t *testing.T) { // AST-47
	chain := []uint{15, 12, 10, 1}
	tests := []struct {
		name  string
		cands []BrandingCandidate
		want  ResolvedBranding
	}{
		{"ไม่มีใครตั้ง", nil, ResolvedBranding{Source: SourceNone}},
		{"แถวว่างทุก field ไม่นับ", []BrandingCandidate{{12, Branding{}}}, ResolvedBranding{Source: SourceNone}},
		{"ชั้นบนมี", []BrandingCandidate{{12, Branding{"Share One", "k12"}}},
			ResolvedBranding{Branding: Branding{"Share One", "k12"}, Source: SourceUpline, InheritedFrom: 12}},
		{"ตัวเองตั้งแค่โลโก้ ไม่เอาชื่อของ share มาผสม",
			[]BrandingCandidate{{15, Branding{"", "k15"}}, {12, Branding{"Share One", "k12"}}},
			ResolvedBranding{Branding: Branding{"", "k15"}, Source: SourceSelf}},
		{"ใกล้กว่าชนะ", []BrandingCandidate{{10, Branding{"Com", "k10"}}, {12, Branding{"Share", ""}}},
			ResolvedBranding{Branding: Branding{"Share", ""}, Source: SourceUpline, InheritedFrom: 12}},
		{"นอกสายถูกข้าม", []BrandingCandidate{{99, Branding{"X", "kx"}}}, ResolvedBranding{Source: SourceNone}},
		{"ข้ามชั้นแรกที่ว่าง ไปเจอชั้นถัดไป", []BrandingCandidate{{12, Branding{}}, {10, Branding{"Com", ""}}},
			ResolvedBranding{Branding: Branding{"Com", ""}, Source: SourceUpline, InheritedFrom: 10}},
	}
	for _, tt := range tests {
		if got := ResolveBranding(chain, tt.cands); got != tt.want {
			t.Errorf("%s: got %+v, want %+v", tt.name, got, tt.want)
		}
	}
}
