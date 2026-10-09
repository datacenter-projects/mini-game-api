// Package membermanagement คือ business logic ของเส้น /manage/members/* — spec: docs/modules/agent_management.md
// helper ที่ใช้ร่วมกับ agent (ผู้สร้าง · ยอดตั้งต้น · log) อยู่ใน service/agent_management · import ทางเดียว
package membermanagement

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	agentManagementCore "app/app/core/agent_management"
	memberManagementDto "app/app/internals/backoffice/dto/member_management"
	"app/app/models"
	agentManagementPostgres "app/app/repository/postgres/agent_management"
	memberManagementPostgres "app/app/repository/postgres/member_management"
	agentAuthService "app/app/service/agent_auth"
	agentManagementService "app/app/service/agent_management"
	"app/pkg/apperr"
	"app/pkg/utils"
	"app/platform/database"

	"gorm.io/gorm"
)

// memberPTValue — ค่า PT ของ Member 1 กลุ่มใน account_change_logs (MGMT-60)
type memberPTValue struct {
	PTBP         int `json:"pt_bp"`
	RemainBP     int `json:"remain_bp"`
	CommissionBP int `json:"commission_bp"`
}

// CreateMemberService — POST /api/v1/bo/pr/manage/members/create (MGMT-02, MGMT-05 – MGMT-15A, MGMT-21, MGMT-60)
// pt ของ Member ไม่เกินค่าที่ผู้สร้างได้รับ (lock แถวผู้สร้าง FOR SHARE ใน LoadCreator) · remain = ค่าที่ผู้สร้างได้รับ − pt
func CreateMemberService(ctx context.Context, actor agentAuthService.Actor, req memberManagementDto.CreateMemberRequest, meta agentAuthService.RequestMeta) (memberManagementDto.CreateMemberResponse, error) {
	var res memberManagementDto.CreateMemberResponse
	if len(req.BalanceMinor) > 0 {
		if err := agentManagementService.CheckPermissionService(ctx, actor, agentManagementCore.MenuPayment, agentManagementCore.LevelEdit); err != nil {
			return res, err
		}
	}
	hash, err := utils.HashPassword(req.Password)
	if err != nil {
		return res, err
	}

	err = database.DBConn.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if id, replay, err := agentManagementService.FindReplay(tx, req.RequestID, actor, agentManagementService.TargetMember); err != nil || replay {
			if err != nil {
				return err
			}
			m, err := memberManagementPostgres.GetUserMemberProfileRepository(tx, id)
			if err != nil {
				return err
			}
			res = memberManagementDto.CreateMemberResponse{ID: m.ID, Username: m.Username}
			return nil
		}

		c, err := agentManagementService.LoadCreator(tx, actor)
		if err != nil {
			return err
		}
		if !agentManagementCore.CanCreateMember(c.UserType) {
			return apperr.ErrCannotCreateType
		}
		currencies, _ := agentManagementCore.ResolveCurrencies(c.UserType, agentManagementCore.UserTypeMember, nil, c.Currencies)
		if len(currencies) != 1 { // ผู้สร้าง Member มี 1 สกุลเสมอ (MGMT-11 – MGMT-13) — ไม่ใช่ = ข้อมูลผู้สร้างไม่ครบ
			return apperr.ErrInternal.Wrap(fmt.Errorf("creator %d has %d currencies, want 1", c.Agent.ID, len(currencies)))
		}
		for g, v := range req.PT { // ไล่ตาม field: pt → commission_percent
			received := c.ReceivedBP[agentManagementCore.PTGroup(g)]
			if err := agentManagementService.OwnPTError(g, agentManagementCore.ValidateMemberPT(v.PTBP, received), received); err != nil {
				return err
			}
			if err := agentManagementService.PTError(g, agentManagementCore.ValidateMemberCommission(v.CommissionBP)); err != nil {
				return err
			}
		}
		if err := agentManagementService.CheckInitialBalance(req.BalanceMinor, agentManagementCore.IsSeamless(c.CompanyType), currencies); err != nil {
			return err
		}
		if err := agentManagementService.LockAndCheckIdentity(tx, req.Username, req.Phone, true); err != nil {
			return err
		}

		now := time.Now()
		cnf, err := json.Marshal(agentManagementCore.ChildChain(c.Chain, c.Agent.ID, c.Agent.Role)) // สายชั้นบน = สายของผู้สร้าง + ผู้สร้าง
		if err != nil {
			return err
		}
		m := models.UserMember{AgentID: c.Agent.ID, Username: req.Username, PasswordHash: hash, Name: req.Name,
			Phone: agentManagementService.OptionalString(req.Phone), Currency: currencies[0], Status: models.AgentStatusActive,
			Cnf: string(cnf), CreatedAt: now, UpdatedAt: now}
		if err := memberManagementPostgres.CreateUserMemberRepository(tx, &m); err != nil {
			return err
		}

		var settings []models.UserMemberGameSetting
		ptLog := map[string]memberPTValue{}
		for _, g := range agentManagementCore.Groups() {
			v := req.PT[string(g)]
			remain := agentManagementCore.MemberRemain(c.ReceivedBP[g], v.PTBP)
			ptLog[string(g)] = memberPTValue{PTBP: v.PTBP, RemainBP: remain, CommissionBP: v.CommissionBP}
			for _, game := range agentManagementCore.GamesOf(g) {
				settings = append(settings, models.UserMemberGameSetting{UserMemberID: m.ID, GameCode: game.GameCode, Category: game.Category,
					PTBP: v.PTBP, RemainBP: remain, CommissionBP: v.CommissionBP,
					CreatedBy: actor.Username, CreatedAt: now, UpdatedBy: actor.Username, UpdatedAt: now})
			}
		}
		if err := memberManagementPostgres.CreateUserMemberGameSettingsRepository(tx, settings); err != nil {
			return err
		}

		if err := agentManagementService.TransferInitialBalance(tx, c, models.BalanceOwnerMember, m.ID, req.BalanceMinor, req.RequestID, actor, now); err != nil {
			return err
		}
		if err := agentManagementPostgres.CreateCreateRequestRepository(tx, &models.CreateRequest{RequestID: req.RequestID, CreatorType: actor.AccountType,
			CreatorID: actor.AccountID(), TargetType: agentManagementService.TargetMember, TargetID: m.ID, CreatedAt: now}); err != nil {
			return err
		}

		created, err := agentManagementService.ChangeLog(actor, meta, req.RequestID, agentManagementService.TargetMember, m.ID, m.Username, models.ChangeCreate, nil,
			map[string]any{"username": m.Username, "name": req.Name, "phone": req.Phone, "agent_id": c.Agent.ID,
				"currency": m.Currency, "pt": ptLog}, now)
		if err != nil {
			return err
		}
		rows := []models.AccountChangeLog{created}
		if len(req.BalanceMinor) > 0 {
			row, err := agentManagementService.ChangeLog(actor, meta, req.RequestID, agentManagementService.TargetMember, m.ID, m.Username, models.ChangeInitialBalance, nil,
				map[string]any{"amounts_minor": req.BalanceMinor}, now)
			if err != nil {
				return err
			}
			rows = append(rows, row)
		}
		if err := agentManagementPostgres.CreateAccountChangeLogsRepository(tx, rows); err != nil {
			return err
		}

		res = memberManagementDto.CreateMemberResponse{ID: m.ID, Username: m.Username}
		return nil
	})
	return res, err
}
