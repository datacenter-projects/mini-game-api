package agentmanagement

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	agentAuthCore "app/app/core/agent_auth"
	agentManagementCore "app/app/core/agent_management"
	agentManagementDto "app/app/internals/backoffice/dto/agent_management"
	"app/app/models"
	agentAuthPostgres "app/app/repository/postgres/agent_auth"
	agentManagementPostgres "app/app/repository/postgres/agent_management"
	agentAuthService "app/app/service/agent_auth"
	"app/pkg/apperr"
	"app/pkg/utils"
	"app/platform/database"

	"gorm.io/gorm"
)

// sub (phase 5) — spec: docs/modules/agent_management.md MGMT-40 – MGMT-46
// เส้นทั้งหมดผ่าน middleware RequireMainAccount แล้ว (sub เรียก = 402311)

const targetSub = "SUB"

// CheckMainAccountService — ใช้โดย middleware RequireMainAccount (MGMT-40)
func CheckMainAccountService(actor agentAuthService.Actor) error {
	if actor.AccountType == models.AccountTypeSub {
		return apperr.ErrSubaccountNotAllowed
	}
	return nil
}

// subStatus — สถานะที่แสดง (MGMT-43): เจ้าของ / หัวสายถูกระงับ / ล็อก = สถานะนั้น · sub ถูกตั้ง SUSPENDED = INACTIVE
func subStatus(own, ownerChain models.AgentStatus) string {
	if ownerChain != models.AgentStatusActive {
		return string(agentAuthCore.WorstStatus(ownerChain, own))
	}
	if own == models.AgentStatusSuspended {
		return agentManagementDto.SubStatusInactive
	}
	return string(own)
}

func subView(s models.Subaccount, ownerRole models.AgentRole, ownerChain models.AgentStatus) (agentManagementDto.SubView, error) {
	stored := map[string]string{}
	if s.Permissions != "" {
		if err := json.Unmarshal([]byte(s.Permissions), &stored); err != nil {
			return agentManagementDto.SubView{}, apperr.ErrInternal.Wrap(err)
		}
	}
	perms := agentManagementCore.SubPermissionsView(ownerRole, stored)
	out := make(map[string]string, len(perms))
	for m, l := range perms {
		out[string(m)] = string(l)
	}
	return agentManagementDto.SubView{ID: s.ID, Username: s.Username, Name: StringOrEmpty(s.Name), Phone: StringOrEmpty(s.Phone),
		Status: subStatus(s.Status, ownerChain), Permissions: out, CreatedAt: OptionalTime(&s.CreatedAt),
		LastLoginAt: OptionalTime(s.LastLoginAt), LastLoginIP: StringOrEmpty(s.LastLoginIP)}, nil
}

// subOwner — เจ้าของที่ผู้เรียกดู sub ได้: ตัวเอง หรือบัญชีในสายล่าง (MGMT-45, MGMT-46) · อื่น = notFound
func subOwner(db *gorm.DB, actor agentAuthService.Actor, ownerID uint, notFound error) (models.UserAgent, models.AgentStatus, error) {
	if ownerID != actor.AgentID {
		ok, err := agentManagementPostgres.IsInDownlineRepository(db, actor.AgentID, ownerID)
		if err != nil {
			return models.UserAgent{}, "", err
		}
		if !ok {
			return models.UserAgent{}, "", notFound
		}
	}
	owner, err := agentManagementPostgres.GetAgentProfileRepository(db, ownerID)
	if err != nil {
		return owner, "", notFoundAs(err, notFound)
	}
	chain, err := ChainStatus(db, owner.ID, owner.Status)
	return owner, chain, err
}

// ListSubaccountsService — POST /manage/subaccounts/list (MGMT-46)
func ListSubaccountsService(ctx context.Context, actor agentAuthService.Actor, req agentManagementDto.SubListRequest,
	page utils.Page) ([]agentManagementDto.SubView, int64, error) {
	db := database.DBConn.WithContext(ctx)
	ownerID := req.OwnerID
	if ownerID == 0 {
		ownerID = actor.AgentID
	}
	owner, chain, err := subOwner(db, actor, ownerID, apperr.ErrDownlineNotFound)
	if err != nil {
		return nil, 0, err
	}
	rows, total, err := agentManagementPostgres.ListSubaccountsRepository(db, owner.ID, req.Q, page.Offset(), page.Limit)
	if err != nil {
		return nil, 0, err
	}
	out := make([]agentManagementDto.SubView, len(rows))
	for i, s := range rows {
		if out[i], err = subView(s, owner.Role, chain); err != nil {
			return nil, 0, err
		}
	}
	return out, total, nil
}

// GetSubaccountService — POST /manage/subaccounts/detail (MGMT-46) · sub ของตัวเองหรือของสายล่าง · อื่น = 402404
func GetSubaccountService(ctx context.Context, actor agentAuthService.Actor, id uint) (agentManagementDto.SubView, error) {
	db := database.DBConn.WithContext(ctx)
	s, err := agentManagementPostgres.GetSubaccountViewRepository(db, id)
	if err != nil {
		return agentManagementDto.SubView{}, err
	}
	if s.ID == 0 {
		return agentManagementDto.SubView{}, apperr.ErrSubaccountNotFound
	}
	owner, chain, err := subOwner(db, actor, s.AgentID, apperr.ErrSubaccountNotFound)
	if err != nil {
		return agentManagementDto.SubView{}, err
	}
	return subView(s, owner.Role, chain)
}

// CreateSubaccountService — POST /manage/subaccounts/create (MGMT-40, MGMT-41)
func CreateSubaccountService(ctx context.Context, actor agentAuthService.Actor, req agentManagementDto.SubCreateRequest,
	meta agentAuthService.RequestMeta) (agentManagementDto.SubCreateResponse, error) {
	var res agentManagementDto.SubCreateResponse
	if actor.Role == models.AgentRoleAdmin { // AUTH-43
		return res, apperr.ErrCannotCreateType
	}
	perms, v, ok := agentManagementCore.NormalizeSubPermissions(actor.Role, req.Permissions)
	if !ok {
		return res, permissionError(v)
	}
	permsJSON, err := json.Marshal(perms)
	if err != nil {
		return res, err
	}
	hash, err := utils.HashPassword(req.Password)
	if err != nil {
		return res, err
	}
	err = database.DBConn.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		username := actor.Username + "@" + req.NameSuffix // actor เป็นบัญชีหลัก = username ของเจ้าของ
		if err := agentManagementPostgres.AdvisoryXactLockRepository(tx, "username:"+username); err != nil {
			return err
		}
		taken, err := agentManagementPostgres.SubaccountUsernameExistsRepository(tx, username)
		if err != nil {
			return err
		}
		if taken {
			return apperr.ErrUsernameTaken
		}
		now := time.Now()
		name := req.Name
		s := models.Subaccount{AgentID: actor.AgentID, Username: username, PasswordHash: hash, Name: &name,
			Phone: OptionalString(req.Phone), Permissions: string(permsJSON), Status: models.AgentStatusActive,
			CreatedAt: now, UpdatedAt: now}
		if err := agentAuthPostgres.CreateSubaccountRepository(tx, &s); err != nil {
			return err
		}
		res = agentManagementDto.SubCreateResponse{ID: s.ID, Username: s.Username}
		return WriteLog(ctx, tx, actor, meta, targetSub, s.ID, s.Username, models.ChangeCreate, nil,
			map[string]any{"username": s.Username, "name": req.Name, "phone": req.Phone, "permissions": perms}, now)
	})
	return res, err
}

// lockOwnSub — lock แถว sub ที่จะแก้ · ต้องเป็น sub ของผู้เรียกเอง (MGMT-45) · อื่น = 402404
func lockOwnSub(tx *gorm.DB, actor agentAuthService.Actor, id uint) (models.Subaccount, error) {
	s, err := agentManagementPostgres.LockSubaccountViewRepository(tx, id)
	if err != nil {
		return s, err
	}
	if s.ID == 0 || s.AgentID != actor.AgentID {
		return s, apperr.ErrSubaccountNotFound
	}
	return s, nil
}

// UpdateSubaccountService — POST /manage/subaccounts/update-info (MGMT-42) · แทนทั้งชุด
func UpdateSubaccountService(ctx context.Context, actor agentAuthService.Actor, req agentManagementDto.SubUpdateRequest,
	meta agentAuthService.RequestMeta) error {
	perms, v, ok := agentManagementCore.NormalizeSubPermissions(actor.Role, *req.Permissions)
	if !ok {
		return permissionError(v)
	}
	permsJSON, err := json.Marshal(perms)
	if err != nil {
		return err
	}
	return database.DBConn.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		s, err := lockOwnSub(tx, actor, req.ID)
		if err != nil {
			return err
		}
		now := time.Now()
		if err := agentManagementPostgres.UpdateSubaccountInfoRepository(tx, s.ID, req.Name, OptionalString(req.Phone.Value), string(permsJSON), now); err != nil {
			return err
		}
		var oldPerms map[string]string
		_ = json.Unmarshal([]byte(s.Permissions), &oldPerms)
		return WriteLog(ctx, tx, actor, meta, targetSub, s.ID, s.Username, models.ChangeUpdateInfo,
			map[string]any{"name": StringOrEmpty(s.Name), "phone": StringOrEmpty(s.Phone), "permissions": oldPerms},
			map[string]any{"name": req.Name, "phone": req.Phone.Value, "permissions": perms}, now)
	})
}

// UpdateSubaccountStatusService — POST /manage/subaccounts/update-status (MGMT-43)
func UpdateSubaccountStatusService(ctx context.Context, actor agentAuthService.Actor, req agentManagementDto.SubStatusRequest,
	meta agentAuthService.RequestMeta) error {
	next := models.AgentStatusActive
	if req.Status == agentManagementDto.SubStatusInactive {
		next = models.AgentStatusSuspended
	}
	return database.DBConn.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		s, err := lockOwnSub(tx, actor, req.ID)
		if err != nil {
			return err
		}
		now := time.Now()
		if err := agentManagementPostgres.UpdateSubaccountStatusRepository(tx, s.ID, next, now); err != nil {
			return err
		}
		return WriteLog(ctx, tx, actor, meta, targetSub, s.ID, s.Username, models.ChangeSubStatus,
			map[string]string{"status": subStatus(s.Status, models.AgentStatusActive)}, map[string]string{"status": req.Status}, now)
	})
}

// permissionError — สิทธิ์ที่ส่งมาผิด (MGMT-50, MGMT-52) → 422 บอกเมนู
func permissionError(v agentManagementCore.PermissionViolation) error {
	field := "permissions." + v.Menu
	if v.Reason == "menu" {
		return apperr.ErrValidation.WithMessage(field+" ไม่มีเมนูนี้สำหรับบัญชีประเภทนี้", field+" is not a menu of this account type")
	}
	return apperr.ErrValidation.WithMessage(field+" ต้องเป็น off / view / edit (dashboard และ report สูงสุด view)",
		field+" must be off, view or edit (dashboard and report up to view)")
}

func notFoundAs(err, as error) error {
	if errors.Is(err, apperr.ErrNotFound) {
		return as
	}
	return err
}
