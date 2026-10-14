// Package gamerate คือ business logic ของอัตราแพ้ชนะ (เมนู ⑥) — spec: docs/modules/game_rate.md
// ค่ากลางชุดเดียวทั้งระบบ · กฎ / สูตรอยู่ core/game_rate · ทำงานตามรูปแบบอัตราของเกม (GameSpec) ไม่ผูกกับรหัสเกม
package gamerate

import (
	"fmt"
	"time"

	gameRateCore "app/app/core/game_rate"
	gameRateDto "app/app/internals/backoffice/dto/game_rate"
	"app/app/models"
	gameRatePostgres "app/app/repository/postgres/game_rate"
	"app/pkg/apperr"
	"app/pkg/utils"

	"gorm.io/gorm"
)

// Rates — ค่าของ 1 เวอร์ชัน (ที่ engine / module เดิมพันอ่าน — หัวข้อ 5.8)
type Rates struct {
	VersionID uint
	GameCode  string
	Version   int
	Kind      gameRateCore.Kind
	Steps     []gameRateCore.Step  // KindSteps
	Scratch   gameRateCore.Scratch // KindPrizes
}

// loadRates — ค่าของหลายเวอร์ชันใน 2 query (ขั้น + รางวัล — ไม่ N+1) · key = id ของเวอร์ชัน
func loadRates(db *gorm.DB, versions []models.GameRateVersion) (map[uint]Rates, error) {
	out := make(map[uint]Rates, len(versions))
	ids := make([]uint, 0, len(versions))
	for _, v := range versions {
		if v.ID == 0 {
			continue
		}
		r := Rates{VersionID: v.ID, GameCode: v.GameCode, Version: v.Version, Kind: kindOf(v)}
		if v.WinPercent != nil {
			r.Scratch.WinPercent = *v.WinPercent
		}
		out[v.ID] = r
		ids = append(ids, v.ID)
	}
	steps, err := gameRatePostgres.ListStepsByVersionIDsRepository(db, ids)
	if err != nil {
		return nil, err
	}
	for _, s := range steps { // เรียง version_id, step แล้ว
		r := out[s.VersionID]
		r.Steps = append(r.Steps, gameRateCore.Step{Multiplier: s.Multiplier, WinPercent: s.WinPercent})
		out[s.VersionID] = r
	}
	prizes, err := gameRatePostgres.ListPrizesByVersionIDsRepository(db, ids)
	if err != nil {
		return nil, err
	}
	for _, p := range prizes { // เรียง version_id, row_no แล้ว
		r := out[p.VersionID]
		r.Scratch.Prizes = append(r.Scratch.Prizes, gameRateCore.Prize{Multiplier: p.Multiplier, SharePercent: p.SharePercent})
		out[p.VersionID] = r
	}
	return out, nil
}

// sameRates — ค่าเท่ากันไหม (RATE-05C)
func sameRates(kind gameRateCore.Kind, a, b Rates) bool {
	if kind == gameRateCore.KindSteps {
		return gameRateCore.SameSteps(a.Steps, b.Steps)
	}
	return gameRateCore.SameScratch(a.Scratch, b.Scratch)
}

// stepsCalc — ค่าที่คำนวณของรูปแบบ steps (RE2 / RE3 / RE6) · ยังไม่เคยตั้ง = steps []
func stepsCalc(steps []gameRateCore.Step) (gameRateDto.StepsCalc, []gameRateCore.RuleIssue) {
	out := gameRateDto.StepsCalc{Steps: []gameRateDto.StepView{}}
	if len(steps) == 0 {
		return out, nil
	}
	res, sum, issues := gameRateCore.ComputeSteps(steps)
	for _, r := range res {
		out.Steps = append(out.Steps, gameRateDto.StepView{Step: r.Step, Multiplier: r.Multiplier, WinPercent: r.WinPercent,
			ReachPercent: r.ReachPercent, PayoutPercent: r.PayoutPercent, HouseEdgePercent: r.HouseEdgePercent, MaxWinPercent: r.MaxWinPercent})
	}
	out.MinHouseEdgePercent, out.MinHouseEdgeStep, out.MaxPayoutPercent = sum.MinHouseEdgePercent, sum.MinHouseEdgeStep, sum.MaxPayoutPercent
	return out, issues
}

// prizesCalc — ค่าที่คำนวณของรูปแบบ prizes · ยังไม่เคยตั้ง = prizes []
func prizesCalc(s gameRateCore.Scratch) (gameRateDto.PrizesCalc, []gameRateCore.RuleIssue) {
	out := gameRateDto.PrizesCalc{Prizes: []gameRateDto.PrizeView{}}
	if len(s.Prizes) == 0 {
		return out, nil
	}
	res, issues := gameRateCore.ComputeScratch(s)
	for _, p := range res.Prizes {
		out.Prizes = append(out.Prizes, gameRateDto.PrizeView{RowNo: p.RowNo, Multiplier: p.Multiplier, SharePercent: p.SharePercent,
			PerTicketPercent: p.PerTicketPercent, PayoutPercent: p.PayoutPercent})
	}
	out.WinPercent, out.AvgMultiplier, out.PayoutPercent = res.WinPercent, res.AvgMultiplier, res.PayoutPercent
	out.HouseEdgePercent, out.MaxWinPercent = res.HouseEdgePercent, res.MaxWinPercent
	return out, issues
}

// ruleIssues — กฎของค่าชุดนี้ (RATE-07 · RATE-32) · ทุกจุดที่ไม่ผ่าน
func ruleIssues(r Rates) []gameRateCore.RuleIssue {
	if r.Kind == gameRateCore.KindSteps {
		_, _, issues := gameRateCore.ComputeSteps(r.Steps)
		return issues
	}
	_, issues := gameRateCore.ComputeScratch(r.Scratch)
	return issues
}

// minHouseEdge — house edge ต่ำสุดของเวอร์ชัน (RE1 / RE5)
func minHouseEdge(r Rates) float64 {
	if r.Kind == gameRateCore.KindSteps {
		_, sum, _ := gameRateCore.ComputeSteps(r.Steps)
		return sum.MinHouseEdgePercent
	}
	res, _ := gameRateCore.ComputeScratch(r.Scratch)
	return res.HouseEdgePercent
}

// issueField — path ของ field ใน request (steps[i].win_percent · win_percent · prizes)
func issueField(is gameRateCore.RuleIssue) string {
	if is.Index >= 0 {
		return fmt.Sprintf("steps[%d].%s", is.Index, is.Field)
	}
	return is.Field
}

// issueError — RuleIssue → error ที่ client เห็น (406101 / 406102) พร้อม field และค่าที่ตั้งได้
func issueError(is gameRateCore.RuleIssue, r Rates) *apperr.Error {
	field := issueField(is)
	if is.Code == gameRateCore.RuleShareSum {
		var sum float64
		for _, p := range r.Scratch.Prizes {
			sum = utils.Round4(sum + p.SharePercent)
		}
		s := utils.FormatNum(sum)
		return apperr.ErrRateShareSum.WithMessage("prizes share_percent รวมต้องเท่ากับ 100 (ตอนนี้ "+s+")",
			"prizes share_percent must add up to 100 (now "+s+")")
	}
	m := utils.FormatNum(is.MaxWinPercent)
	return apperr.ErrRateHouseEdge.WithMessage(field+" ตั้งได้ไม่เกิน "+m+" (ร้านต้องได้อย่างน้อย 1%)",
		field+" must be at most "+m+" (house edge must be at least 1%)")
}

// issueViews — RuleIssue ทั้งหมด → รายการของ RE3 (lead Q2)
func issueViews(issues []gameRateCore.RuleIssue, r Rates) []gameRateDto.Issue {
	out := make([]gameRateDto.Issue, 0, len(issues))
	for _, is := range issues {
		e := issueError(is, r)
		out = append(out, gameRateDto.Issue{Field: issueField(is), Code: e.Code, Message: e.MsgTH})
	}
	return out
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format(time.RFC3339Nano)
}

// detailView — RE2 / RE6 ตามรูปแบบของเกม · v.ID 0 = ยังไม่เคยตั้ง
func detailView(spec gameRateCore.GameSpec, gameCode string, v models.GameRateVersion, r Rates) any {
	if spec.Kind == gameRateCore.KindSteps {
		calc, _ := stepsCalc(r.Steps)
		return gameRateDto.StepsDetail{GameCode: gameCode, Version: v.Version, StepsCalc: calc, CreatedAt: formatTime(v.CreatedAt), CreatedBy: v.CreatedBy}
	}
	calc, _ := prizesCalc(r.Scratch)
	return gameRateDto.PrizesDetail{GameCode: gameCode, Version: v.Version, PrizesCalc: calc, CreatedAt: formatTime(v.CreatedAt), CreatedBy: v.CreatedBy}
}

// kindOf — รูปแบบอัตราของเวอร์ชัน: ตาม gameSpecs · เกมที่ไม่มีใน gameSpecs แล้ว (ถูกเอาออกจากโค้ด แต่เดิมพันเก่ายังอ้าง)
// ตัดสินจากข้อมูลที่เก็บ: มี win_percent ในแถวเวอร์ชัน = prizes · ไม่มี = steps
func kindOf(v models.GameRateVersion) gameRateCore.Kind {
	if spec, ok := gameRateCore.SpecOf(v.GameCode); ok {
		return spec.Kind
	}
	if v.WinPercent != nil {
		return gameRateCore.KindPrizes
	}
	return gameRateCore.KindSteps
}

// clone — สำเนาที่ไม่แชร์ slice กับต้นฉบับ (ค่าใน cache ห้ามถูกแก้จากผู้เรียก)
func (r Rates) clone() Rates {
	r.Steps = append([]gameRateCore.Step(nil), r.Steps...)
	r.Scratch.Prizes = append([]gameRateCore.Prize(nil), r.Scratch.Prizes...)
	return r
}
