package gamerate

import (
	"context"
	"time"

	agentManagementCore "app/app/core/agent_management"
	gameCore "app/app/core/game"
	gameRateDto "app/app/internals/backoffice/dto/game_rate"
	"app/app/models"
	agentManagementPostgres "app/app/repository/postgres/agent_management"
	gameRatePostgres "app/app/repository/postgres/game_rate"
	agentAuthService "app/app/service/agent_auth"
	agentManagementService "app/app/service/agent_management"
	"app/platform/database"
	"app/platform/logger"

	"gorm.io/gorm"
)

// SetGameStatusService — RE8 POST /rate/status/update · สวิตช์ปิด / เปิดเกมทั้งระบบ (RATE-01B · lead Q1 / QS1–QS3)
// ตั้ง status_game ของทุกบัญชี Superadmin ใน tx เดียว · ไม่แก้สายล่าง — ผลไหลลงผ่านการเช็คทั้งสาย (core IsGameOpen)
// Superadmin ที่ยังไม่มีแถวของเกมนั้น = เพิ่มแถวตามที่สั่ง (lead G-S1 — ไม่งั้นสวิตช์ล้มเงียบ) · ไม่มีแถว = เปิดอยู่ (IsGameOpen ไม่นับ)
// ทุกบัญชีเป็นค่านั้นอยู่แล้ว = changed false ไม่เขียน log · log UPDATE_GAMES 1 แถวต่อบัญชีที่ค่าเปลี่ยน
func SetGameStatusService(ctx context.Context, actor agentAuthService.Actor, req gameRateDto.StatusRequest,
	meta agentAuthService.RequestMeta) (gameRateDto.StatusResponse, error) {
	want := *req.StatusGame
	res := gameRateDto.StatusResponse{GameCode: req.GameCode, StatusGame: want}
	err := database.DBConn.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		rows, err := gameRatePostgres.LockSuperadminGameStatusRepository(tx, req.GameCode)
		if err != nil {
			return err
		}
		now := time.Now()
		var update []uint
		var missing []models.AgentGameSetting
		var logs []models.AccountChangeLog
		for _, r := range rows {
			was := r.StatusGame || !r.HasRow
			if !r.HasRow {
				missing = append(missing, superadminGameRow(r, req.GameCode, want, now))
			} else if was != want {
				update = append(update, r.AgentID)
			}
			if was == want {
				continue
			}
			l, err := agentManagementService.ChangeLog(actor, meta, logger.RequestID(ctx), agentManagementService.TargetAgent, r.AgentID, r.Username,
				models.ChangeUpdateGames, map[string]any{"status_game": map[string]bool{req.GameCode: was}},
				map[string]any{"status_game": map[string]bool{req.GameCode: want}}, now)
			if err != nil {
				return err
			}
			logs = append(logs, l)
		}
		if err := agentManagementPostgres.CreateAgentGameSettingsRepository(tx, missing); err != nil {
			return err
		}
		if err := agentManagementPostgres.UpdateStatusGameManyRepository(tx, update, []string{req.GameCode}, want); err != nil {
			return err
		}
		res.Changed = len(logs) > 0
		return agentManagementPostgres.CreateAccountChangeLogsRepository(tx, logs)
	})
	if err != nil {
		return res, err
	}
	// เวลา + คนที่เปิด / ปิดครั้งล่าสุด (หลัง commit — changed true = ครั้งนี้เอง)
	changes, err := latestStatusChanges(database.DBConn.WithContext(ctx), []string{req.GameCode})
	if err != nil {
		return res, err
	}
	if ch, ok := changes[req.GameCode]; ok {
		res.StatusUpdatedAt, res.StatusUpdatedBy = formatTime(ch.CreatedAt), ch.ActorUsername
	}
	return res, nil
}

// superadminGameRow — แถว agent_game_settings ที่ขาดของ Superadmin · ค่า PT เหมือนตอนสร้าง Superadmin (ได้รับ 100% · รับ PT · ผู้สร้าง = ตัวเอง — MGMT-22)
func superadminGameRow(r gameRatePostgres.SuperadminGameStatus, gameCode string, on bool, now time.Time) models.AgentGameSetting {
	var category string
	for _, g := range gameCore.All() {
		if g.GameCode == gameCode {
			category = g.Category
		}
	}
	return models.AgentGameSetting{AgentID: r.AgentID, GameCode: gameCode, Category: category, PTFromParent: agentManagementCore.FullPT,
		Status: true, StatusGame: on, CreatedBy: r.Username, CreatedAt: now, UpdatedBy: r.Username, UpdatedAt: now}
}

// latestStatusChanges — เปิด / ปิดครั้งล่าสุดต่อเกม (game_code → ครั้งล่าสุด) · เกมที่ไม่เคยเปลี่ยนไม่อยู่ใน map
func latestStatusChanges(db *gorm.DB, gameCodes []string) (map[string]gameRatePostgres.StatusChange, error) {
	rows, err := gameRatePostgres.ListLatestStatusChangesRepository(db, gameCodes)
	if err != nil {
		return nil, err
	}
	out := make(map[string]gameRatePostgres.StatusChange, len(rows))
	for _, r := range rows {
		out[r.GameCode] = r
	}
	return out, nil
}
