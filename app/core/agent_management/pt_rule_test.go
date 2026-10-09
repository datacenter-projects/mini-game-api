package agentmanagement

import "testing"

func TestGroups(t *testing.T) { // MGMT-16
	games := GamesOf(PTGroupMinigame)
	if len(games) != 3 {
		t.Fatalf("GamesOf(game) = %d games, want 3", len(games))
	}
	for _, g := range games {
		if g.Category != "minigame" {
			t.Errorf("%s category = %q", g.GameCode, g.Category)
		}
		if grp, ok := GroupOfGame(g.GameCode); !ok || grp != PTGroupMinigame {
			t.Errorf("GroupOfGame(%s) = %q, %v", g.GameCode, grp, ok)
		}
	}
	if _, ok := GroupOfGame("baccarat"); ok {
		t.Error("GroupOfGame(baccarat) should be false")
	}
	if GamesOf("provider") != nil {
		t.Error("GamesOf(provider) should be nil")
	}
}

func TestValidateChildPT(t *testing.T) { // MGMT-18, MGMT-19, MGMT-25
	tests := []struct {
		name     string
		v        ChildPT
		received float64
		master   bool
		want     PTViolation
	}{
		{"ปกติ", ChildPT{70, 0, 0, 0.5}, 90, false, PTOK},
		{"ให้เท่าที่ได้รับ", ChildPT{90, 90, 90, 1}, 90, false, PTOK},
		{"0 ทั้งหมด", ChildPT{0, 0, 0, 0}, 90, false, PTOK},
		{"30.25 ไม่ลง 0.5", ChildPT{30.25, 0, 0, 0}, 90, false, PTInvalidStep},
		{"ติดลบ", ChildPT{-0.5, 0, 0, 0}, 90, false, PTInvalidStep},
		{"force ไม่ลง step", ChildPT{50, 0.1, 0, 0}, 90, false, PTInvalidStep},
		{"commission 0.05", ChildPT{50, 0, 0, 0.05}, 90, false, PTInvalidStep},
		{"commission 1.01 → ไม่ลง step", ChildPT{50, 0, 0, 1.01}, 90, false, PTInvalidStep},
		{"commission 1.1", ChildPT{50, 0, 0, 1.1}, 90, false, PTCommissionExceeded},
		{"ได้รับ 80 ให้ 80.5", ChildPT{80.5, 0, 0, 0}, 80, false, PTExceedsReceived},
		{"ให้ 60 force 70", ChildPT{60, 70, 0, 0}, 90, false, PTForceRemainExceeded},
		{"ให้ 60 remain 60.5", ChildPT{60, 0, 60.5, 0}, 90, false, PTForceRemainExceeded},
		{"Seamless Master ให้เท่าที่ได้รับ", ChildPT{80, 0, 0, 0.3}, 80, true, PTOK},
		{"Seamless Master ให้น้อยกว่า", ChildPT{75, 0, 0, 0}, 80, true, PTSeamlessMasterLock},
		{"Seamless Master ตั้ง force", ChildPT{80, 5, 0, 0}, 80, true, PTSeamlessMasterLock},
	}
	for _, tt := range tests {
		if got := ValidateChildPT(tt.v, tt.received, tt.master); got != tt.want {
			t.Errorf("%s: ValidateChildPT = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestValidateMemberCommission(t *testing.T) { // MGMT-21
	tests := []struct {
		bp   float64
		want PTViolation
	}{
		{0, PTOK},
		{0.6, PTOK},
		{1, PTOK},
		{1.1, PTCommissionExceeded},
		{0.15, PTInvalidStep},
		{-0.1, PTInvalidStep},
	}
	for _, tt := range tests {
		if got := ValidateMemberCommission(tt.bp); got != tt.want {
			t.Errorf("ValidateMemberCommission(%v) = %v, want %v", tt.bp, got, tt.want)
		}
	}
}

func TestValidateMemberPT(t *testing.T) { // MGMT-21 แก้ 2026-10-09
	tests := []struct {
		name     string
		pt       float64
		received float64
		want     PTViolation
	}{
		{"ถือทั้งหมด", 60, 60, PTOK},
		{"ไม่ถือ", 0, 60, PTOK},
		{"ถือบางส่วน", 30.5, 60, PTOK},
		{"เกินที่ได้รับ", 65, 60, PTExceedsReceived},
		{"ไม่ลง step", 30.25, 60, PTInvalidStep},
		{"ติดลบ", -0.5, 60, PTInvalidStep},
		{"เกิน 100", 100.5, 100, PTInvalidStep},
	}
	for _, tt := range tests {
		if got := ValidateMemberPT(tt.pt, tt.received); got != tt.want {
			t.Errorf("%s: ValidateMemberPT(%v, %v) = %v, want %v", tt.name, tt.pt, tt.received, got, tt.want)
		}
	}
}

func TestMemberRemain(t *testing.T) { // MGMT-21 แก้ 2026-10-09
	tests := []struct {
		received, pt, want float64
	}{
		{60, 50, 10},
		{60, 60, 0},
		{60, 0, 60},
	}
	for _, tt := range tests {
		if got := MemberRemain(tt.received, tt.pt); got != tt.want {
			t.Errorf("MemberRemain(%v, %v) = %v, want %v", tt.received, tt.pt, got, tt.want)
		}
	}
}

func TestMinPTFromParent(t *testing.T) { // MGMT-24 · R1 แก้ 2026-10-09
	tests := []struct {
		name    string
		grand   []float64
		members []float64
		want    float64
	}{
		{"ตัวอย่าง spec share1: ให้ agent1 60 · Member pt 30", []float64{60}, []float64{30}, 60},
		{"Member pt มากกว่าลูก", []float64{50, 20}, []float64{70}, 70},
		{"R1: Member pt 50 / 20 ไม่มีลูก", nil, []float64{50, 20}, 50},
		{"หลายลูก", []float64{50, 75, 20}, nil, 75},
		{"ไม่มีลูกไม่มี Member", nil, nil, 0},
	}
	for _, tt := range tests {
		if got := MinPTFromParent(tt.grand, tt.members); got != tt.want {
			t.Errorf("%s: MinPTFromParent = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestIsGameOpen(t *testing.T) { // MGMT-20
	tests := []struct {
		name  string
		chain []bool
		want  bool
	}{
		{"เปิดทุกชั้น", []bool{true, true, true}, true},
		{"ตัวเองปิดเกม", []bool{false, true}, false},
		{"หัวสายปิดเกม · ลูกเปิด", []bool{true, false}, false},
		{"ไม่มีชั้น", nil, true},
	}
	for _, tt := range tests {
		if got := IsGameOpen(tt.chain); got != tt.want {
			t.Errorf("%s: IsGameOpen = %v, want %v", tt.name, got, tt.want)
		}
	}
}
