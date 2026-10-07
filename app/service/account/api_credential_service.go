package account

import (
	"context"
	"time"

	accountCore "app/app/core/account"
	agentManagementCore "app/app/core/agent_management"
	accountDto "app/app/internals/backoffice/dto/account"
	"app/app/models"
	"app/app/repository/postgres"
	agentAuthService "app/app/service/agent_auth"
	"app/pkg/apperr"
	"app/pkg/configs"
	"app/pkg/utils"
	"app/platform/database"
	"app/platform/logger"

	"gorm.io/gorm"
)

// keyOwner — เจ้าของ Key = บัญชีหลักที่ request ทำงานในนาม (sub = เจ้าของ — ACC-02) · ไม่ใช่ประเภทเจ้าของ = 403301 (ACC-01)
func keyOwner(db *gorm.DB, actor agentAuthService.Actor) (models.UserAgent, error) {
	owner, err := postgres.GetAgentProfileRepository(db, actor.AgentID)
	if err != nil {
		return owner, notFoundAsSessionEnded(err)
	}
	if !accountCore.IsAPIKeyOwner(agentManagementCore.UserTypeOf(owner.Role, owner.AgentType)) {
		return owner, apperr.ErrNoAPICredential
	}
	return owner, nil
}

// GetAPICredentialService — GET /api/v1/bo/pr/account/api-credential (ACC-01 – ACC-05, ACC-10)
// ยังไม่มี Key = สร้างตอนเปิดครั้งแรก (บัญชีที่สร้างก่อนมี MGMT-04)
func GetAPICredentialService(ctx context.Context, actor agentAuthService.Actor) (accountDto.APICredentialResponse, error) {
	var res accountDto.APICredentialResponse
	db := database.DBConn.WithContext(ctx)
	owner, err := keyOwner(db, actor)
	if err != nil {
		return res, err
	}
	if err := createAPICredentialIfAbsent(db, owner.ID); err != nil {
		return res, err
	}
	cred, err := postgres.GetAPICredentialRepository(db, owner.ID)
	if err != nil {
		return res, err
	}
	key, err := utils.DecryptAPIKey(configs.Cfg.Account.APIKeyEncryptionKey, cred.KeyCiphertext)
	if err != nil {
		return res, err
	}
	ips, err := postgres.ListAPIAllowedIPsRepository(db, owner.ID)
	if err != nil {
		return res, err
	}
	return accountDto.APICredentialResponse{Username: owner.Username, Key: key,
		CallbackURL: accountCore.OptionalString(cred.CallbackURL), AllowedIPs: append([]string{}, ips...)}, nil
}

// SaveAPICredentialService — POST /api/v1/bo/pr/account/update-credential (ACC-06 – ACC-09)
// passcode ตรวจแล้วที่ middleware · แทนทั้งชุด (ลิงก์ + IP) ใน tx เดียวกับประวัติ
func SaveAPICredentialService(ctx context.Context, actor agentAuthService.Actor, req accountDto.SaveAPICredentialRequest,
	meta agentAuthService.RequestMeta) error {
	return database.DBConn.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		owner, err := keyOwner(tx, actor)
		if err != nil {
			return err
		}
		if err := createAPICredentialIfAbsent(tx, owner.ID); err != nil {
			return err
		}
		cred, err := postgres.LockAPICredentialRepository(tx, owner.ID)
		if err != nil {
			return err
		}
		oldIPs, err := postgres.ListAPIAllowedIPsRepository(tx, owner.ID)
		if err != nil {
			return err
		}
		var newURL *string
		if req.NormalizedCallbackURL != "" {
			newURL = &req.NormalizedCallbackURL
		}
		if err := postgres.UpdateAPICredentialCallbackRepository(tx, owner.ID, newURL); err != nil {
			return err
		}
		if err := postgres.ReplaceAPIAllowedIPsRepository(tx, owner.ID, req.NormalizedIPs); err != nil {
			return err
		}
		l := models.APICredentialLog{AgentID: owner.ID, ActorType: actor.AccountType, ActorID: actor.AccountID(),
			ActorUsername: actor.Username, OldCallbackURL: cred.CallbackURL, NewCallbackURL: newURL,
			OldIPs: utils.TextArray(append([]string{}, oldIPs...)), NewIPs: utils.TextArray(append([]string{}, req.NormalizedIPs...)),
			CreatedAt: time.Now()}
		if meta.IP != "" {
			l.IP = &meta.IP
		}
		if rid := logger.RequestID(ctx); rid != "" {
			l.RequestID = &rid
		}
		return postgres.CreateAPICredentialLogRepository(tx, &l)
	})
}
