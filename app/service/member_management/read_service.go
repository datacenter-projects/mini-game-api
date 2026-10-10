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

// GetMemberDetailService — POST /api/v1/bo/pr/manage/members/detail/get (M2 · MGMT-29) · ผู้สร้างต้องเป็นตัวเองหรืออยู่ในสายล่าง
// pt แสดงเสมอ (ไม่มีเมนู pt แล้ว — lead P4)
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
	settings, err := memberManagementPostgres.ListUserMemberGameSettingsByIDsRepository(db, []uint{m.ID})
	if err != nil {
		return res, err
	}
	amount := map[string]float64{m.Currency: m.Credit} // ยอด = user_members.credit · ฝั่ง Seamless = 0 เสมอ (MGMT-15B)
	member := string(agentManagementCore.UserTypeMember)
	res = memberManagementDto.MemberDetailResponse{ID: m.ID, Role: member, UserType: member, Username: m.Username, Name: m.Name,
		PhoneCountryCode: agentManagementService.StringOrEmpty(m.PhoneCountryCode), Phone: agentManagementService.StringOrEmpty(m.Phone),
		Status:         string(agentAuthCore.WorstStatus(creatorStatus, m.Status)),
		ParentUsername: creator.Username, Currencies: []string{m.Currency},
		Balances:    agentManagementService.BalanceViews([]string{m.Currency}, amount),
		PT:          agentManagementService.MemberPTViews(settings),
		LastLoginAt: agentManagementService.OptionalTime(m.LastLoginAt), LastLoginIP: agentManagementService.StringOrEmpty(m.LastLoginIP),
		CreatedAt: agentManagementService.OptionalTime(&m.CreatedAt)}
	return res, nil
}

// MemberListMax — members/list ส่งไม่เกินกี่รายการ (lead M5)
const MemberListMax = 500

// ListMembersService — POST /api/v1/bo/pr/manage/members/list (M5 · MGMT-35M) · Member ลูกตรงของตัวเอง A→Z ไว้ทำ dropdown
// status = สถานะที่ใช้งานจริง (ผู้เรียกถูกระงับ = Member ทุกคนแสดง SUSPENDED)
func ListMembersService(ctx context.Context, actor agentAuthService.Actor, req memberManagementDto.MemberListRequest) ([]memberManagementDto.MemberListRow, error) {
	db := database.DBConn.WithContext(ctx)
	me, err := agentManagementPostgres.GetAgentProfileRepository(db, actor.AgentID)
	if err != nil {
		return nil, err
	}
	myStatus, err := agentManagementService.ChainStatus(db, me.ID, me.Status)
	if err != nil {
		return nil, err
	}
	members, err := memberManagementPostgres.ListUserMemberDirectChildrenRepository(db, actor.AgentID, req.Keyword, MemberListMax)
	if err != nil {
		return nil, err
	}
	out := make([]memberManagementDto.MemberListRow, len(members))
	for i, m := range members {
		out[i] = memberManagementDto.MemberListRow{ID: m.ID, Username: m.Username, Name: m.Name,
			Status: string(agentAuthCore.WorstStatus(myStatus, m.Status))}
	}
	return out, nil
}
