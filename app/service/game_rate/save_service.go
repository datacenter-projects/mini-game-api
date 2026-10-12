package gamerate

import (
	"context"
	"errors"
	"fmt"
	"time"

	gameRateCore "app/app/core/game_rate"
	gameRateDto "app/app/internals/backoffice/dto/game_rate"
	"app/app/models"
	gameRatePostgres "app/app/repository/postgres/game_rate"
	agentAuthService "app/app/service/agent_auth"
	"app/pkg/apperr"
	"app/platform/database"
	"app/platform/logger"

	"gorm.io/gorm"
)

// การบันทึก (RE4) และย้อนเวอร์ชัน (RE7) — tx เดียว · advisory lock ต่อเกม → base_version (406301) → กฎ (406101 / 406102)
// → ค่าเท่าปัจจุบัน = ไม่สร้างเวอร์ชัน (created false — RATE-05C) → insert เวอร์ชัน + แถวลูกแบบ batch (RATE-05 · RATE-08)

// staleError — base_version ไม่ตรง · msg บอกเวอร์ชันล่าสุดและคนแก้ (lead Q5 — error ไม่มี data)
func staleError(latest models.GameRateVersion) error {
	if latest.Version == 0 {
		return apperr.ErrRateStale.WithMessage("เกมนี้ยังไม่มีเวอร์ชัน (base_version ต้องเป็น 0) กรุณาโหลดใหม่",
			"this game has no version yet (base_version must be 0), please reload")
	}
	return apperr.ErrRateStale.WithMessage(
		fmt.Sprintf("มีคนแก้ไปแล้ว (v%d โดย %s) กรุณาโหลดใหม่", latest.Version, latest.CreatedBy),
		fmt.Sprintf("rates were changed by someone else (v%d by %s), please reload", latest.Version, latest.CreatedBy))
}

// saveRates — ส่วนร่วมของ RE4 / RE7 ภายใน tx · คืน (เวอร์ชันหลังบันทึก, สร้างใหม่ไหม)
func saveRates(ctx context.Context, tx *gorm.DB, actor agentAuthService.Actor, meta agentAuthService.RequestMeta,
	next Rates, baseVersion, rollbackFrom int) (int, bool, error) {
	if err := gameRatePostgres.LockGameRateRepository(tx, next.GameCode); err != nil {
		return 0, false, err
	}
	latest, err := gameRatePostgres.GetLatestVersionRepository(tx, next.GameCode)
	if err != nil {
		return 0, false, err
	}
	if baseVersion != latest.Version {
		return 0, false, staleError(latest)
	}
	if err := checkFormat(next); err != nil { // รูปแบบตามกฎปัจจุบัน (เพดาน 50 · ตัวคูณ · จำนวนขั้น / แถว) — rollback ต้องผ่านด้วย (RATE-05A)
		return 0, false, err
	}
	if issues := ruleIssues(next); len(issues) > 0 { // กฎปัจจุบัน — rollback ไปเวอร์ชันที่ไม่ผ่านกฎใหม่ก็ไม่ได้ (RATE-05A)
		return 0, false, issueError(issues[0], next)
	}
	if latest.ID != 0 {
		cur, err := loadRates(tx, []models.GameRateVersion{latest})
		if err != nil {
			return 0, false, err
		}
		if sameRates(next.Kind, next, cur[latest.ID]) {
			return latest.Version, false, nil
		}
	}

	v := models.GameRateVersion{GameCode: next.GameCode, Version: latest.Version + 1, ActorType: actor.AccountType, ActorID: actor.AccountID(),
		CreatedBy: actor.Username, IP: meta.IP, RequestID: logger.RequestID(ctx), RollbackFrom: rollbackFrom, CreatedAt: time.Now()}
	if next.Kind == gameRateCore.KindPrizes {
		w := next.Scratch.WinPercent
		v.WinPercent = &w
	}
	if err := tx.SavePoint("rate_version").Error; err != nil {
		return 0, false, err
	}
	if err := gameRatePostgres.CreateVersionRepository(tx, &v); err != nil {
		if !errors.Is(err, apperr.ErrConflict) {
			return 0, false, err
		}
		// เลขชน (หลุด lock) = มีคนบันทึกตัดหน้า · ย้อน savepoint แล้วอ่านเวอร์ชันล่าสุดให้ msg บอกเวอร์ชัน + คนแก้ (Q5)
		if err := tx.RollbackTo("rate_version").Error; err != nil {
			return 0, false, err
		}
		latest, err := gameRatePostgres.GetLatestVersionRepository(tx, next.GameCode)
		if err != nil {
			return 0, false, err
		}
		return 0, false, staleError(latest)
	}
	switch next.Kind {
	case gameRateCore.KindSteps:
		rows := make([]models.GameRateStep, len(next.Steps))
		for i, s := range next.Steps {
			rows[i] = models.GameRateStep{VersionID: v.ID, Step: i + 1, Multiplier: s.Multiplier, WinPercent: s.WinPercent}
		}
		err = gameRatePostgres.CreateStepsRepository(tx, rows)
	case gameRateCore.KindPrizes:
		rows := make([]models.GameRatePrize, len(next.Scratch.Prizes))
		for i, p := range next.Scratch.Prizes {
			rows[i] = models.GameRatePrize{VersionID: v.ID, RowNo: i + 1, Multiplier: p.Multiplier, SharePercent: p.SharePercent}
		}
		err = gameRatePostgres.CreatePrizesRepository(tx, rows)
	}
	return v.Version, true, err
}

// UpdateRateService — RE4 POST /rate/detail/update (RATE-05 · RATE-05B · RATE-05C · RATE-07 · RATE-08)
func UpdateRateService(ctx context.Context, actor agentAuthService.Actor, req gameRateDto.UpdateRequest,
	meta agentAuthService.RequestMeta) (gameRateDto.SaveResponse, error) {
	res := gameRateDto.SaveResponse{GameCode: req.GameCode}
	next := Rates{GameCode: req.GameCode, Kind: req.Spec.Kind, Steps: req.Values, Scratch: req.Scratch}
	err := database.DBConn.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var err error
		res.Version, res.Created, err = saveRates(ctx, tx, actor, meta, next, *req.BaseVersion, 0)
		return err
	})
	return res, err
}

// RollbackRateService — RE7 POST /rate/versions/rollback (RATE-05A) · คัดลอกค่าของเวอร์ชันนั้นเป็นเวอร์ชันใหม่
// ลำดับ: เวอร์ชันมีจริง (406402) → base_version → กฎปัจจุบัน → ค่าเท่าปัจจุบัน (created false · rollback_from 0)
func RollbackRateService(ctx context.Context, actor agentAuthService.Actor, req gameRateDto.RollbackRequest,
	meta agentAuthService.RequestMeta) (gameRateDto.RollbackResponse, error) {
	res := gameRateDto.RollbackResponse{GameCode: req.GameCode}
	err := database.DBConn.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		target, err := gameRatePostgres.GetVersionRepository(tx, req.GameCode, req.Version)
		if err != nil {
			return err
		}
		if target.ID == 0 {
			return apperr.ErrRateVersionNotFound
		}
		rates, err := loadRates(tx, []models.GameRateVersion{target})
		if err != nil {
			return err
		}
		res.Version, res.Created, err = saveRates(ctx, tx, actor, meta, rates[target.ID], *req.BaseVersion, req.Version)
		if res.Created {
			res.RollbackFrom = req.Version
		}
		return err
	})
	return res, err
}

// checkFormat — รูปแบบของค่าชุดนี้ตามกฎปัจจุบัน (core CheckStepsFormat / CheckScratchFormat) · ข้อความเดียวกับ DTO
// RE4 ผ่าน DTO มาแล้ว (ตรวจซ้ำได้ไม่ผิด) · RE7 ค่ามาจาก DB จึงต้องตรวจที่นี่
func checkFormat(r Rates) error {
	spec, ok := gameRateCore.SpecOf(r.GameCode)
	if !ok {
		return apperr.ErrRateGameNotFound
	}
	if spec.Kind == gameRateCore.KindSteps {
		return gameRateDto.FormatError(gameRateCore.CheckStepsFormat(r.Steps, spec.StepCount), "steps", spec.StepCount)
	}
	return gameRateDto.FormatError(gameRateCore.CheckScratchFormat(r.Scratch), "prizes", 0)
}
