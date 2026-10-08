// Package account คือ business logic ของหน้าบัญชีหลังบ้าน — spec: docs/modules/account.md
package account

import (
	"context"
	"errors"
	"time"

	accountCore "app/app/core/account"
	agentManagementCore "app/app/core/agent_management"
	accountDto "app/app/internals/backoffice/dto/account"
	"app/app/models"
	"app/app/repository/postgres"
	agentAuthService "app/app/service/agent_auth"
	agentManagementService "app/app/service/agent_management"
	"app/pkg/apperr"
	"app/platform/database"

	"gorm.io/gorm"
)

// GetProfileService — GET /api/v1/bo/pr/account/profile · rule: ACC-11 – ACC-19, ACC-30, ACC-32
// สถานะ / role / passcode มาจาก Actor ที่ middleware โหลดแล้ว · sub ได้ประเภท สกุล ยอด และค่าหุ้นส่วนของผู้สร้าง (ACC-15)
func GetProfileService(ctx context.Context, actor agentAuthService.Actor) (accountDto.ProfileResponse, error) {
	res := accountDto.ProfileResponse{
		Username:     actor.Username,
		Role:         string(actor.Role),
		Status:       string(actor.EffectiveStatus),
		IsSubaccount: actor.AccountType == models.AccountTypeSub,
		PasscodeSet:  actor.PasscodeSet,
	}
	db := database.DBConn.WithContext(ctx)

	owner, err := postgres.GetUserAgentProfileRepository(db, actor.AgentID)
	if err != nil {
		return res, notFoundAsSessionEnded(err)
	}
	res.UserType = string(agentManagementCore.UserTypeOf(owner.Role, owner.AgentType))
	if res.IsSubaccount {
		sub, err := postgres.GetSubaccountProfileRepository(db, actor.SubaccountID)
		if err != nil {
			return res, notFoundAsSessionEnded(err)
		}
		res.OwnerUsername = owner.Username
		setLogin(&res, sub.LastLoginAt, sub.LastLoginIP, sub.CreatedAt)
	} else {
		setLogin(&res, owner.LastLoginAt, owner.LastLoginIP, owner.CreatedAt)
	}

	if err := setAccountData(db, &res, owner.ID); err != nil {
		return res, err
	}
	perms, err := agentManagementService.PermissionsOf(ctx, actor)
	if err != nil {
		return res, err
	}
	if actor.EffectiveStatus == models.AgentStatusSuspended { // ACC-12 · AUTH-54
		perms = accountCore.SuspendedPermissions(perms)
	}
	res.Permissions = make(map[string]string, len(perms))
	for m, l := range perms {
		res.Permissions[string(m)] = string(l)
	}
	return res, nil
}

// setAccountData — สกุล ยอด ค่าหุ้นส่วน และเปิด / ปิดเกม ของบัญชีหลัก (ACC-16, ACC-19)
// ADMIN / บัญชีที่ไม่มีแถว = รายการว่าง · สกุลที่ยังไม่มียอด และบัญชีฝั่ง Seamless = 0
func setAccountData(db *gorm.DB, res *accountDto.ProfileResponse, agentID uint) error {
	currencies, err := postgres.ListAgentCurrenciesRepository(db, agentID)
	if err != nil {
		return err
	}
	balances, err := postgres.ListAgentBalancesByIDsRepository(db, []uint{agentID})
	if err != nil {
		return err
	}
	settings, err := postgres.ListAgentGameSettingsRepository(db, agentID)
	if err != nil {
		return err
	}

	amount := make(map[string]int64, len(balances))
	for _, b := range balances {
		amount[b.Currency] = b.Amount
	}
	res.Currencies = append([]string{}, currencies...)
	res.Balances = agentManagementService.BalanceViews(currencies, amount)
	res.PT, res.StatusGame = agentManagementService.AgentPTViews(settings)
	return nil
}

func setLogin(res *accountDto.ProfileResponse, at *time.Time, ip *string, created time.Time) {
	res.LastLoginAt = accountCore.OptionalTime(at)
	res.LastLoginIP = accountCore.OptionalString(ip)
	res.CreatedAt = accountCore.OptionalTime(&created)
}

// บัญชีหายไประหว่าง request (middleware เพิ่งโหลดได้) ถือว่า session สิ้นสุด แบบเดียวกับ AuthenticateService
func notFoundAsSessionEnded(err error) error {
	if errors.Is(err, apperr.ErrNotFound) {
		return apperr.ErrSessionEnded
	}
	return err
}
