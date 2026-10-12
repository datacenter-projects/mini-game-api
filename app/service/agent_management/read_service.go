package agentmanagement

import (
	"context"
	"encoding/json"
	"errors"

	agentAuthCore "app/app/core/agent_auth"
	agentManagementCore "app/app/core/agent_management"
	agentManagementDto "app/app/internals/backoffice/dto/agent_management"
	"app/app/models"
	agentManagementPostgres "app/app/repository/postgres/agent_management"
	memberManagementPostgres "app/app/repository/postgres/member_management"
	agentAuthService "app/app/service/agent_auth"
	"app/pkg/apperr"
	"app/pkg/utils"
	"app/platform/database"

	"gorm.io/gorm"
)

// ListDownlinesService — POST /api/v1/bo/pr/manage/downlines/list (MGMT-26 – MGMT-28 · lead E2)
// parent_id ไม่ส่ง = ตัวเอง · อื่นต้องเป็นบัญชีฝั่ง agent ในสายล่าง (402402)
// keyword ว่าง = ลูกตรงของ parent_id · ส่ง = ค้น username ทุกชั้นใต้ parent_id · ข้อมูลประกอบอ่านทีละหน้าแบบ batch
func ListDownlinesService(ctx context.Context, actor agentAuthService.Actor, q agentManagementDto.DownlinesRequest,
	page utils.Page) ([]agentManagementDto.DownlineRow, int64, error) {
	db := database.DBConn.WithContext(ctx)
	parentID := actor.AgentID
	if q.ParentID != 0 && q.ParentID != actor.AgentID {
		ok, err := agentManagementPostgres.IsInDownlineRepository(db, actor.AgentID, q.ParentID)
		if err != nil {
			return nil, 0, err
		}
		if !ok {
			return nil, 0, apperr.ErrDownlineNotFound
		}
		parentID = q.ParentID
	}
	parent, err := agentManagementPostgres.GetAgentProfileRepository(db, parentID)
	if err != nil {
		return nil, 0, NotFoundAsDownline(err)
	}
	parentStatus, err := ChainStatus(db, parent.ID, parent.Status)
	if err != nil {
		return nil, 0, err
	}

	type hit struct {
		row            agentManagementPostgres.DownlineRow
		status         models.AgentStatus
		parentUsername string
	}
	var hits []hit
	var total int64
	if q.Keyword == "" {
		rows, n, err := agentManagementPostgres.ListDownlinesRepository(db, parentID, "", page.Offset(), page.Limit)
		if err != nil {
			return nil, 0, err
		}
		for _, r := range rows {
			hits = append(hits, hit{r, agentAuthCore.WorstStatus(parentStatus, r.Status), parent.Username})
		}
		total = n
	} else {
		rows, n, err := agentManagementPostgres.SearchDownlinesRepository(db, parentID, parentStatus, q.Keyword, page.Offset(), page.Limit)
		if err != nil {
			return nil, 0, err
		}
		for _, r := range rows {
			hits = append(hits, hit{r.DownlineRow, r.EffectiveStatus, r.ParentUsername})
		}
		total = n
	}

	rows := make([]agentManagementPostgres.DownlineRow, len(hits))
	for i, h := range hits {
		rows[i] = h.row
	}
	agentIDs, memberIDs := splitIDs(rows)
	extra, err := loadRowExtras(db, agentIDs, memberIDs)
	if err != nil {
		return nil, 0, err
	}
	out := make([]agentManagementDto.DownlineRow, len(hits))
	for i, h := range hits {
		out[i] = downlineRow(h.row, h.status, h.parentUsername, extra)
	}
	return out, total, nil
}

// splitIDs — แยก id ฝั่ง agent / Member ของแถวในหน้า
func splitIDs(rows []agentManagementPostgres.DownlineRow) (agentIDs, memberIDs []uint) {
	for _, r := range rows {
		if r.IsMember {
			memberIDs = append(memberIDs, r.ID)
		} else {
			agentIDs = append(agentIDs, r.ID)
		}
	}
	return agentIDs, memberIDs
}

// downlineRow — 1 แถวของรายชื่อ / ผลค้น (MGMT-28) · status = สถานะที่ใช้งานจริง · pt แสดงเสมอ · status_game เฉพาะแถว agent
func downlineRow(r agentManagementPostgres.DownlineRow, status models.AgentStatus, parentUsername string, extra rowExtras) agentManagementDto.DownlineRow {
	row := agentManagementDto.DownlineRow{ID: r.ID, Username: r.Username, Name: StringOrEmpty(r.Name), PhoneCountryCode: StringOrEmpty(r.PhoneCountryCode), Phone: StringOrEmpty(r.Phone),
		Status: string(status), ParentUsername: parentUsername, LastLoginAt: OptionalTime(r.LastLoginAt), LastLoginIP: StringOrEmpty(r.LastLoginIP), CreatedAt: OptionalTime(&r.CreatedAt)}
	if r.IsMember {
		row.Role, row.UserType = string(agentManagementCore.UserTypeMember), string(agentManagementCore.UserTypeMember)
		row.Balances = BalanceViews([]string{StringOrEmpty(r.Currency)}, extra.memberBalance[r.ID])
		row.PT = MemberPTViews(extra.memberSettings[r.ID])
		return row
	}
	row.Role, row.UserType = string(r.Role), string(agentManagementCore.UserTypeOf(r.Role, r.AgentType))
	row.Balances = BalanceViews(extra.agentCurrencies[r.ID], extra.agentBalance[r.ID])
	row.PT, row.StatusGame = AgentPTViews(extra.agentSettings[r.ID])
	return row
}

// rowExtras — สกุล ยอด และค่าหุ้นส่วนของทุกแถวในหน้า (query ละครั้ง ไม่ N+1 — กฎข้อ 23)
type rowExtras struct {
	agentCurrencies map[uint][]string
	agentBalance    map[uint]map[string]float64
	agentSettings   map[uint][]models.AgentGameSetting
	memberBalance   map[uint]map[string]float64
	memberSettings  map[uint][]models.UserMemberGameSetting
}

func loadRowExtras(db *gorm.DB, agentIDs, memberIDs []uint) (rowExtras, error) {
	e := rowExtras{map[uint][]string{}, map[uint]map[string]float64{}, map[uint][]models.AgentGameSetting{},
		map[uint]map[string]float64{}, map[uint][]models.UserMemberGameSetting{}}
	curs, err := agentManagementPostgres.ListAgentCurrenciesByIDsRepository(db, agentIDs)
	if err != nil {
		return e, err
	}
	for _, c := range curs {
		e.agentCurrencies[c.AgentID] = append(e.agentCurrencies[c.AgentID], c.Currency)
	}
	abs, err := agentManagementPostgres.ListAgentBalancesByIDsRepository(db, agentIDs)
	if err != nil {
		return e, err
	}
	for _, b := range abs {
		if e.agentBalance[b.AgentID] == nil {
			e.agentBalance[b.AgentID] = map[string]float64{}
		}
		e.agentBalance[b.AgentID][b.Currency] = b.Amount
	}
	ags, err := agentManagementPostgres.ListAgentGameSettingsByIDsRepository(db, agentIDs)
	if err != nil {
		return e, err
	}
	for _, s := range ags {
		e.agentSettings[s.AgentID] = append(e.agentSettings[s.AgentID], s)
	}
	mbs, err := memberManagementPostgres.ListUserMemberCreditsByIDsRepository(db, memberIDs) // ยอดของ Member = user_members.credit
	if err != nil {
		return e, err
	}
	for _, m := range mbs {
		e.memberBalance[m.ID] = map[string]float64{m.Currency: m.Credit}
	}
	mgs, err := memberManagementPostgres.ListUserMemberGameSettingsByIDsRepository(db, memberIDs)
	if err != nil {
		return e, err
	}
	for _, s := range mgs {
		e.memberSettings[s.UserMemberID] = append(e.memberSettings[s.UserMemberID], s)
	}
	return e, nil
}

// GetAgentDetailService — POST /api/v1/bo/pr/manage/agents/detail/get (MGMT-29) · เฉพาะบัญชีในสายล่าง (402402)
func GetAgentDetailService(ctx context.Context, actor agentAuthService.Actor, id uint) (agentManagementDto.AgentDetailResponse, error) {
	var res agentManagementDto.AgentDetailResponse
	db := database.DBConn.WithContext(ctx)
	ok, err := agentManagementPostgres.IsInDownlineRepository(db, actor.AgentID, id)
	if err != nil {
		return res, err
	}
	if !ok {
		return res, apperr.ErrDownlineNotFound
	}
	a, err := agentManagementPostgres.GetAgentDetailRepository(db, id)
	if err != nil {
		return res, NotFoundAsDownline(err)
	}
	parent, err := agentManagementPostgres.GetAgentProfileRepository(db, *a.ParentID) // อยู่ในสายล่าง = มี parent เสมอ
	if err != nil {
		return res, err
	}
	status, err := ChainStatus(db, a.ID, a.Status)
	if err != nil {
		return res, err
	}
	extra, err := loadRowExtras(db, []uint{a.ID}, nil)
	if err != nil {
		return res, err
	}
	pt, statusGame := AgentPTViews(extra.agentSettings[a.ID])
	effective, err := effectiveStatusGame(db, a.Cnf, statusGame)
	if err != nil {
		return res, err
	}
	res = agentManagementDto.AgentDetailResponse{ID: a.ID, Role: string(a.Role),
		UserType: string(agentManagementCore.UserTypeOf(a.Role, a.AgentType)), Username: a.Username,
		Name: StringOrEmpty(a.Name), PhoneCountryCode: StringOrEmpty(a.PhoneCountryCode), Phone: StringOrEmpty(a.Phone), Status: string(status), ParentUsername: parent.Username,
		Currencies: append([]string{}, extra.agentCurrencies[a.ID]...),
		Balances:   BalanceViews(extra.agentCurrencies[a.ID], extra.agentBalance[a.ID]),
		PT:         pt, StatusGame: statusGame, StatusGameEffective: effective, PasscodeSet: a.PasscodeHash != nil,
		LastLoginAt: OptionalTime(a.LastLoginAt), LastLoginIP: StringOrEmpty(a.LastLoginIP), CreatedAt: OptionalTime(&a.CreatedAt)}
	return res, nil
}

// effectiveStatusGame — เกมเปิดจริงเมื่อบัญชีและหัวสายทุกชั้นเปิด (MGMT-20 · lead E3-1) · หัวสายอ่านจาก cnf (MGMT-61) query เดียว
func effectiveStatusGame(db *gorm.DB, cnf string, own map[string]bool) (map[string]bool, error) {
	var chain agentManagementCore.Chain
	if err := json.Unmarshal([]byte(cnf), &chain); err != nil {
		return nil, apperr.ErrInternal.Wrap(err)
	}
	ids := make([]uint, len(chain.Parent))
	for i, p := range chain.Parent {
		ids[i] = p.ID
	}
	upline, err := agentManagementPostgres.ListAgentGameSettingsByIDsRepository(db, ids)
	if err != nil {
		return nil, err
	}
	byGame := map[string][]bool{}
	for _, s := range upline {
		byGame[s.GameCode] = append(byGame[s.GameCode], s.StatusGame)
	}
	out := make(map[string]bool, len(own))
	for game, on := range own {
		out[game] = agentManagementCore.IsGameOpen(append([]bool{on}, byGame[game]...))
	}
	return out, nil
}

// AgentListMax — agents/list ส่งไม่เกินกี่รายการ (lead E6b)
const AgentListMax = 500

// ListAgentsService — POST /api/v1/bo/pr/manage/agents/list (MGMT-35 · lead E6) · ลูกตรงฝั่ง agent A→Z ไว้ทำ dropdown
func ListAgentsService(ctx context.Context, actor agentAuthService.Actor, req agentManagementDto.AgentListRequest) ([]agentManagementDto.AgentListRow, error) {
	db := database.DBConn.WithContext(ctx)
	me, err := agentManagementPostgres.GetAgentProfileRepository(db, actor.AgentID)
	if err != nil {
		return nil, err
	}
	myStatus, err := ChainStatus(db, me.ID, me.Status)
	if err != nil {
		return nil, err
	}
	children, err := agentManagementPostgres.ListAgentDirectChildrenRepository(db, actor.AgentID, req.Keyword, AgentListMax)
	if err != nil {
		return nil, err
	}
	out := make([]agentManagementDto.AgentListRow, len(children))
	for i, c := range children {
		out[i] = agentManagementDto.AgentListRow{ID: c.ID, Username: c.Username, Name: StringOrEmpty(c.Name),
			UserType: string(agentManagementCore.UserTypeOf(c.Role, c.AgentType)), Status: string(agentAuthCore.WorstStatus(myStatus, c.Status))}
	}
	return out, nil
}

func NotFoundAsDownline(err error) error {
	if errors.Is(err, apperr.ErrNotFound) {
		return apperr.ErrDownlineNotFound
	}
	return err
}
