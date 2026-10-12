package gamerate

import (
	"context"
	"time"

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
		var ids []uint
		var logs []models.AccountChangeLog
		for _, r := range rows {
			if r.StatusGame == want {
				continue
			}
			ids = append(ids, r.AgentID)
			l, err := agentManagementService.ChangeLog(actor, meta, logger.RequestID(ctx), agentManagementService.TargetAgent, r.AgentID, r.Username,
				models.ChangeUpdateGames, map[string]any{"status_game": map[string]bool{req.GameCode: r.StatusGame}},
				map[string]any{"status_game": map[string]bool{req.GameCode: want}}, now)
			if err != nil {
				return err
			}
			logs = append(logs, l)
		}
		if len(ids) == 0 {
			return nil
		}
		if err := agentManagementPostgres.UpdateStatusGameManyRepository(tx, ids, []string{req.GameCode}, want); err != nil {
			return err
		}
		res.Changed = true
		return agentManagementPostgres.CreateAccountChangeLogsRepository(tx, logs)
	})
	return res, err
}
