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

// การรีเซ็ตรหัสผ่าน / passcode ระดับข้อมูลบัญชี (AUTH-46 – AUTH-49) — ใครสั่งรีเซ็ตได้ใคร เป็นกฎของ module admin_management
// (เส้น admin reset และ scripts/reset_credentials) ซึ่งเรียกฟังก์ชันในไฟล์นี้

// ResetTarget — บัญชีที่จะถูกรีเซ็ต · ไม่เปิดข้อมูล credential ออกนอก module
type ResetTarget struct{ acc account }

func (t ResetTarget) Username() string                { return t.acc.Username }
func (t ResetTarget) AccountType() models.AccountType { return t.acc.Type }
func (t ResetTarget) ID() uint                        { return t.acc.ID }
func (t ResetTarget) IsSub() bool                     { return t.acc.IsSub() }
func (t ResetTarget) Role() models.AgentRole          { return t.acc.Role } // sub = role ของผู้สร้าง
func (t ResetTarget) Status() models.AgentStatus      { return t.acc.Status }
func (t ResetTarget) UplineLocked() bool              { return t.acc.UplineLocked() }

// FindResetTargetService — หาบัญชี (ฝั่ง agent หรือ sub) ตาม username พร้อมสถานะหัวสาย · ไม่พบ = ErrResetTargetNotFound
func FindResetTargetService(ctx context.Context, rawUsername string) (ResetTarget, error) {
	username := agentAuthCore.NormalizeUsername(rawUsername)
	if agentAuthCore.IsSubaccountUsername(username) {
		if _, _, ok := agentAuthCore.SplitSubaccountUsername(username); !ok {
			return ResetTarget{}, apperr.ErrResetTargetNotFound
		}
	}
	db := database.DBConn.WithContext(ctx)
	acc, err := loadAccountForLogin(db, username)
	if errors.Is(err, apperr.ErrNotFound) {
		return ResetTarget{}, apperr.ErrResetTargetNotFound
	}
	if err != nil {
		return ResetTarget{}, err
	}
	if acc, err = withUplineStatus(db, acc); err != nil {
		return ResetTarget{}, err
	}
	return ResetTarget{acc: acc}, nil
}

// ResetBy — ผู้สั่งรีเซ็ต: บัญชีที่ login (Actor) หรือ script บนเซิร์ฟเวอร์ (ScriptOperator = user ของเครื่อง)
type ResetBy struct {
	Actor          *Actor
	ScriptOperator string
	Meta           RequestMeta
}

// ResetOptions — รีเซ็ตอะไรบ้าง
type ResetOptions struct {
	Password     bool
	Passcode     bool
	RefuseLocked bool // เป้าหมายถูกล็อก = ErrResetTargetLocked (เส้น admin) · script กู้บัญชีได้แม้ถูกล็อก
}

// ResetResult — ค่าชั่วคราว แสดงครั้งเดียว ห้าม log (AUTH-46)
type ResetResult struct {
	Username      string
	TempPassword  string // "" = ไม่ได้รีเซ็ต
	TempPasscode  string
	TempExpiresAt time.Time
}

// ResetCredentialsService — ตั้งค่าชั่วคราว + บังคับเปลี่ยน + audit ใน transaction เดียว (AUTH-46 – AUTH-49)
// หลัง commit: ล้างบล็อก login (AUTH-10) / passcode (AUTH-35) และตัด session ของเป้าหมาย
func ResetCredentialsService(ctx context.Context, t ResetTarget, opt ResetOptions, by ResetBy) (ResetResult, error) {
	now := time.Now()
	res := ResetResult{Username: t.acc.Username, TempExpiresAt: now.Add(configs.Cfg.Auth.TempCredentialTTL)}
	audit := func(action models.AuthAuditAction, acc account) auditEntry {
		if by.Actor != nil {
			return actorEntry(action, *by.Actor, acc)
		}
		return auditEntry{Action: action, ActorType: models.AuthAuditActorScript, ActorUsername: truncate(by.ScriptOperator, 71), Target: &acc}
	}
	err := database.DBConn.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		acc, err := lockAccountCredentials(tx, t.acc.Type, t.acc.ID)
		if err != nil {
			return err
		}
		if opt.RefuseLocked && acc.Status == models.AgentStatusLocked {
			return apperr.ErrResetTargetLocked
		}
		if opt.Passcode && !acc.PasscodeSet() { // AUTH-47
			return apperr.ErrResetTargetNoPasscode
		}
		if opt.Password {
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
			if err := writeAudit(ctx, tx, audit(models.AuthAuditResetPassword, acc), by.Meta); err != nil {
				return err
			}
		}
		if opt.Passcode {
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
			if err := writeAudit(ctx, tx, audit(models.AuthAuditResetPasscode, acc), by.Meta); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return ResetResult{}, err
	}

	if opt.Password {
		if err := agentAuthRedis.ClearBOLoginBlockRepository(ctx, t.acc.Username); err != nil {
			logger.Ctx(ctx).Warnw("clear login block failed", "username", t.acc.Username, "error", err)
		}
	}
	if opt.Passcode {
		if err := agentAuthRedis.ClearBOPasscodeBlockRepository(ctx, t.acc.Type, t.acc.ID); err != nil {
			logger.Ctx(ctx).Warnw("clear passcode block failed", "account_type", t.acc.Type, "account_id", t.acc.ID, "error", err)
		}
	}
	endAllSessions(ctx, t.acc)
	return res, nil
}

// newTempPassword สุ่มรหัสชั่วคราวที่ผ่าน AUTH-36 และไม่ซ้ำกับรหัสปัจจุบัน/ก่อนหน้า (AUTH-46, AUTH-48)
func newTempPassword(acc account) (string, error) {
	for {
		p, err := utils.RandomString(agentAuthCore.TempPasswordAlphabet, agentAuthCore.TempPasswordLength)
		if err != nil {
			return "", err
		}
		if agentAuthCore.CheckPasswordPolicy(p) == agentAuthCore.PasswordOK && !isRecentPassword(acc, p) {
			return p, nil
		}
	}
}
