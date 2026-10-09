package membermanagement

import (
	"context"

	agentAuthCore "app/app/core/agent_auth"
	agentManagementCore "app/app/core/agent_management"
	memberManagementDto "app/app/internals/backoffice/dto/member_management"
	agentManagementPostgres "app/app/repository/postgres/agent_management"
	memberManagementPostgres "app/app/repository/postgres/member_management"
	agentAuthService "app/app/service/agent_auth"
	agentManagementService "app/app/service/agent_management"
	"app/pkg/apperr"
	"app/platform/database"
)

// GetMemberDetailService — POST /api/v1/bo/pr/manage/members/detail (MGMT-29) · ผู้สร้างต้องเป็นตัวเองหรืออยู่ในสายล่าง
func GetMemberDetailService(ctx context.Context, actor agentAuthService.Actor, id uint) (memberManagementDto.MemberDetailResponse, error) {
	var res memberManagementDto.MemberDetailResponse
	db := database.DBConn.WithContext(ctx)
	m, err := memberManagementPostgres.GetUserMemberDetailRepository(db, id)
	if err != nil {
		return res, agentManagementService.NotFoundAsDownline(err)
	}
	if m.AgentID != actor.AgentID {
		ok, err := agentManagementPostgres.IsInDownlineRepository(db, actor.AgentID, m.AgentID)
		if err != nil {
			return res, err
		}
		if !ok {
			return res, apperr.ErrDownlineNotFound
		}
	}
	creator, err := agentManagementPostgres.GetAgentProfileRepository(db, m.AgentID)
	if err != nil {
		return res, err
	}
	creatorStatus, err := agentManagementService.ChainStatus(db, creator.ID, creator.Status)
	if err != nil {
		return res, err
	}
	amount := map[string]float64{m.Currency: m.Credit} // ยอด = user_members.credit
	member := string(agentManagementCore.UserTypeMember)
	res = memberManagementDto.MemberDetailResponse{ID: m.ID, Role: member, UserType: member, Username: m.Username, Name: m.Name,
		Phone: agentManagementService.StringOrEmpty(m.Phone), Status: string(agentAuthCore.WorstStatus(creatorStatus, m.Status)),
		ParentUsername: creator.Username, Currencies: []string{m.Currency},
		Balances:    agentManagementService.BalanceViews([]string{m.Currency}, amount),
		LastLoginAt: agentManagementService.OptionalTime(m.LastLoginAt), LastLoginIP: agentManagementService.StringOrEmpty(m.LastLoginIP),
		CreatedAt: agentManagementService.OptionalTime(&m.CreatedAt)}
	show, err := agentManagementService.CanSeePT(ctx, actor)
	if err != nil {
		return res, err
	}
	if show {
		settings, err := memberManagementPostgres.ListUserMemberGameSettingsByIDsRepository(db, []uint{m.ID})
		if err != nil {
			return res, err
		}
		res.PT = agentManagementService.MemberPTViews(settings)
	}
	return res, nil
}
