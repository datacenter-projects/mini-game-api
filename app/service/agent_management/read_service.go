package agentmanagement

import (
	"context"
	"errors"

	agentAuthCore "app/app/core/agent_auth"
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

// ListDownlinesService — GET /api/v1/bo/pr/manage/downlines (MGMT-26 – MGMT-28)
// parent_id ไม่ส่ง = ตัวเอง · อื่นต้องเป็นบัญชีฝั่ง agent ในสายล่าง (402402) · ข้อมูลประกอบอ่านทีละหน้าแบบ batch
func ListDownlinesService(ctx context.Context, actor agentAuthService.Actor, q agentManagementDto.DownlinesQuery,
	page utils.Page) ([]agentManagementDto.DownlineRow, int64, error) {
	db := database.DBConn.WithContext(ctx)
	parentID := actor.AgentID
	if q.ParentID != 0 && q.ParentID != actor.AgentID {
		ok, err := postgres.IsInDownlineRepository(db, actor.AgentID, q.ParentID)
		if err != nil {
			return nil, 0, err
		}
		if !ok {
			return nil, 0, apperr.ErrDownlineNotFound
		}
		parentID = q.ParentID
	}
	parent, err := postgres.GetAgentProfileRepository(db, parentID)
	if err != nil {
		return nil, 0, notFoundAsDownline(err)
	}
	parentStatus, err := chainStatus(db, parent.ID, parent.Status)
	if err != nil {
		return nil, 0, err
	}
	showPT, err := canSeePT(ctx, actor)
	if err != nil {
		return nil, 0, err
	}

	rows, total, err := postgres.ListDownlinesRepository(db, parentID, q.Q, page.Offset(), page.Limit)
	if err != nil {
		return nil, 0, err
	}
	var agentIDs, memberIDs []uint
	for _, r := range rows {
		if r.IsMember {
			memberIDs = append(memberIDs, r.ID)
		} else {
			agentIDs = append(agentIDs, r.ID)
		}
	}
	extra, err := loadRowExtras(db, agentIDs, memberIDs)
	if err != nil {
		return nil, 0, err
	}

	out := make([]agentManagementDto.DownlineRow, len(rows))
	for i, r := range rows {
		row := agentManagementDto.DownlineRow{ID: r.ID, Username: r.Username, Name: stringOrEmpty(r.Name), Phone: stringOrEmpty(r.Phone),
			Status: string(agentAuthCore.WorstStatus(parentStatus, r.Status))}
		if r.IsMember {
			row.Role, row.UserType = string(agentManagementCore.UserTypeMember), string(agentManagementCore.UserTypeMember)
			row.Balances = BalanceViews([]string{stringOrEmpty(r.Currency)}, extra.memberBalance[r.ID])
			if showPT {
				row.PT = MemberPTViews(extra.memberSettings[r.ID])
			}
		} else {
			row.Role, row.UserType = string(r.Role), string(agentManagementCore.UserTypeOf(r.Role, r.AgentType))
			row.Balances = BalanceViews(extra.agentCurrencies[r.ID], extra.agentBalance[r.ID])
			if showPT {
				pt, _ := AgentPTViews(extra.agentSettings[r.ID])
				row.PT = pt
			}
		}
		out[i] = row
	}
	return out, total, nil
}

// rowExtras — สกุล ยอด และค่าหุ้นส่วนของทุกแถวในหน้า (query ละครั้ง ไม่ N+1 — กฎข้อ 23)
type rowExtras struct {
	agentCurrencies map[uint][]string
	agentBalance    map[uint]map[string]int64
	agentSettings   map[uint][]models.AgentGameSetting
	memberBalance   map[uint]map[string]int64
	memberSettings  map[uint][]models.MemberGameSetting
}

func loadRowExtras(db *gorm.DB, agentIDs, memberIDs []uint) (rowExtras, error) {
	e := rowExtras{map[uint][]string{}, map[uint]map[string]int64{}, map[uint][]models.AgentGameSetting{},
		map[uint]map[string]int64{}, map[uint][]models.MemberGameSetting{}}
	curs, err := postgres.ListAgentCurrenciesByIDsRepository(db, agentIDs)
	if err != nil {
		return e, err
	}
	for _, c := range curs {
		e.agentCurrencies[c.AgentID] = append(e.agentCurrencies[c.AgentID], c.Currency)
	}
	abs, err := postgres.ListAgentBalancesByIDsRepository(db, agentIDs)
	if err != nil {
		return e, err
	}
	for _, b := range abs {
		if e.agentBalance[b.AgentID] == nil {
			e.agentBalance[b.AgentID] = map[string]int64{}
		}
		e.agentBalance[b.AgentID][b.Currency] = b.Amount
	}
	ags, err := postgres.ListAgentGameSettingsByIDsRepository(db, agentIDs)
	if err != nil {
		return e, err
	}
	for _, s := range ags {
		e.agentSettings[s.AgentID] = append(e.agentSettings[s.AgentID], s)
	}
	mbs, err := postgres.ListMemberBalancesByIDsRepository(db, memberIDs)
	if err != nil {
		return e, err
	}
	for _, b := range mbs {
		if e.memberBalance[b.MemberID] == nil {
			e.memberBalance[b.MemberID] = map[string]int64{}
		}
		e.memberBalance[b.MemberID][b.Currency] = b.Amount
	}
	mgs, err := postgres.ListMemberGameSettingsByIDsRepository(db, memberIDs)
	if err != nil {
		return e, err
	}
	for _, s := range mgs {
		e.memberSettings[s.MemberID] = append(e.memberSettings[s.MemberID], s)
	}
	return e, nil
}

// GetAgentDetailService — GET /api/v1/bo/pr/manage/agents/:id (MGMT-29) · เฉพาะบัญชีในสายล่าง (402402)
func GetAgentDetailService(ctx context.Context, actor agentAuthService.Actor, id uint) (agentManagementDto.AgentDetailResponse, error) {
	var res agentManagementDto.AgentDetailResponse
	db := database.DBConn.WithContext(ctx)
	ok, err := postgres.IsInDownlineRepository(db, actor.AgentID, id)
	if err != nil {
		return res, err
	}
	if !ok {
		return res, apperr.ErrDownlineNotFound
	}
	a, err := postgres.GetAgentDetailRepository(db, id)
	if err != nil {
		return res, notFoundAsDownline(err)
	}
	parent, err := postgres.GetAgentProfileRepository(db, *a.ParentID) // อยู่ในสายล่าง = มี parent เสมอ
	if err != nil {
		return res, err
	}
	status, err := chainStatus(db, a.ID, a.Status)
	if err != nil {
		return res, err
	}
	extra, err := loadRowExtras(db, []uint{a.ID}, nil)
	if err != nil {
		return res, err
	}
	pt, statusGame := AgentPTViews(extra.agentSettings[a.ID])
	res = agentManagementDto.AgentDetailResponse{ID: a.ID, Role: string(a.Role),
		UserType: string(agentManagementCore.UserTypeOf(a.Role, a.AgentType)), Username: a.Username,
		Name: stringOrEmpty(a.Name), Phone: stringOrEmpty(a.Phone), Status: string(status), ParentUsername: parent.Username,
		Currencies: append([]string{}, extra.agentCurrencies[a.ID]...),
		Balances:   BalanceViews(extra.agentCurrencies[a.ID], extra.agentBalance[a.ID]),
		StatusGame: statusGame, PasscodeSet: a.PasscodeHash != nil,
		LastLoginAt: optionalTime(a.LastLoginAt), LastLoginIP: stringOrEmpty(a.LastLoginIP), CreatedAt: optionalTime(&a.CreatedAt)}
	show, err := canSeePT(ctx, actor)
	if err != nil {
		return res, err
	}
	if show {
		res.PT = pt
	}
	return res, nil
}

// GetMemberDetailService — GET /api/v1/bo/pr/manage/members/:id (MGMT-29) · ผู้สร้างต้องเป็นตัวเองหรืออยู่ในสายล่าง
func GetMemberDetailService(ctx context.Context, actor agentAuthService.Actor, id uint) (agentManagementDto.MemberDetailResponse, error) {
	var res agentManagementDto.MemberDetailResponse
	db := database.DBConn.WithContext(ctx)
	m, err := postgres.GetMemberDetailRepository(db, id)
	if err != nil {
		return res, notFoundAsDownline(err)
	}
	if m.AgentID != actor.AgentID {
		ok, err := postgres.IsInDownlineRepository(db, actor.AgentID, m.AgentID)
		if err != nil {
			return res, err
		}
		if !ok {
			return res, apperr.ErrDownlineNotFound
		}
	}
	creator, err := postgres.GetAgentProfileRepository(db, m.AgentID)
	if err != nil {
		return res, err
	}
	creatorStatus, err := chainStatus(db, creator.ID, creator.Status)
	if err != nil {
		return res, err
	}
	extra, err := loadRowExtras(db, nil, []uint{m.ID})
	if err != nil {
		return res, err
	}
	member := string(agentManagementCore.UserTypeMember)
	res = agentManagementDto.MemberDetailResponse{ID: m.ID, Role: member, UserType: member, Username: m.Username, Name: m.Name,
		Phone: stringOrEmpty(m.Phone), Status: string(agentAuthCore.WorstStatus(creatorStatus, m.Status)),
		ParentUsername: creator.Username, Currencies: []string{m.Currency},
		Balances:    BalanceViews([]string{m.Currency}, extra.memberBalance[m.ID]),
		LastLoginAt: optionalTime(m.LastLoginAt), LastLoginIP: stringOrEmpty(m.LastLoginIP), CreatedAt: optionalTime(&m.CreatedAt)}
	show, err := canSeePT(ctx, actor)
	if err != nil {
		return res, err
	}
	if show {
		res.PT = MemberPTViews(extra.memberSettings[m.ID])
	}
	return res, nil
}

// ListCopySourcesService — GET /api/v1/bo/pr/manage/agents/copy-sources (MGMT-35) · ลูกตรงฝั่ง agent ทั้งหมด A→Z
func ListCopySourcesService(ctx context.Context, actor agentAuthService.Actor) ([]agentManagementDto.CopySource, error) {
	db := database.DBConn.WithContext(ctx)
	children, err := postgres.ListAgentDirectChildrenRepository(db, actor.AgentID)
	if err != nil {
		return nil, err
	}
	ids := make([]uint, len(children))
	for i, c := range children {
		ids[i] = c.ID
	}
	settings, err := postgres.ListAgentGameSettingsByIDsRepository(db, ids)
	if err != nil {
		return nil, err
	}
	byID := map[uint][]models.AgentGameSetting{}
	for _, s := range settings {
		byID[s.AgentID] = append(byID[s.AgentID], s)
	}
	out := make([]agentManagementDto.CopySource, len(children))
	for i, c := range children {
		pt, statusGame := AgentPTViews(byID[c.ID])
		out[i] = agentManagementDto.CopySource{ID: c.ID, Username: c.Username,
			UserType: string(agentManagementCore.UserTypeOf(c.Role, c.AgentType)), PT: pt, StatusGame: statusGame}
	}
	return out, nil
}

func notFoundAsDownline(err error) error {
	if errors.Is(err, apperr.ErrNotFound) {
		return apperr.ErrDownlineNotFound
	}
	return err
}
