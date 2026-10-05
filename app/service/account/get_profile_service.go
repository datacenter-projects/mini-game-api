// Package account คือ business logic ของหน้าบัญชีหลังบ้าน — spec: docs/modules/account.md
package account

import (
	"context"
	"errors"

	accountDto "app/app/internals/backoffice/dto/account"
	"app/app/models"
	"app/app/repository/postgres"
	agentAuthService "app/app/service/agent_auth"
	"app/pkg/apperr"
	"app/platform/database"
)

// GetProfileService — GET /api/v1/bo/pr/account/profile · rule: ACC-11, ACC-12, ACC-14
// สถานะ / role / passcode มาจาก Actor ที่ middleware โหลดแล้ว ส่วนที่เหลืออ่านจากแถวของบัญชีตัวเอง
func GetProfileService(ctx context.Context, actor agentAuthService.Actor) (accountDto.ProfileResponse, error) {
	res := accountDto.ProfileResponse{
		Username:        actor.Username,
		Role:            string(actor.Role),
		Status:          string(actor.Status),
		EffectiveStatus: string(actor.EffectiveStatus),
		IsSubaccount:    actor.AccountType == models.AccountTypeSub,
		PasscodeSet:     actor.PasscodeSet,
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
		res.OwnerUsername = &owner.Username
		res.LastLoginAt, res.LastLoginIP, res.CreatedAt = sub.LastLoginAt, sub.LastLoginIP, sub.CreatedAt
		return res, nil
	}

	agent, err := postgres.GetUserAgentProfileRepository(db, actor.AgentID)
	if err != nil {
		return res, notFoundAsSessionEnded(err)
	}
	res.LastLoginAt, res.LastLoginIP, res.CreatedAt = agent.LastLoginAt, agent.LastLoginIP, agent.CreatedAt
	return res, nil
}

// บัญชีหายไประหว่าง request (middleware เพิ่งโหลดได้) ถือว่า session สิ้นสุด แบบเดียวกับ AuthenticateService
func notFoundAsSessionEnded(err error) error {
	if errors.Is(err, apperr.ErrNotFound) {
		return apperr.ErrSessionEnded
	}
	return err
}
