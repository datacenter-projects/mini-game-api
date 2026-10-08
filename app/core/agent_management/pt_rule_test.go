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
		received int
		master   bool
		want     PTViolation
	}{
		{"ปกติ", ChildPT{7000, 0, 0, 50}, 9000, false, PTOK},
		{"ให้เท่าที่ได้รับ", ChildPT{9000, 9000, 9000, 100}, 9000, false, PTOK},
		{"0 ทั้งหมด", ChildPT{0, 0, 0, 0}, 9000, false, PTOK},
		{"30.25 ไม่ลง 0.5", ChildPT{3025, 0, 0, 0}, 9000, false, PTInvalidStep},
		{"ติดลบ", ChildPT{-50, 0, 0, 0}, 9000, false, PTInvalidStep},
		{"force ไม่ลง step", ChildPT{5000, 10, 0, 0}, 9000, false, PTInvalidStep},
		{"commission 0.05", ChildPT{5000, 0, 0, 5}, 9000, false, PTInvalidStep},
		{"commission 1.01 → ไม่ลง step", ChildPT{5000, 0, 0, 101}, 9000, false, PTInvalidStep},
		{"commission 1.1", ChildPT{5000, 0, 0, 110}, 9000, false, PTCommissionExceeded},
		{"ได้รับ 80 ให้ 80.5", ChildPT{8050, 0, 0, 0}, 8000, false, PTExceedsReceived},
		{"ให้ 60 force 70", ChildPT{6000, 7000, 0, 0}, 9000, false, PTForceRemainExceeded},
		{"ให้ 60 remain 60.5", ChildPT{6000, 0, 6050, 0}, 9000, false, PTForceRemainExceeded},
		{"Seamless Master ให้เท่าที่ได้รับ", ChildPT{8000, 0, 0, 30}, 8000, true, PTOK},
		{"Seamless Master ให้น้อยกว่า", ChildPT{7500, 0, 0, 0}, 8000, true, PTSeamlessMasterLock},
		{"Seamless Master ตั้ง force", ChildPT{8000, 500, 0, 0}, 8000, true, PTSeamlessMasterLock},
	}
	for _, tt := range tests {
		if got := ValidateChildPT(tt.v, tt.received, tt.master); got != tt.want {
			t.Errorf("%s: ValidateChildPT = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestValidateMemberCommission(t *testing.T) { // MGMT-21
	tests := []struct {
		bp   int
		want PTViolation
	}{
		{0, PTOK},
		{60, PTOK},
		{100, PTOK},
		{110, PTCommissionExceeded},
		{15, PTInvalidStep},
		{-10, PTInvalidStep},
	}
	for _, tt := range tests {
		if got := ValidateMemberCommission(tt.bp); got != tt.want {
			t.Errorf("ValidateMemberCommission(%d) = %v, want %v", tt.bp, got, tt.want)
		}
	}
}

func TestValidateOwnPT(t *testing.T) { // MGMT-22, MGMT-19
	tests := []struct {
		pt, received int
		master       bool
		want         PTViolation
	}{
		{4000, 6000, false, PTOK},
		{6000, 6000, false, PTOK},
		{6100, 6000, false, PTExceedsReceived},
		{4025, 6000, false, PTInvalidStep},
		{0, 8000, true, PTOK},
		{100, 8000, true, PTSeamlessMasterLock},
	}
	for _, tt := range tests {
		if got := ValidateOwnPT(tt.pt, tt.received, tt.master); got != tt.want {
			t.Errorf("ValidateOwnPT(%d, %d, %v) = %v, want %v", tt.pt, tt.received, tt.master, got, tt.want)
		}
	}
}

func TestInitialOwnPT(t *testing.T) { // MGMT-22, MGMT-19
	if got := InitialOwnPT(7000, false); got != 7000 {
		t.Errorf("InitialOwnPT(7000) = %d, want 7000", got)
	}
	if got := InitialOwnPT(8000, true); got != 0 {
		t.Errorf("InitialOwnPT(master) = %d, want 0", got)
	}
}

func TestMinPTFromParent(t *testing.T) { // MGMT-24
	tests := []struct {
		name  string
		pt    int
		grand []int
		want  int
	}{
		{"ตัวอย่าง spec share1: ให้ agent1 60 · pt 30", 3000, []int{6000}, 6000},
		{"ไม่มีลูก", 3000, nil, 3000},
		{"pt มากกว่าลูกทุกคน", 7000, []int{5000, 2000}, 7000},
		{"หลายลูก", 1000, []int{5000, 7500, 2000}, 7500},
		{"ไม่ถือไม่มีลูก", 0, nil, 0},
	}
	for _, tt := range tests {
		if got := MinPTFromParent(tt.pt, tt.grand); got != tt.want {
			t.Errorf("%s: MinPTFromParent = %d, want %d", tt.name, got, tt.want)
		}
	}
}

func TestIsGameOpen(t *testing.T) { // MGMT-20
	on := GameSwitch{true, true}
	tests := []struct {
		name  string
		chain []GameSwitch
		want  bool
	}{
		{"เปิดทุกชั้น", []GameSwitch{on, on, on}, true},
		{"ปิดทั้งกลุ่มที่ตัวเอง", []GameSwitch{{false, true}, on}, false},
		{"หัวสายปิดเกม · ลูกเปิด", []GameSwitch{on, {true, false}}, false},
		{"ไม่มีชั้น", nil, true},
	}
	for _, tt := range tests {
		if got := IsGameOpen(tt.chain); got != tt.want {
			t.Errorf("%s: IsGameOpen = %v, want %v", tt.name, got, tt.want)
		}
	}
}
