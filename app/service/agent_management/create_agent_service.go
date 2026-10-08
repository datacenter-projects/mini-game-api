package agentmanagement

import (
	"context"
	"time"

	accountCore "app/app/core/account"
	agentManagementCore "app/app/core/agent_management"
	agentManagementDto "app/app/internals/backoffice/dto/agent_management"
	"app/app/models"
	agentAuthPostgres "app/app/repository/postgres/agent_auth"
	agentManagementPostgres "app/app/repository/postgres/agent_management"
	agentAuthService "app/app/service/agent_auth"
	"app/pkg/utils"
	"app/platform/database"

	"gorm.io/gorm"
)

const (
	targetAgent  = "AGENT"
	TargetMember = "MEMBER"
)

// CreateAgentService — POST /api/v1/bo/pr/manage/agents/create (MGMT-02 – MGMT-25, MGMT-60)
// สิทธิ์ member / pt = edit ตรวจที่ route · payment = edit ตรวจที่นี่เมื่อส่ง balance (MGMT-51)
// เจ้าของ Key ของ account 1.3 ได้ Key พร้อมบัญชี (MGMT-04)
func CreateAgentService(ctx context.Context, actor agentAuthService.Actor, req agentManagementDto.CreateAgentRequest,
	meta agentAuthService.RequestMeta) (agentManagementDto.CreateAgentResponse, error) {
	var res agentManagementDto.CreateAgentResponse
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
		if id, replay, err := FindReplay(tx, req.RequestID, actor, targetAgent); err != nil || replay {
			if err != nil {
				return err
			}
			a, err := agentManagementPostgres.GetAgentProfileRepository(tx, id)
			if err != nil {
				return err
			}
			res = agentManagementDto.CreateAgentResponse{ID: a.ID, Username: a.Username, UserType: string(agentManagementCore.UserTypeOf(a.Role, a.AgentType))}
			return nil
		}

		c, err := LoadCreator(tx, actor)
		if err != nil {
			return err
		}
		// กฎที่ต้องดู DB ไล่ตามลำดับ field ใน body (รูปแบบเช็คครบแล้วใน DTO) — หัวข้อ 7.1
		newAcc, ok := agentManagementCore.ResolveNewAgent(c.UserType, agentManagementCore.UserType(req.UserType))
		if !ok {
			return CannotCreateTypeError(c.UserType)
		}
		if err := LockAndCheckIdentity(tx, req.Username, req.Phone, false); err != nil {
			return err
		}
		currencies, cv := agentManagementCore.ResolveCurrencies(c.UserType, newAcc.UserType, req.Currencies, c.Currencies)
		if cv != agentManagementCore.CurrencyOK {
			return CurrencyError(c.UserType, newAcc.UserType, cv, c.Currencies)
		}
		if err := CheckInitialBalance(req.BalanceMinor, c.newAccountSeamless(newAcc.UserType), currencies); err != nil {
			return err
		}
		if err := CheckCreatorBalance(tx, c, req.BalanceMinor); err != nil {
			return err
		}
		creatorIsMaster := c.UserType == agentManagementCore.UserTypeCompanySeamlessMaster
		for _, g := range SortedGroups(req.PT) {
			is := agentManagementCore.CheckChildPT(req.PT[g].Parsed, c.ReceivedBP[agentManagementCore.PTGroup(g)], creatorIsMaster)
			if err := ChildPTError(g, is); err != nil {
				return err
			}
		}

		now := time.Now()
		name := req.Name
		a := models.UserAgent{ParentID: &c.Agent.ID, Username: req.Username, PasswordHash: hash, Name: &name,
			Phone: OptionalString(req.Phone), AgentType: newAcc.AgentType, Role: newAcc.Role, Status: models.AgentStatusActive,
			CreatedAt: now, UpdatedAt: now}
		if err := agentAuthPostgres.CreateUserAgentRepository(tx, &a); err != nil {
			return err
		}

		curRows := make([]models.AgentCurrency, 0, len(currencies))
		for _, cur := range currencies {
			curRows = append(curRows, models.AgentCurrency{AgentID: a.ID, Currency: cur})
		}
		if err := agentManagementPostgres.CreateAgentCurrenciesRepository(tx, curRows); err != nil {
			return err
		}

		newIsMaster := newAcc.UserType == agentManagementCore.UserTypeCompanySeamlessMaster
		statusGame := map[string]bool{}
		var settings []models.AgentGameSetting
		for _, g := range agentManagementCore.Groups() {
			v := req.PT[string(g)].Parsed
			groupOn := *req.PT[string(g)].Status
			for _, game := range agentManagementCore.GamesOf(g) {
				on, sent := req.StatusGame[game.GameCode]
				if !sent {
					on = true // ไม่ส่ง = เปิด
				}
				statusGame[game.GameCode] = on
				settings = append(settings, models.AgentGameSetting{AgentID: a.ID, GameCode: game.GameCode, Category: game.Category,
					PTFromParentBP: v.PTFromParentBP, PTBP: agentManagementCore.InitialOwnPT(v.PTFromParentBP, newIsMaster),
					ForceBP: v.ForceBP, RemainBP: v.RemainBP, CommissionBP: v.CommissionBP,
					Status: groupOn, StatusGame: on, CreatedBy: actor.Username, CreatedAt: now, UpdatedBy: actor.Username, UpdatedAt: now})
			}
		}
		if err := agentManagementPostgres.CreateAgentGameSettingsRepository(tx, settings); err != nil {
			return err
		}

		if err := TransferInitialBalance(tx, c, models.BalanceOwnerAgent, a.ID, req.BalanceMinor, req.RequestID, actor, now); err != nil {
			return err
		}
		// MGMT-04: เจ้าของ Key ของ account 1.3 ได้ Key ใน tx เดียวกัน
		if accountCore.IsAPIKeyOwner(newAcc.UserType) {
			if err := CreateAPICredentialIfAbsent(tx, a.ID); err != nil {
				return err
			}
		}
		if err := agentManagementPostgres.CreateCreateRequestRepository(tx, &models.CreateRequest{RequestID: req.RequestID, CreatorType: actor.AccountType,
			CreatorID: actor.AccountID(), TargetType: targetAgent, TargetID: a.ID, CreatedAt: now}); err != nil {
			return err
		}

		ptLog := map[string]agentManagementCore.ChildPT{}
		for g, v := range req.PT {
			ptLog[g] = v.Parsed
		}
		created, err := ChangeLog(actor, meta, req.RequestID, targetAgent, a.ID, a.Username, models.ChangeCreate, nil,
			map[string]any{"user_type": newAcc.UserType, "username": a.Username, "name": req.Name, "phone": req.Phone,
				"parent_id": c.Agent.ID, "currencies": currencies, "pt_bp": ptLog, "status_game": statusGame}, now)
		if err != nil {
			return err
		}
		rows := []models.AccountChangeLog{created}
		if len(req.BalanceMinor) > 0 {
			row, err := ChangeLog(actor, meta, req.RequestID, targetAgent, a.ID, a.Username, models.ChangeInitialBalance, nil,
				map[string]any{"amounts_minor": req.BalanceMinor}, now)
			if err != nil {
				return err
			}
			rows = append(rows, row)
		}
		if err := agentManagementPostgres.CreateAccountChangeLogsRepository(tx, rows); err != nil {
			return err
		}

		res = agentManagementDto.CreateAgentResponse{ID: a.ID, Username: a.Username, UserType: string(newAcc.UserType)}
		return nil
	})
	return res, err
}
