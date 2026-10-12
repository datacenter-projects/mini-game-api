package agentmanagement

import (
	"context"
	"encoding/json"
	"errors"
	"maps"
	"strings"
	"time"

	agentAuthCore "app/app/core/agent_auth"
	agentManagementCore "app/app/core/agent_management"
	agentManagementDto "app/app/internals/backoffice/dto/agent_management"
	"app/app/models"
	agentAuthPostgres "app/app/repository/postgres/agent_auth"
	agentManagementPostgres "app/app/repository/postgres/agent_management"
	agentAuthRedis "app/app/repository/redis/agent_auth"
	agentAuthService "app/app/service/agent_auth"
	"app/pkg/apperr"
	"app/pkg/utils"
	"app/platform/database"
	"app/platform/logger"

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

// SubStatus — สถานะที่แสดง (MGMT-43): เจ้าของ / หัวสายถูกระงับ / ล็อก = สถานะนั้น · sub ถูกตั้ง SUSPENDED = INACTIVE
func SubStatus(own, ownerChain models.AgentStatus) string {
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
	return agentManagementDto.SubView{ID: s.ID, Username: s.Username, Name: StringOrEmpty(s.Name), PhoneCountryCode: StringOrEmpty(s.PhoneCountryCode), Phone: StringOrEmpty(s.Phone),
		Status: SubStatus(s.Status, ownerChain), Permissions: out, CreatedAt: OptionalTime(&s.CreatedAt),
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
	rows, total, err := agentManagementPostgres.ListSubaccountsRepository(db, owner.ID, req.Keyword, page.Offset(), page.Limit)
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

// GetSubaccountService — POST /manage/subaccounts/detail/get (MGMT-46) · sub ของตัวเองหรือของสายล่าง · อื่น = 402404
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
		return res, permissionError(v, actor.Role)
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
		// MustChangePassword: เจ้าของตั้งรหัสให้ sub ต้องเปลี่ยนเองตอน login ครั้งแรก · ไม่ตั้งวันหมดอายุ (MGMT-41 · lead SQ3)
		s := models.Subaccount{AgentID: actor.AgentID, Username: username, PasswordHash: hash, Name: &name,
			PhoneCountryCode: OptionalString(req.PhoneCountryCode), Phone: OptionalString(req.Phone), Permissions: string(permsJSON), Status: models.AgentStatusActive,
			MustChangePassword: true,
			CreatedAt:          now, UpdatedAt: now}
		if err := agentAuthPostgres.CreateSubaccountRepository(tx, &s); err != nil {
			return err
		}
		res = agentManagementDto.SubCreateResponse{ID: s.ID, Username: s.Username}
		return WriteLog(ctx, tx, actor, meta, targetSub, s.ID, s.Username, models.ChangeCreate, nil,
			map[string]any{"username": s.Username, "name": req.Name, "phone_country_code": req.PhoneCountryCode, "phone": req.Phone, "permissions": perms}, now)
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

// UpdateSubaccountService — POST /manage/subaccounts/detail/update (MGMT-42 · lead SN4 / SQ5 / SQ6)
// เจ้าของเท่านั้น · tx เดียว: section ไหนผิดไม่บันทึกเลย · ทำตามลำดับ info → permissions
// log แยกแถว UPDATE_INFO / SUB_PERMISSIONS เฉพาะ section ที่เปลี่ยนจริง · ค่าเดิม = ไม่เขียน log
func UpdateSubaccountService(ctx context.Context, actor agentAuthService.Actor, req agentManagementDto.SubUpdateRequest,
	meta agentAuthService.RequestMeta) error {
	var perms map[agentManagementCore.Menu]agentManagementCore.Level
	if req.Permissions != nil { // ตรวจก่อนเปิด tx — ค่าที่ใช้ได้ขึ้นกับประเภทเจ้าของ (MGMT-52)
		var v agentManagementCore.PermissionViolation
		var ok bool
		if perms, v, ok = agentManagementCore.NormalizeSubPermissions(actor.Role, *req.Permissions); !ok {
			return permissionError(v, actor.Role)
		}
	}
	return database.DBConn.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		s, err := lockOwnSub(tx, actor, req.ID)
		if err != nil {
			return err
		}
		now := time.Now()
		if req.Info != nil {
			if err := applySubInfo(ctx, tx, actor, meta, s, *req.Info, now); err != nil {
				return err
			}
		}
		if req.Permissions != nil {
			return applySubPermissions(ctx, tx, actor, meta, s, perms, now)
		}
		return nil
	})
}

// applySubInfo — section info (MGMT-41 · MGMT-08 — เบอร์ของ sub ซ้ำได้) · field ที่ไม่ส่งคงค่าเดิม · ไม่เปลี่ยน = ไม่เขียน log
func applySubInfo(ctx context.Context, tx *gorm.DB, actor agentAuthService.Actor, meta agentAuthService.RequestMeta,
	s models.Subaccount, in agentManagementDto.AgentInfoSection, now time.Time) error {
	old := AgentInfoValue{StringOrEmpty(s.Name), StringOrEmpty(s.PhoneCountryCode), StringOrEmpty(s.Phone)}
	next := old
	if in.Name != nil {
		next.Name = *in.Name
	}
	if in.Phone.Present { // DTO บังคับส่งคู่กับ phone_country_code
		next.PhoneCountryCode, next.Phone = in.PhoneCountryCode.Value, in.Phone.Value
	}
	if next == old {
		return nil
	}
	if err := agentManagementPostgres.UpdateSubaccountInfoRepository(tx, s.ID, next.Name, OptionalString(next.PhoneCountryCode), OptionalString(next.Phone), now); err != nil {
		return err
	}
	return WriteLog(ctx, tx, actor, meta, targetSub, s.ID, s.Username, models.ChangeUpdateInfo, old, next, now)
}

// applySubPermissions — section permissions แทนทั้งชุด (MGMT-50) · เทียบกับสิทธิ์เดิมแบบครบทุกเมนู (ไม่มี key = off) · ไม่เปลี่ยน = ไม่เขียน log
// request ถัดไปของ sub ใช้สิทธิ์ใหม่ทันที (RequirePermission อ่านจาก DB ทุก request)
func applySubPermissions(ctx context.Context, tx *gorm.DB, actor agentAuthService.Actor, meta agentAuthService.RequestMeta,
	s models.Subaccount, perms map[agentManagementCore.Menu]agentManagementCore.Level, now time.Time) error {
	stored := map[string]string{}
	if s.Permissions != "" {
		if err := json.Unmarshal([]byte(s.Permissions), &stored); err != nil {
			return apperr.ErrInternal.Wrap(err)
		}
	}
	old := agentManagementCore.SubPermissionsView(actor.Role, stored)
	if maps.Equal(old, perms) {
		return nil
	}
	permsJSON, err := json.Marshal(perms)
	if err != nil {
		return err
	}
	if err := agentManagementPostgres.UpdateSubaccountPermissionsRepository(tx, s.ID, string(permsJSON), now); err != nil {
		return err
	}
	return WriteLog(ctx, tx, actor, meta, targetSub, s.ID, s.Username, models.ChangeSubPermissions,
		map[string]any{"permissions": old}, map[string]any{"permissions": perms}, now)
}

// UpdateSubaccountStatusService — POST /manage/subaccounts/status/update (MGMT-43 · lead SA1 / SQ1)
// ตั้ง INACTIVE: ลบ session ทุกอันของ sub นั้นหลัง commit (request ถัดไป 401203) · login ใหม่ไม่ได้ (401312)
func UpdateSubaccountStatusService(ctx context.Context, actor agentAuthService.Actor, req agentManagementDto.SubStatusRequest,
	meta agentAuthService.RequestMeta) error {
	next := models.AgentStatusActive
	if req.Status == agentManagementDto.SubStatusInactive {
		next = models.AgentStatusSuspended
	}
	var changedID uint
	err := database.DBConn.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		s, err := lockOwnSub(tx, actor, req.ID)
		if err != nil {
			return err
		}
		if s.Status == next {
			return nil // ค่าเดิม = 200 ไม่เขียน log (กฎเดียวกับ agents/status/update — lead MQ5 / V1 · SA1)
		}
		now := time.Now()
		if err := agentManagementPostgres.UpdateSubaccountStatusRepository(tx, s.ID, next, now); err != nil {
			return err
		}
		changedID = s.ID
		return WriteLog(ctx, tx, actor, meta, targetSub, s.ID, s.Username, models.ChangeSubStatus,
			map[string]string{"status": SubStatus(s.Status, models.AgentStatusActive)}, map[string]string{"status": req.Status}, now)
	})
	if err != nil || changedID == 0 || next != models.AgentStatusSuspended {
		return err
	}
	// ลบไม่สำเร็จไม่ทำให้คำขอล้ม — middleware Authenticated ปฏิเสธ sub INACTIVE อยู่แล้ว (401312)
	if err := agentAuthRedis.DeleteBOSessionsOfAccountRepository(ctx, models.AccountTypeSub, changedID); err != nil {
		logger.Ctx(ctx).Warnw("delete sessions of inactive sub failed", "sub_id", changedID, "error", err)
	}
	return nil
}

// permissionError — สิทธิ์ที่ส่งมาผิด (MGMT-50, MGMT-52) → 422 บอกเมนู
func permissionError(v agentManagementCore.PermissionViolation, ownerRole models.AgentRole) error {
	field := "permissions." + v.Menu
	if v.Reason == "menu" {
		menus := agentManagementCore.MenusForRole(ownerRole)
		names := make([]string, len(menus))
		for i, m := range menus {
			names[i] = string(m)
		}
		list := strings.Join(names, ", ")
		return apperr.ErrValidation.WithMessage(field+" ไม่มีเมนูนี้ · เมนูที่ให้ได้: "+list, field+" is not a menu · allowed menus: "+list)
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
