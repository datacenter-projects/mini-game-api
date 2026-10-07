// Package account คือ business logic ของหน้าบัญชีหลังบ้าน — spec: docs/modules/account.md
package account

import (
	"context"
	"errors"
	"time"

	accountCore "app/app/core/account"
	accountDto "app/app/internals/backoffice/dto/account"
	"app/app/models"
	"app/app/repository/postgres"
	agentAuthService "app/app/service/agent_auth"
	"app/pkg/apperr"
	"app/platform/database"
)

// GetProfileService — GET /api/v1/bo/pr/account/profile · rule: ACC-11, ACC-12, ACC-14, ACC-15, ACC-30, ACC-32
// สถานะ / role / passcode มาจาก Actor ที่ middleware โหลดแล้ว ส่วนที่เหลืออ่านจากแถวของบัญชีตัวเอง
// ประเภทย่อย, สกุลเงิน, ยอดเงิน, ค่าหุ้นส่วน และสิทธิ์ ต้องใช้ตารางของ module ② — ส่งค่าว่างไปก่อน
func GetProfileService(ctx context.Context, actor agentAuthService.Actor) (accountDto.ProfileResponse, error) {
	res := accountDto.ProfileResponse{
		Username:     actor.Username,
		Role:         string(actor.Role),
		UserType:     accountCore.UserTypeFromRole(actor.Role),
		Status:       string(actor.EffectiveStatus),
		IsSubaccount: actor.AccountType == models.AccountTypeSub,
		PasscodeSet:  actor.PasscodeSet,
		Currencies:   []string{},
		Balances:     []accountDto.ProfileBalance{},
		PT:           map[string]any{},
		StatusGame:   map[string]bool{},
		Permissions:  map[string]string{},
	}
	db := database.DBConn.WithContext(ctx)

	if res.IsSubaccount {
		sub, err := postgres.GetSubaccountProfileRepository(db, actor.SubaccountID)
		if err != nil {
			return res, notFoundAsSessionEnded(err)
		}
		owner, err := postgres.GetUserAgentAuthByIDRepository(db, sub.AgentID)
		if err != nil {
			return res, notFoundAsSessionEnded(err)
		}
		res.OwnerUsername = owner.Username
		setLogin(&res, sub.LastLoginAt, sub.LastLoginIP, sub.CreatedAt)
		return res, nil
	}

	agent, err := postgres.GetUserAgentProfileRepository(db, actor.AgentID)
	if err != nil {
		return res, notFoundAsSessionEnded(err)
	}
	setLogin(&res, agent.LastLoginAt, agent.LastLoginIP, agent.CreatedAt)
	return res, nil
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
