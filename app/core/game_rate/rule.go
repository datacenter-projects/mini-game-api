// Package gamerate คือกฎและสูตรของอัตราแพ้ชนะ — spec: docs/modules/game_rate.md หัวข้อ 3
// pure function ไม่แตะ DB / config · ตรวจและคำนวณด้วยเลขแม่นยำ (big.Rat) ไม่ใช่ float (lead RR1)
// ทศนิยมไม่เกิน 4 ตำแหน่งตรวจที่ DTO (utils.IsDecimal4) · core รับค่าที่ผ่านรูปแบบนั้นแล้ว
package gamerate

import "app/app/core/game"

// Kind — รูปแบบอัตราของเกม (RATE-01A) · เกมใหม่ที่ใช้รูปแบบเดิม = เพิ่มแค่ใน gameSpecs ไม่ต้องแก้ API / ตาราง / service
// รูปแบบใหม่ = เพิ่ม Kind + กฎ / สูตรของรูปแบบนั้นที่นี่
type Kind string

const (
	KindSteps  Kind = "steps"  // ทายต่อเป็นขั้น แต่ละขั้น multiplier + win_percent (Coin Flip · Rock Paper Scissors)
	KindPrizes Kind = "prizes" // โอกาสชนะของใบ + ตารางรางวัล (Scratch Card)
)

// GameSpec — รูปแบบอัตราและจำนวนขั้นของเกม (StepCount ใช้กับ KindSteps เท่านั้น)
type GameSpec struct {
	Kind      Kind
	StepCount int
}

// gameSpecs — ทุกเกมใน core/game ต้องมีที่นี่ (test บังคับ)
var gameSpecs = map[string]GameSpec{
	game.CoinToss:          {KindSteps, 10}, // RATE-10
	game.RockPaperScissors: {KindSteps, 10}, // RATE-20 · ไม่มีเสมอ (lead Q7)
	game.ScratchCard:       {KindPrizes, 0}, // RATE-30
}

// SpecOf — รูปแบบอัตราของเกม · เกมที่ไม่มี = ok false (406401)
func SpecOf(gameCode string) (GameSpec, bool) {
	s, ok := gameSpecs[gameCode]
	return s, ok && game.IsGame(gameCode)
}

// ค่าคงที่ของกฎ (RATE-03 · RATE-04 · RATE-07 · RATE-31)
const (
	MinMultiplierStep   = 1.0    // ตัวคูณขั้นต่ำของรูปแบบ steps (RATE-11)
	MaxMultiplier       = 1000.0 // เพดานตัวคูณทุกเกม (lead RR3)
	MaxPercent          = 100.0  // win_percent / share_percent สูงสุด
	MinHouseEdgePercent = 1.0    // ร้านต้องได้อย่างน้อยทุกจุดที่หยุดได้ (lead RR2 · RATE-07)
	MinPrizes           = 1      // แถวรางวัล Scratch (RATE-31)
	MaxPrizes           = 10
)

// Step — 1 ขั้นของรูปแบบ steps (Coin / RPS · RPS ไม่มีเสมอ — lead Q7)
type Step struct {
	Multiplier float64
	WinPercent float64
}

// Prize — 1 แถวรางวัลของรูปแบบ prizes (Scratch Card)
type Prize struct {
	Multiplier   float64
	SharePercent float64
}

// Scratch — ค่าของรูปแบบ prizes (Scratch Card)
type Scratch struct {
	WinPercent float64
	Prizes     []Prize
}

// FormatReason — เหตุที่รูปแบบไม่ผ่าน (422) · DTO แปลงเป็นข้อความพร้อม field
type FormatReason string

const (
	ReasonStepCount       FormatReason = "step_count"       // ไม่ครบ 10 ขั้น
	ReasonMultiplierRange FormatReason = "multiplier_range" // Coin / RPS: 1–1,000 · รางวัล: > 0 และ ≤ 1,000
	ReasonMultiplierOrder FormatReason = "multiplier_order" // ต้องมากกว่าขั้นก่อนหน้า
	ReasonWinRange        FormatReason = "win_range"        // > 0 และ ≤ 100
	ReasonPrizeCount      FormatReason = "prize_count"      // 1–10 แถว
	ReasonShareRange      FormatReason = "share_range"      // > 0 และ ≤ 100
	ReasonPrizeDuplicate  FormatReason = "prize_duplicate"  // ตัวคูณซ้ำ
)

// FormatIssue — รูปแบบที่ผิดตัวแรก · Index = ลำดับใน array (0 = ตัวแรก) · -1 = ทั้ง array
type FormatIssue struct {
	Field  string // "steps" · "multiplier" · "win_percent" · "prizes" · "share_percent"
	Index  int
	Reason FormatReason
}

// CheckStepsFormat — รูปแบบ steps (RATE-10 · RATE-11 · RATE-20 · RATE-21) · ต้องครบ stepCount ขั้นของเกม · ไล่ตามลำดับขั้น · คืนตัวแรกที่ผิด
func CheckStepsFormat(steps []Step, stepCount int) *FormatIssue {
	if len(steps) != stepCount {
		return &FormatIssue{"steps", -1, ReasonStepCount}
	}
	for i, s := range steps {
		if s.Multiplier < MinMultiplierStep || s.Multiplier > MaxMultiplier {
			return &FormatIssue{"multiplier", i, ReasonMultiplierRange}
		}
		if i > 0 && s.Multiplier <= steps[i-1].Multiplier {
			return &FormatIssue{"multiplier", i, ReasonMultiplierOrder}
		}
		if s.WinPercent <= 0 || s.WinPercent > MaxPercent {
			return &FormatIssue{"win_percent", i, ReasonWinRange}
		}
	}
	return nil
}

// CheckScratchFormat — Scratch Card (RATE-30 · RATE-31) · win_percent ก่อน แล้วไล่แถวรางวัล
func CheckScratchFormat(s Scratch) *FormatIssue {
	if s.WinPercent <= 0 || s.WinPercent > MaxPercent {
		return &FormatIssue{"win_percent", -1, ReasonWinRange}
	}
	if len(s.Prizes) < MinPrizes || len(s.Prizes) > MaxPrizes {
		return &FormatIssue{"prizes", -1, ReasonPrizeCount}
	}
	seen := make(map[float64]bool, len(s.Prizes))
	for i, p := range s.Prizes {
		if p.Multiplier <= 0 || p.Multiplier > MaxMultiplier {
			return &FormatIssue{"multiplier", i, ReasonMultiplierRange}
		}
		if seen[p.Multiplier] {
			return &FormatIssue{"multiplier", i, ReasonPrizeDuplicate}
		}
		seen[p.Multiplier] = true
		if p.SharePercent <= 0 || p.SharePercent > MaxPercent {
			return &FormatIssue{"share_percent", i, ReasonShareRange}
		}
	}
	return nil
}
