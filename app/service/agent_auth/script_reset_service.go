package agentauth

import (
	"context"
	"errors"
	"time"

	agentAuthCore "app/app/core/agent_auth"
	"app/app/models"
	agentAuthRedis "app/app/repository/redis/agent_auth"
	"app/pkg/apperr"
	"app/pkg/configs"
	"app/pkg/utils"
	"app/platform/database"
	"app/platform/logger"

	"gorm.io/gorm"
)

// ScriptResetRequest — สิ่งที่ scripts/reset_credentials ส่งมา (AUTH-51)
type ScriptResetRequest struct {
	Username string
	Password bool   // รีเซ็ตรหัสผ่าน
	Passcode bool   // รีเซ็ต passcode
	Operator string // ชื่อผู้รัน script (user ของเครื่อง) — เก็บเป็น actor_username
}

// ScriptResetResult — ค่าชั่วคราว แสดงครั้งเดียว ห้าม log (AUTH-46)
type ScriptResetResult struct {
	Username      string
	TempPassword  string // "" = ไม่ได้รีเซ็ต
	TempPasscode  string
	TempExpiresAt time.Time
}

// ScriptResetCredentialsService — รีเซ็ตรหัสผ่าน / passcode ของ SUPERADMIN หรือ ADMIN จาก script บนเซิร์ฟเวอร์ (AUTH-51)
// กติกาเดียวกับ admin reset (AUTH-46–AUTH-48) · audit actor = SCRIPT · รีเซ็ตทั้งสองอย่างได้ใน transaction เดียว
func ScriptResetCredentialsService(ctx context.Context, req ScriptResetRequest) (ScriptResetResult, error) {
	var res ScriptResetResult
	if !req.Password && !req.Passcode {
		return res, apperr.ErrValidation.WithMessage("ต้องเลือกรีเซ็ตรหัสผ่านหรือ passcode อย่างน้อย 1 อย่าง",
			"choose to reset the password, the passcode, or both")
	}
	username := agentAuthCore.NormalizeUsername(req.Username)
	db := database.DBConn.WithContext(ctx)
	target, err := loadAccountForLogin(db, username)
	if errors.Is(err, apperr.ErrNotFound) {
		return res, apperr.ErrResetTargetNotFound
	}
	if err != nil {
		return res, err
	}
	if target.IsSub() || (target.Role != models.AgentRoleSuperAdmin && target.Role != models.AgentRoleAdmin) {
		return res, apperr.ErrResetNotAllowed // บัญชีอื่นใช้ API admin reset
	}

	now := time.Now()
	res = ScriptResetResult{Username: target.Username, TempExpiresAt: now.Add(configs.Cfg.Auth.TempCredentialTTL)}
	audit := func(action models.AuthAuditAction, acc account) auditEntry {
		return auditEntry{Action: action, ActorType: models.AuthAuditActorScript, ActorUsername: truncate(req.Operator, 71), Target: &acc}
	}
	meta := RequestMeta{UserAgent: "scripts/reset_credentials"}
	err = database.DBConn.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		acc, err := lockAccountCredentials(tx, target.Type, target.ID)
		if err != nil {
			return err
		}
		if req.Passcode && !acc.PasscodeSet() { // AUTH-47
			return apperr.ErrResetTargetNoPasscode
		}
		if req.Password {
			if res.TempPassword, err = newTempPassword(acc); err != nil {
				return err
			}
			hash, err := utils.HashPassword(res.TempPassword)
			if err != nil {
				return err
			}
			if err := updatePassword(tx, acc, hash, true, &res.TempExpiresAt, now); err != nil {
				return err
			}
			if err := writeAudit(ctx, tx, audit(models.AuthAuditResetPassword, acc), meta); err != nil {
				return err
			}
		}
		if req.Passcode {
			for res.TempPasscode == "" || utils.CheckPassword(*acc.PasscodeHash, res.TempPasscode) { // ห้ามซ้ำกับตัวปัจจุบัน
				if res.TempPasscode, err = utils.RandomString("0123456789", agentAuthCore.PasscodeLength); err != nil {
					return err
				}
			}
			hash, err := utils.HashPassword(res.TempPasscode)
			if err != nil {
				return err
			}
			if err := updatePasscode(tx, acc, hash, true, &res.TempExpiresAt, now); err != nil {
				return err
			}
			if err := writeAudit(ctx, tx, audit(models.AuthAuditResetPasscode, acc), meta); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return ScriptResetResult{}, err
	}

	// หลัง commit: ล้างบล็อก (AUTH-10 / AUTH-35) และเตะ session ของเป้าหมาย
	if req.Password {
		if err := agentAuthRedis.ClearBOLoginBlockRepository(ctx, target.Username); err != nil {
			logger.Ctx(ctx).Warnw("clear login block failed", "username", target.Username, "error", err)
		}
	}
	if req.Passcode {
		if err := agentAuthRedis.ClearBOPasscodeBlockRepository(ctx, target.Type, target.ID); err != nil {
			logger.Ctx(ctx).Warnw("clear passcode block failed", "account_type", target.Type, "account_id", target.ID, "error", err)
		}
	}
	endAllSessions(ctx, target)
	return res, nil
}
