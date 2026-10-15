package gamerate

// เทียบเวอร์ชัน: ค่าเท่ากัน (RATE-05C — บันทึก / ย้อนแล้วไม่สร้างเวอร์ชันใหม่) และสิ่งที่เปลี่ยน (RE5 `changes` — Q4)
// ค่าที่เข้ามาปัด 4 ตำแหน่งแล้ว (ผ่าน IsDecimal4 / อ่านจาก DB) จึงเทียบด้วย == ได้ตรง

// SameSteps — Coin / RPS เท่ากันทุกขั้น
func SameSteps(a, b []Step) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// SameScratch — Scratch เท่ากัน: win_percent และตารางรางวัลตามลำดับแถว
func SameScratch(a, b Scratch) bool {
	if a.WinPercent != b.WinPercent || len(a.Prizes) != len(b.Prizes) {
		return false
	}
	for i := range a.Prizes {
		if a.Prizes[i] != b.Prizes[i] {
			return false
		}
	}
	return true
}

// ChangeType — ชนิดของสิ่งที่เปลี่ยน (RE5)
type ChangeType string

const (
	ChangeStep         ChangeType = "step"          // Coin / RPS · Step + Field
	ChangeWinPercent   ChangeType = "win_percent"   // Scratch
	ChangePrizeAdded   ChangeType = "prize_added"   // Scratch · From = nil
	ChangePrizeRemoved ChangeType = "prize_removed" // Scratch · To = nil
	ChangePrizeShare   ChangeType = "prize_share"   // Scratch · share ของรางวัลเดิมเปลี่ยน
)

// Change — 1 รายการที่เปลี่ยนเทียบกับเวอร์ชันก่อนหน้า · From / To nil = ไม่มีค่า (JSON null)
type Change struct {
	Type       ChangeType
	Step       int     // ChangeStep เท่านั้น (1–10)
	Field      string  // ChangeStep: "multiplier" / "win_percent"
	Multiplier float64 // รางวัล Scratch ที่เปลี่ยน
	From       *float64
	To         *float64
}

func ptr(v float64) *float64 { return &v }

// DiffSteps — ไล่ทีละขั้น multiplier ก่อน win_percent · prev nil (เวอร์ชันแรก) = ไม่มีรายการ
// จำนวนขั้นต่างกัน (เกมเปลี่ยน StepCount): ขั้นที่เพิ่ม From = nil · ขั้นที่หายไป To = nil
func DiffSteps(prev, cur []Step) []Change {
	out := []Change{}
	if prev == nil {
		return out
	}
	stepValue := func(steps []Step, i int, field string) *float64 {
		if i >= len(steps) {
			return nil
		}
		if field == "multiplier" {
			return ptr(steps[i].Multiplier)
		}
		return ptr(steps[i].WinPercent)
	}
	for i := 0; i < max(len(prev), len(cur)); i++ {
		for _, field := range []string{"multiplier", "win_percent"} {
			from, to := stepValue(prev, i, field), stepValue(cur, i, field)
			if from == nil || to == nil || *from != *to {
				out = append(out, Change{Type: ChangeStep, Step: i + 1, Field: field, From: from, To: to})
			}
		}
	}
	return out
}

// DiffScratch — win_percent ก่อน · แล้วรางวัลจับคู่ด้วย multiplier (ห้ามซ้ำ — RATE-31): ตามลำดับเวอร์ชันใหม่ (เพิ่ม / share เปลี่ยน)
// แล้วรางวัลที่หายไปตามลำดับเวอร์ชันเก่า · prev nil (เวอร์ชันแรก) = ไม่มีรายการ
func DiffScratch(prev *Scratch, cur Scratch) []Change {
	out := []Change{}
	if prev == nil {
		return out
	}
	if prev.WinPercent != cur.WinPercent {
		out = append(out, Change{Type: ChangeWinPercent, From: ptr(prev.WinPercent), To: ptr(cur.WinPercent)})
	}
	old := make(map[float64]float64, len(prev.Prizes))
	for _, p := range prev.Prizes {
		old[p.Multiplier] = p.SharePercent
	}
	kept := make(map[float64]bool, len(cur.Prizes))
	for _, p := range cur.Prizes {
		share, ok := old[p.Multiplier]
		switch {
		case !ok:
			out = append(out, Change{Type: ChangePrizeAdded, Multiplier: p.Multiplier, To: ptr(p.SharePercent)})
		case share != p.SharePercent:
			out = append(out, Change{Type: ChangePrizeShare, Multiplier: p.Multiplier, From: ptr(share), To: ptr(p.SharePercent)})
		}
		kept[p.Multiplier] = true
	}
	for _, p := range prev.Prizes {
		if !kept[p.Multiplier] {
			out = append(out, Change{Type: ChangePrizeRemoved, Multiplier: p.Multiplier, From: ptr(p.SharePercent)})
		}
	}
	return out
}
