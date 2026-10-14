package gamerate

import (
	"reflect"
	"testing"

	"app/app/core/game"
)

// ชุดตัวอย่าง Coin / RPS: ขั้น 1–2 จาก diagram ของ lead · ขั้น 3–10 = ตัวคูณ 1.9^n ปัด 4 ตำแหน่ง · ชนะลดขั้นละ 2% (spec 5.2)
func diagramSteps() []Step {
	return []Step{{1.9, 50}, {3.61, 48}, {6.859, 46}, {13.0321, 44}, {24.761, 42},
		{47.0459, 40}, {89.3872, 38}, {169.8356, 36}, {322.6877, 34}, {613.1066, 32}}
}

// ตัวอย่าง Scratch ใน diagram: W 50% · 0.5×/35 · 1×/35 · 2×/18 · 5×/10 · 8×/2
func diagramScratch() Scratch {
	return Scratch{WinPercent: 50, Prizes: []Prize{{0.5, 35}, {1, 35}, {2, 18}, {5, 10}, {8, 2}}}
}

func TestEveryGameHasSpec(t *testing.T) { // RATE-01A: เพิ่มเกมใน core/game ต้องเพิ่มรูปแบบอัตราที่นี่ด้วย
	for _, g := range game.All() {
		s, ok := SpecOf(g.GameCode)
		if !ok {
			t.Fatalf("เกม %s ไม่มีรูปแบบอัตราใน gameSpecs", g.GameCode)
		}
		if s.Kind == KindSteps && s.StepCount <= 0 {
			t.Fatalf("เกม %s รูปแบบ steps ต้องมีจำนวนขั้น", g.GameCode)
		}
	}
	if _, ok := SpecOf("slot"); ok {
		t.Fatal("เกมที่ไม่มีต้องได้ ok false")
	}
	if s, _ := SpecOf(game.RockPaperScissors); s != (GameSpec{KindSteps, 10}) { // lead Q7: RPS แบบเดียวกับ Coin
		t.Fatalf("RPS spec %+v", s)
	}
}

func TestComputeStepsDiagram(t *testing.T) { // RATE-13 / RATE-22 · ตัวอย่าง 5% · 13.36%
	res, sum, issues := ComputeSteps(diagramSteps())
	if len(issues) != 0 {
		t.Fatalf("ชุดตัวอย่างต้องผ่าน ได้ %+v", issues)
	}
	want := []StepResult{
		{1, 1.9, 50, 50, 95, 5, 50},         // max_win = 99 ÷ 1.9 = 52.105… เกินเพดาน → 50
		{2, 3.61, 48, 24, 86.64, 13.36, 50}, // reach = 50% × 48% · edge = 100 − 24 × 3.61
	}
	if !reflect.DeepEqual(res[:2], want) {
		t.Fatalf("ขั้น 1–2 = %+v\nwant %+v", res[:2], want)
	}
	if res[9].MaxWinPercent != 50 { // เพดาน win_percent 50 (MaxWinPercent)
		t.Fatalf("ขั้น 10 max_win %v", res[9].MaxWinPercent)
	}
	if sum != (StepsSummary{MinHouseEdgePercent: 5, MinHouseEdgeStep: 1, MaxPayoutPercent: 95}) {
		t.Fatalf("summary %+v", sum)
	}
}

func TestComputeStepsHouseEdgeBoundary(t *testing.T) { // RATE-07 · lead RR2: ≥ 1% ผ่าน · ต่ำกว่า 1% ไม่ผ่าน
	steps := diagramSteps()
	tests := []struct {
		name  string
		step1 Step
		ok    bool
	}{
		{"edge = 1% พอดี", Step{1.98, 50}, true},
		{"edge 0.99998%", Step{1.98, 50.0001}, false},
		{"ค่าที่ max_win บอก (ปัดลง 49.7487) ตั้งได้จริง", Step{1.99, 49.7487}, true},
		{"สูงกว่า max_win 0.0001 ไม่ผ่าน", Step{1.99, 49.7488}, false},
	}
	for _, tt := range tests {
		s := append([]Step{tt.step1}, steps[1:]...)
		s[1].Multiplier = 3.61 // ยังมากกว่าขั้น 1
		_, _, issues := ComputeSteps(s)
		got := true
		for _, is := range issues {
			if is.Index == 0 {
				got = false
				if is.Code != RuleHouseEdge || is.Field != "win_percent" {
					t.Fatalf("%s: issue %+v", tt.name, is)
				}
			}
		}
		if got != tt.ok {
			t.Fatalf("%s: ผ่าน = %v want %v (issues %+v)", tt.name, got, tt.ok, issues)
		}
	}
}

func TestComputeStepsAllIssuesReported(t *testing.T) { // RE3 แสดงทุกขั้นที่ผิด · สรุปชี้ขั้นที่ร้านได้น้อยสุด
	s := diagramSteps()
	s[2].WinPercent = 61 // ขั้น 3 เกิน max_win (สูตร 60.1399… · แสดง 50 ตามเพดาน)
	s[4].WinPercent = 90 // ขั้น 5
	_, sum, issues := ComputeSteps(s)
	// ขั้น 6 ไม่ผ่านตามด้วย เพราะโอกาสถึงขั้น 6 สูงขึ้นจากขั้น 3 / 5 (reach สะสม)
	if len(issues) != 3 || issues[0].Index != 2 || issues[1].Index != 4 || issues[2].Index != 5 {
		t.Fatalf("issues %+v", issues)
	}
	if issues[0].MaxWinPercent != 50 {
		t.Fatalf("ขั้น 3 max_win %v", issues[0].MaxWinPercent)
	}
	if sum != (StepsSummary{MinHouseEdgePercent: -43.5504, MinHouseEdgeStep: 5, MaxPayoutPercent: 143.5504}) { // ขั้น 5 ร้านขาดทุนมากสุด
		t.Fatalf("summary %+v", sum)
	}
}

func TestComputeScratchDiagram(t *testing.T) { // RATE-33 / 34 · ตัวอย่าง 22.75% · max_win 99 ÷ 1.545
	res, issues := ComputeScratch(diagramScratch())
	if len(issues) != 0 {
		t.Fatalf("issues %+v", issues)
	}
	if res.AvgMultiplier != 1.545 || res.PayoutPercent != 77.25 || res.HouseEdgePercent != 22.75 || res.MaxWinPercent != 50 { // สูตร 99 ÷ 1.545 = 64.0776 เกินเพดาน → 50
		t.Fatalf("result %+v", res)
	}
	wantPer := []float64{17.5, 17.5, 9, 5, 1}
	wantPay := []float64{8.75, 17.5, 18, 25, 8} // รวม 77.25 = payout ของใบ
	for i, p := range res.Prizes {
		if p.RowNo != i+1 || p.PerTicketPercent != wantPer[i] || p.PayoutPercent != wantPay[i] {
			t.Fatalf("prize %d %+v", i, p)
		}
	}
}

func TestComputeScratchIssues(t *testing.T) { // RATE-32 / RATE-33
	tests := []struct {
		name  string
		edit  func(s *Scratch)
		codes []RuleCode
	}{
		{"W 65 ร้านขาดทุน", func(s *Scratch) { s.WinPercent = 65 }, []RuleCode{RuleHouseEdge}},
		{"W 64.0776 = house edge 1% พอดี (ช่วงตรวจที่ DTO)", func(s *Scratch) { s.WinPercent = 64.0776 }, nil},
		{"W 64.0777 เกิน", func(s *Scratch) { s.WinPercent = 64.0777 }, []RuleCode{RuleHouseEdge}},
		{"share รวม 99.5", func(s *Scratch) { s.Prizes[0].SharePercent = 34.5 }, []RuleCode{RuleShareSum}},
		{"share รวม 100.0001", func(s *Scratch) { s.Prizes[0].SharePercent = 35.0001 }, []RuleCode{RuleShareSum}},
		{"share ผิด + ขาดทุน", func(s *Scratch) { s.WinPercent = 90; s.Prizes[0].SharePercent = 30 }, []RuleCode{RuleShareSum, RuleHouseEdge}},
	}
	for _, tt := range tests {
		s := diagramScratch()
		tt.edit(&s)
		_, issues := ComputeScratch(s)
		var got []RuleCode
		for _, is := range issues {
			got = append(got, is.Code)
		}
		if !reflect.DeepEqual(got, tt.codes) {
			t.Fatalf("%s: %v want %v", tt.name, got, tt.codes)
		}
	}
}

func TestCheckStepsFormat(t *testing.T) { // RATE-10 / 11 / 20 / 21 · lead RR3
	tests := []struct {
		name string
		edit func(s []Step) []Step
		want *FormatIssue
	}{
		{"ผ่าน", func(s []Step) []Step { return s }, nil},
		{"9 ขั้น", func(s []Step) []Step { return s[:9] }, &FormatIssue{"steps", -1, ReasonStepCount}},
		{"ตัวคูณ < 1", func(s []Step) []Step { s[0].Multiplier = 0.99; return s }, &FormatIssue{"multiplier", 0, ReasonMultiplierRange}},
		{"ตัวคูณ > 1,000", func(s []Step) []Step { s[9].Multiplier = 1000.0001; return s }, &FormatIssue{"multiplier", 9, ReasonMultiplierRange}},
		{"ตัวคูณ 1,000 พอดี", func(s []Step) []Step { s[9].Multiplier = 1000; return s }, nil},
		{"ตัวคูณขั้น 4 ≤ ขั้น 3", func(s []Step) []Step { s[3].Multiplier = 6.859; return s }, &FormatIssue{"multiplier", 3, ReasonMultiplierOrder}},
		{"win 0", func(s []Step) []Step { s[1].WinPercent = 0; return s }, &FormatIssue{"win_percent", 1, ReasonWinRange}},
		{"win 50.0001 (เพดาน 50)", func(s []Step) []Step { s[1].WinPercent = 50.0001; return s }, &FormatIssue{"win_percent", 1, ReasonWinRange}},
		{"ผิดหลายจุด ได้ตัวแรก", func(s []Step) []Step { s[5].WinPercent = 0; s[2].Multiplier = 0; return s }, &FormatIssue{"multiplier", 2, ReasonMultiplierRange}},
	}
	for _, tt := range tests {
		got := CheckStepsFormat(tt.edit(diagramSteps()), 10)
		if !reflect.DeepEqual(got, tt.want) {
			t.Fatalf("%s: %+v want %+v", tt.name, got, tt.want)
		}
	}
}

func TestCheckScratchFormat(t *testing.T) { // RATE-30 / 31
	many := make([]Prize, 11)
	for i := range many {
		many[i] = Prize{float64(i + 1), 1}
	}
	tests := []struct {
		name string
		edit func(s *Scratch)
		want *FormatIssue
	}{
		{"ผ่าน", func(s *Scratch) {}, nil},
		{"win 0", func(s *Scratch) { s.WinPercent = 0 }, &FormatIssue{"win_percent", -1, ReasonWinRange}},
		{"win 50.0001 (เพดาน 50)", func(s *Scratch) { s.WinPercent = 50.0001 }, &FormatIssue{"win_percent", -1, ReasonWinRange}},
		{"win 50 พอดี", func(s *Scratch) { s.WinPercent = 50 }, nil},
		{"0 แถว", func(s *Scratch) { s.Prizes = nil }, &FormatIssue{"prizes", -1, ReasonPrizeCount}},
		{"11 แถว", func(s *Scratch) { s.Prizes = many }, &FormatIssue{"prizes", -1, ReasonPrizeCount}},
		{"รางวัล 0 เท่า", func(s *Scratch) { s.Prizes[1].Multiplier = 0 }, &FormatIssue{"multiplier", 1, ReasonMultiplierRange}},
		{"ตัวคูณซ้ำ", func(s *Scratch) { s.Prizes[2].Multiplier = 1 }, &FormatIssue{"multiplier", 2, ReasonPrizeDuplicate}},
		{"share 0", func(s *Scratch) { s.Prizes[4].SharePercent = 0 }, &FormatIssue{"share_percent", 4, ReasonShareRange}},
	}
	for _, tt := range tests {
		s := diagramScratch()
		tt.edit(&s)
		if got := CheckScratchFormat(s); !reflect.DeepEqual(got, tt.want) {
			t.Fatalf("%s: %+v want %+v", tt.name, got, tt.want)
		}
	}
}

func TestSame(t *testing.T) { // RATE-05C
	a, b := diagramSteps(), diagramSteps()
	if !SameSteps(a, b) {
		t.Fatal("ค่าเดียวกันต้องเท่ากัน")
	}
	b[9].WinPercent = 31.9999
	if SameSteps(a, b) {
		t.Fatal("ต่าง 0.0001 ต้องไม่เท่ากัน")
	}
	x, y := diagramScratch(), diagramScratch()
	if !SameScratch(x, y) {
		t.Fatal("Scratch เดียวกันต้องเท่ากัน")
	}
	y.Prizes[0], y.Prizes[1] = y.Prizes[1], y.Prizes[0] // ลำดับแถวต่าง = ไม่เท่า (เทียบตามลำดับ)
	if SameScratch(x, y) {
		t.Fatal("ลำดับแถวต่างต้องไม่เท่ากัน")
	}
}

func TestDiffSteps(t *testing.T) { // RE5 changes (Q4)
	if got := DiffSteps(nil, diagramSteps()); len(got) != 0 {
		t.Fatalf("เวอร์ชันแรกต้องไม่มีรายการ %+v", got)
	}
	prev, cur := diagramSteps(), diagramSteps()
	cur[0].WinPercent = 49
	cur[3].Multiplier = 12
	cur[3].WinPercent = 45
	got := DiffSteps(prev, cur)
	want := []Change{
		{Type: ChangeStep, Step: 1, Field: "win_percent", From: ptr(50), To: ptr(49)},
		{Type: ChangeStep, Step: 4, Field: "multiplier", From: ptr(13.0321), To: ptr(12)},
		{Type: ChangeStep, Step: 4, Field: "win_percent", From: ptr(44), To: ptr(45)},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("%+v\nwant %+v", got, want)
	}
}

func TestDiffStepsCountChanged(t *testing.T) { // เกมเปลี่ยนจำนวนขั้น: ขั้นที่หายไป To nil · ขั้นที่เพิ่ม From nil
	prev := []Step{{1.9, 50}, {3.61, 48}, {6.859, 46}}
	cur := []Step{{1.9, 50}, {3.61, 48}}
	want := []Change{
		{Type: ChangeStep, Step: 3, Field: "multiplier", From: ptr(6.859)},
		{Type: ChangeStep, Step: 3, Field: "win_percent", From: ptr(46)},
	}
	if got := DiffSteps(prev, cur); !reflect.DeepEqual(got, want) {
		t.Fatalf("ลดขั้น %+v", got)
	}
	got := DiffSteps(cur, prev)
	if len(got) != 2 || got[0].From != nil || *got[0].To != 6.859 || got[1].From != nil {
		t.Fatalf("เพิ่มขั้น %+v", got)
	}
}

func TestDiffScratch(t *testing.T) { // RE5 changes (Q4) · จับคู่รางวัลด้วย multiplier
	if got := DiffScratch(nil, diagramScratch()); len(got) != 0 {
		t.Fatalf("เวอร์ชันแรกต้องไม่มีรายการ %+v", got)
	}
	prev := Scratch{WinPercent: 35, Prizes: []Prize{{0.5, 20}, {1, 50}, {2, 30}}}
	cur := Scratch{WinPercent: 50, Prizes: []Prize{{1, 60}, {2, 30}, {5, 10}}}
	want := []Change{
		{Type: ChangeWinPercent, From: ptr(35), To: ptr(50)},
		{Type: ChangePrizeShare, Multiplier: 1, From: ptr(50), To: ptr(60)},
		{Type: ChangePrizeAdded, Multiplier: 5, To: ptr(10)},
		{Type: ChangePrizeRemoved, Multiplier: 0.5, From: ptr(20)},
	}
	if got := DiffScratch(&prev, cur); !reflect.DeepEqual(got, want) {
		t.Fatalf("%+v\nwant %+v", got, want)
	}
	if got := DiffScratch(&prev, prev); len(got) != 0 {
		t.Fatalf("ค่าเดิมต้องไม่มีรายการ %+v", got)
	}
}
