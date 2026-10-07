package agentmanagement

import (
	"context"
	"fmt"
	"time"

	agentManagementCore "app/app/core/agent_management"
	agentManagementDto "app/app/internals/backoffice/dto/agent_management"
	"app/app/models"
	"app/app/repository/postgres"
	agentAuthService "app/app/service/agent_auth"
	"app/pkg/apperr"
	"app/pkg/utils"
	"app/platform/database"

	"gorm.io/gorm"
)

// CreateMemberService — POST /api/v1/bo/pr/manage/members (MGMT-02, MGMT-05 – MGMT-15A, MGMT-21, MGMT-60)
func CreateMemberService(ctx context.Context, actor agentAuthService.Actor, req agentManagementDto.CreateMemberRequest,
	meta agentAuthService.RequestMeta) (agentManagementDto.CreateMemberResponse, error) {
	var res agentManagementDto.CreateMemberResponse
	if len(req.BalanceMinor) > 0 {
		if err := CheckPermissionService(ctx, actor, agentManagementCore.MenuPayment, agentManagementCore.LevelEdit); err != nil {
			return res, err
		}
	}
	hash, err := utils.HashPassword(req.Password)
	if err != nil {
		return res, err
	}

	err = database.DBConn.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if id, replay, err := findReplay(tx, req.RequestID, actor, targetMember); err != nil || replay {
			if err != nil {
				return err
			}
			m, err := postgres.GetMemberProfileRepository(tx, id)
			if err != nil {
				return err
			}
			res = agentManagementDto.CreateMemberResponse{ID: m.ID, Username: m.Username}
			return nil
		}

		c, err := loadCreator(tx, actor)
		if err != nil {
			return err
		}
		if !agentManagementCore.CanCreateMember(c.userType) {
			return apperr.ErrCannotCreateType
		}
		currencies, _ := agentManagementCore.ResolveCurrencies(c.userType, agentManagementCore.UserTypeMember, nil, c.currencies)
		if len(currencies) != 1 { // ผู้สร้าง Member มี 1 สกุลเสมอ (MGMT-11 – MGMT-13) — ไม่ใช่ = ข้อมูลผู้สร้างไม่ครบ
			return apperr.ErrInternal.Wrap(fmt.Errorf("creator %d has %d currencies, want 1", c.agent.ID, len(currencies)))
		}
		for g, v := range req.PT {
			if err := ptError(g, agentManagementCore.ValidateMemberCommission(v.CommissionBP)); err != nil {
				return err
			}
		}
		if err := checkInitialBalance(req.BalanceMinor, agentManagementCore.IsSeamless(c.companyType), currencies); err != nil {
			return err
		}
		if err := lockAndCheckIdentity(tx, req.Username, req.Phone, true); err != nil {
			return err
		}

		now := time.Now()
		m := models.Member{AgentID: c.agent.ID, Username: req.Username, PasswordHash: hash, Name: req.Name,
			Phone: optionalString(req.Phone), Currency: currencies[0], Status: models.AgentStatusActive, CreatedAt: now, UpdatedAt: now}
		if err := postgres.CreateMemberRepository(tx, &m); err != nil {
			return err
		}

		var settings []models.MemberGameSetting
		commission := map[string]int{}
		for _, g := range agentManagementCore.Groups() {
			bp := req.PT[string(g)].CommissionBP
			commission[string(g)] = bp
			for _, game := range agentManagementCore.GamesOf(g) {
				settings = append(settings, models.MemberGameSetting{MemberID: m.ID, GameCode: game.GameCode, Category: game.Category,
					CommissionBP: bp, UpdatedAt: now})
			}
		}
		if err := postgres.CreateMemberGameSettingsRepository(tx, settings); err != nil {
			return err
		}

		if err := transferInitialBalance(tx, c, models.BalanceOwnerMember, m.ID, req.BalanceMinor, req.RequestID, actor, now); err != nil {
			return err
		}
		if err := postgres.CreateCreateRequestRepository(tx, &models.CreateRequest{RequestID: req.RequestID, CreatorType: actor.AccountType,
			CreatorID: actor.AccountID(), TargetType: targetMember, TargetID: m.ID, CreatedAt: now}); err != nil {
			return err
		}

		created, err := changeLog(actor, meta, req.RequestID, targetMember, m.ID, m.Username, models.ChangeCreate, nil,
			map[string]any{"username": m.Username, "name": req.Name, "phone": req.Phone, "agent_id": c.agent.ID,
				"currency": m.Currency, "commission_bp": commission}, now)
		if err != nil {
			return err
		}
		rows := []models.AccountChangeLog{created}
		if len(req.BalanceMinor) > 0 {
			row, err := changeLog(actor, meta, req.RequestID, targetMember, m.ID, m.Username, models.ChangeInitialBalance, nil,
				map[string]any{"amounts_minor": req.BalanceMinor}, now)
			if err != nil {
				return err
			}
			rows = append(rows, row)
		}
		if err := postgres.CreateAccountChangeLogsRepository(tx, rows); err != nil {
			return err
		}

		res = agentManagementDto.CreateMemberResponse{ID: m.ID, Username: m.Username}
		return nil
	})
	return res, err
}
