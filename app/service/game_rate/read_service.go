package gamerate

import (
	"context"

	gameCore "app/app/core/game"
	gameRateCore "app/app/core/game_rate"
	gameRateDto "app/app/internals/backoffice/dto/game_rate"
	"app/app/models"
	gameRatePostgres "app/app/repository/postgres/game_rate"
	"app/pkg/apperr"
	"app/pkg/utils"
	"app/platform/database"
)

// ListRatesService — RE1 POST /rate/list · ทุกเกมตามลำดับใน core/game · ยังไม่เคยตั้ง = version 0
func ListRatesService(ctx context.Context) ([]gameRateDto.ListRow, error) {
	db := database.DBConn.WithContext(ctx)
	games := gameCore.All()
	codes := make([]string, len(games))
	for i, g := range games {
		codes[i] = g.GameCode
	}
	latest, err := gameRatePostgres.ListLatestVersionsRepository(db, codes)
	if err != nil {
		return nil, err
	}
	rates, err := loadRates(db, latest)
	if err != nil {
		return nil, err
	}
	closed, err := gameRatePostgres.ListClosedGamesRepository(db) // สวิตช์ปิดเกม (RE8)
	if err != nil {
		return nil, err
	}
	isClosed := make(map[string]bool, len(closed))
	for _, code := range closed {
		isClosed[code] = true
	}
	changes, err := latestStatusChanges(db, codes) // เปิด / ปิดครั้งล่าสุด (RE8)
	if err != nil {
		return nil, err
	}
	byCode := make(map[string]models.GameRateVersion, len(latest))
	for _, v := range latest {
		byCode[v.GameCode] = v
	}
	out := make([]gameRateDto.ListRow, len(codes))
	for i, code := range codes {
		row := gameRateDto.ListRow{GameCode: code, StatusGame: !isClosed[code]}
		if ch, ok := changes[code]; ok {
			row.StatusUpdatedAt, row.StatusUpdatedBy = formatTime(ch.CreatedAt), ch.ActorUsername
		}
		if v, ok := byCode[code]; ok {
			row.Version, row.MinHouseEdgePercent = v.Version, minHouseEdge(rates[v.ID])
			row.CreatedAt, row.CreatedBy = formatTime(v.CreatedAt), v.CreatedBy
		}
		out[i] = row
	}
	return out, nil
}

// GetRateService — RE2 POST /rate/detail/get · ค่าปัจจุบัน + ค่าที่คำนวณ
func GetRateService(ctx context.Context, gameCode string) (any, error) {
	spec, ok := gameRateCore.SpecOf(gameCode)
	if !ok {
		return nil, apperr.ErrRateGameNotFound
	}
	db := database.DBConn.WithContext(ctx)
	v, err := gameRatePostgres.GetLatestVersionRepository(db, gameCode)
	if err != nil {
		return nil, err
	}
	rates, err := loadRates(db, []models.GameRateVersion{v})
	if err != nil {
		return nil, err
	}
	return detailView(spec, gameCode, v, rates[v.ID]), nil
}

// CheckRateService — RE3 POST /rate/detail/check (lead RN2 / Q2) · ไม่บันทึก ไม่แตะ DB
// กฎไม่ผ่านยังตอบ 200 พร้อมค่าคำนวณ · valid false + issues ทุกจุด
func CheckRateService(req gameRateDto.CheckRequest) any {
	r := Rates{GameCode: req.GameCode, Kind: req.Spec.Kind, Steps: req.Values, Scratch: req.Scratch}
	if req.Spec.Kind == gameRateCore.KindSteps {
		calc, issues := stepsCalc(req.Values)
		return gameRateDto.StepsCheck{GameCode: req.GameCode, Valid: len(issues) == 0, Issues: issueViews(issues, r), StepsCalc: calc}
	}
	calc, issues := prizesCalc(req.Scratch)
	return gameRateDto.PrizesCheck{GameCode: req.GameCode, Valid: len(issues) == 0, Issues: issueViews(issues, r), PrizesCalc: calc}
}

// GetVersionService — RE6 POST /rate/versions/detail/get · ไม่พบ = 406402
func GetVersionService(ctx context.Context, gameCode string, version int) (any, error) {
	spec, ok := gameRateCore.SpecOf(gameCode)
	if !ok {
		return nil, apperr.ErrRateGameNotFound
	}
	db := database.DBConn.WithContext(ctx)
	v, err := gameRatePostgres.GetVersionRepository(db, gameCode, version)
	if err != nil {
		return nil, err
	}
	if v.ID == 0 {
		return nil, apperr.ErrRateVersionNotFound
	}
	rates, err := loadRates(db, []models.GameRateVersion{v})
	if err != nil {
		return nil, err
	}
	return detailView(spec, gameCode, v, rates[v.ID]), nil
}

// ListVersionsService — RE5 POST /rate/versions/list · ใหม่ → เก่า · changes เทียบกับเวอร์ชันก่อนหน้า · same_as_current (lead Q4)
// โหลดค่าของทุกแถวในหน้า + เวอร์ชันก่อนแถวสุดท้าย + เวอร์ชันปัจจุบันแบบ batch (ไม่ N+1)
func ListVersionsService(ctx context.Context, gameCode string, page utils.Page) ([]gameRateDto.VersionRow, int64, error) {
	spec, ok := gameRateCore.SpecOf(gameCode)
	if !ok {
		return nil, 0, apperr.ErrRateGameNotFound
	}
	db := database.DBConn.WithContext(ctx)
	rows, total, err := gameRatePostgres.ListVersionsRepository(db, gameCode, page.Offset(), page.Limit)
	if err != nil || len(rows) == 0 {
		return []gameRateDto.VersionRow{}, total, err
	}
	all := append([]models.GameRateVersion{}, rows...)
	var prevOfLast models.GameRateVersion
	if last := rows[len(rows)-1].Version; last > 1 {
		if prevOfLast, err = gameRatePostgres.GetVersionRepository(db, gameCode, last-1); err != nil {
			return nil, 0, err
		}
		all = append(all, prevOfLast)
	}
	current, err := gameRatePostgres.GetLatestVersionRepository(db, gameCode)
	if err != nil {
		return nil, 0, err
	}
	all = append(all, current)
	rates, err := loadRates(db, all)
	if err != nil {
		return nil, 0, err
	}

	out := make([]gameRateDto.VersionRow, len(rows))
	for i, v := range rows {
		cur := rates[v.ID]
		var prev *Rates // เวอร์ชันก่อนหน้า = แถวถัดไปในหน้า หรือ prevOfLast · เวอร์ชันแรก = nil
		if i+1 < len(rows) {
			p := rates[rows[i+1].ID]
			prev = &p
		} else if prevOfLast.ID != 0 {
			p := rates[prevOfLast.ID]
			prev = &p
		}
		row := gameRateDto.VersionRow{Version: v.Version, MinHouseEdgePercent: minHouseEdge(cur), RollbackFrom: v.RollbackFrom,
			SameAsCurrent: sameRates(spec.Kind, cur, rates[current.ID]), Changes: changeViews(spec.Kind, prev, cur),
			CreatedAt: formatTime(v.CreatedAt), CreatedBy: v.CreatedBy, IP: v.IP}
		if spec.Kind == gameRateCore.KindPrizes {
			w := cur.Scratch.WinPercent
			row.WinPercent = &w
		}
		out[i] = row
	}
	return out, total, nil
}

// changeViews — สิ่งที่เปลี่ยนเทียบกับเวอร์ชันก่อนหน้า (core DiffSteps / DiffScratch) · เวอร์ชันแรก = []
func changeViews(kind gameRateCore.Kind, prev *Rates, cur Rates) []gameRateDto.ChangeView {
	var changes []gameRateCore.Change
	if kind == gameRateCore.KindSteps {
		var p []gameRateCore.Step
		if prev != nil {
			p = prev.Steps
		}
		changes = gameRateCore.DiffSteps(p, cur.Steps)
	} else {
		var p *gameRateCore.Scratch
		if prev != nil {
			p = &prev.Scratch
		}
		changes = gameRateCore.DiffScratch(p, cur.Scratch)
	}
	out := make([]gameRateDto.ChangeView, len(changes))
	for i, c := range changes {
		out[i] = gameRateDto.ChangeView{Type: string(c.Type), Step: c.Step, Field: c.Field, Multiplier: c.Multiplier, From: c.From, To: c.To}
	}
	return out
}
