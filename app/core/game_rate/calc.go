package gamerate

import (
	"math/big"
	"strconv"
)

// สูตรคำนวณ (RATE-13 · RATE-22 · RATE-33 · RATE-34) — คำนวณด้วย big.Rat แล้วค่อยปัด 4 ตำแหน่งตอนคืนค่า (RATE-09)
// max_win ปัด**ลง** 4 ตำแหน่ง: ค่าที่แสดงต้องตั้งได้จริง (ปัดขึ้นแล้วตั้งค่านั้น house edge จะต่ำกว่า 1%)

var (
	ratOne         = big.NewRat(1, 1)
	ratHundred     = big.NewRat(100, 1)
	ratMinEdge     = big.NewRat(1, 100) // 1% ในรูปสัดส่วน (lead RR2)
	ratMaxPayout   = big.NewRat(99, 100)
	ratMaxWin      = big.NewRat(int64(MaxWinPercent), 1) // เพดาน win_percent เป็น %
	ratTenThousand = big.NewInt(10000)
)

// rat — float64 ที่ผ่าน IsDecimal4 แล้ว → เลขแม่นยำ (ผ่านข้อความ 4 ตำแหน่ง ไม่ใช่ค่าฐาน 2 ของ float)
func rat(v float64) *big.Rat {
	r, _ := new(big.Rat).SetString(strconv.FormatFloat(v, 'f', 4, 64))
	return r
}

// pct — สัดส่วน (0–1) จาก % ที่ตั้ง
func pct(v float64) *big.Rat { return new(big.Rat).Quo(rat(v), ratHundred) }

// round4 — ปัดครึ่งขึ้น 4 ตำแหน่ง (เหมือน utils.Round4)
func round4(r *big.Rat) float64 {
	v, _ := strconv.ParseFloat(r.FloatString(4), 64)
	return v
}

// floor4 — ปัดลง 4 ตำแหน่ง (ค่าบวก)
func floor4(r *big.Rat) float64 {
	n := new(big.Int).Mul(r.Num(), ratTenThousand)
	n.Quo(n, r.Denom())
	v, _ := strconv.ParseFloat(new(big.Rat).SetFrac(n, ratTenThousand).FloatString(4), 64)
	return v
}

// percentOf — สัดส่วน → % (×100)
func percentOf(r *big.Rat) *big.Rat { return new(big.Rat).Mul(r, ratHundred) }

// StepResult — ค่าที่คำนวณของ 1 ขั้น (RE2 / RE3)
type StepResult struct {
	Step             int
	Multiplier       float64
	WinPercent       float64
	ReachPercent     float64
	PayoutPercent    float64
	HouseEdgePercent float64
	MaxWinPercent    float64
}

// StepsSummary — สรุปทั้งเกมของ Coin / RPS (กล่องสรุปของหน้าบ้าน — Q8)
type StepsSummary struct {
	MinHouseEdgePercent float64
	MinHouseEdgeStep    int // ขั้นที่ร้านได้น้อยสุด · ค่าเท่ากันเอาขั้นแรก
	MaxPayoutPercent    float64
}

// RuleCode — กฎที่ไม่ผ่าน (406101 / 406102) · DTO / service แปลงเป็น error
type RuleCode string

const (
	RuleHouseEdge RuleCode = "house_edge" // 406101
	RuleShareSum  RuleCode = "share_sum"  // 406102
)

// RuleIssue — จุดที่ไม่ผ่านกฎ · Index = ลำดับใน array (-1 = ทั้งเกม) · MaxWinPercent = win_percent สูงสุดที่ตั้งได้ (house edge)
type RuleIssue struct {
	Code          RuleCode
	Field         string // "win_percent" · "prizes"
	Index         int
	MaxWinPercent float64
}

// ComputeSteps — Coin / RPS (RATE-13 · RATE-22 — สูตรเดียวกัน) · ทุกขั้นต้องได้ house edge ≥ 1% (RATE-07)
// reach_n = Π win_i · payout_n = reach_n × multiplier_n · house_edge_n = 100% − payout_n
// max_win_n = 99% ÷ (reach_{n−1} × multiplier_n) ไม่เกิน 50 (เพดาน win_percent) · คืน issue ทุกขั้นที่ไม่ผ่าน (RE3 แสดงครบ)
func ComputeSteps(steps []Step) ([]StepResult, StepsSummary, []RuleIssue) {
	out := make([]StepResult, len(steps))
	var issues []RuleIssue
	var sum StepsSummary
	var minEdge *big.Rat
	reachPrev := new(big.Rat).Set(ratOne)
	for i, s := range steps {
		mult := rat(s.Multiplier)
		reach := new(big.Rat).Mul(reachPrev, pct(s.WinPercent))
		payout := new(big.Rat).Mul(reach, mult)
		edge := new(big.Rat).Sub(ratOne, payout)
		maxWin := percentOf(new(big.Rat).Quo(ratMaxPayout, new(big.Rat).Mul(reachPrev, mult)))
		if maxWin.Cmp(ratMaxWin) > 0 { // ไม่เกินเพดาน win_percent (50)
			maxWin = new(big.Rat).Set(ratMaxWin)
		}
		out[i] = StepResult{Step: i + 1, Multiplier: s.Multiplier, WinPercent: s.WinPercent,
			ReachPercent: round4(percentOf(reach)), PayoutPercent: round4(percentOf(payout)),
			HouseEdgePercent: round4(percentOf(edge)), MaxWinPercent: floor4(maxWin)}
		if edge.Cmp(ratMinEdge) < 0 {
			issues = append(issues, RuleIssue{Code: RuleHouseEdge, Field: "win_percent", Index: i, MaxWinPercent: floor4(maxWin)})
		}
		if minEdge == nil || edge.Cmp(minEdge) < 0 {
			minEdge = edge
			sum.MinHouseEdgeStep = i + 1
		}
		reachPrev = reach
	}
	if minEdge != nil {
		sum.MinHouseEdgePercent = round4(percentOf(minEdge))
		sum.MaxPayoutPercent = round4(percentOf(new(big.Rat).Sub(ratOne, minEdge)))
	}
	return out, sum, issues
}

// PrizeResult — 1 แถวรางวัลพร้อม % ต่อใบ (RATE-34)
type PrizeResult struct {
	RowNo            int
	Multiplier       float64
	SharePercent     float64
	PerTicketPercent float64 // โอกาสที่ 1 ใบได้รางวัลแถวนี้ = W × share
	PayoutPercent    float64 // ผลตอบแทนที่แถวนี้จ่ายคืน = W × share × multiplier · ทุกแถวรวม ≈ PayoutPercent ของใบ
}

// ScratchResult — ค่าที่คำนวณของ Scratch Card (RATE-33 · RATE-34)
type ScratchResult struct {
	WinPercent       float64
	Prizes           []PrizeResult
	AvgMultiplier    float64
	PayoutPercent    float64
	HouseEdgePercent float64
	MaxWinPercent    float64
}

// ComputeScratch — A = Σ(multiplier × share ÷ 100) · P = W × A · house_edge = 100% − P ≥ 1% · max_win = 99% ÷ A ไม่เกิน 50
// share รวมต้อง = 100 พอดี (RATE-32 · ตรวจก่อน house edge) · คืน issue ทุกข้อที่ไม่ผ่าน
func ComputeScratch(s Scratch) (ScratchResult, []RuleIssue) {
	res := ScratchResult{WinPercent: s.WinPercent, Prizes: make([]PrizeResult, len(s.Prizes))}
	var issues []RuleIssue
	w := pct(s.WinPercent)
	avg := new(big.Rat)
	shareSum := new(big.Rat)
	for i, p := range s.Prizes {
		share := pct(p.SharePercent)
		avg.Add(avg, new(big.Rat).Mul(rat(p.Multiplier), share))
		shareSum.Add(shareSum, share)
		res.Prizes[i] = PrizeResult{RowNo: i + 1, Multiplier: p.Multiplier, SharePercent: p.SharePercent,
			PerTicketPercent: round4(percentOf(new(big.Rat).Mul(w, share))),
			PayoutPercent:    round4(percentOf(new(big.Rat).Mul(new(big.Rat).Mul(w, share), rat(p.Multiplier))))}
	}
	if shareSum.Cmp(ratOne) != 0 {
		issues = append(issues, RuleIssue{Code: RuleShareSum, Field: "prizes", Index: -1})
	}
	payout := new(big.Rat).Mul(w, avg)
	edge := new(big.Rat).Sub(ratOne, payout)
	res.AvgMultiplier = round4(avg)
	res.PayoutPercent = round4(percentOf(payout))
	res.HouseEdgePercent = round4(percentOf(edge))
	if avg.Sign() > 0 {
		maxWin := percentOf(new(big.Rat).Quo(ratMaxPayout, avg))
		if maxWin.Cmp(ratMaxWin) > 0 { // ไม่เกินเพดาน win_percent (50)
			maxWin = new(big.Rat).Set(ratMaxWin)
		}
		res.MaxWinPercent = floor4(maxWin)
	}
	if edge.Cmp(ratMinEdge) < 0 {
		issues = append(issues, RuleIssue{Code: RuleHouseEdge, Field: "win_percent", Index: -1, MaxWinPercent: res.MaxWinPercent})
	}
	return res, issues
}
