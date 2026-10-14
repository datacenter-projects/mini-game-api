package depositwithdraw

import (
	"context"

	agentManagementCore "app/app/core/agent_management"
	depositWithdrawCore "app/app/core/deposit_withdraw"
	depositWithdrawDto "app/app/internals/backoffice/dto/deposit_withdraw"
	"app/app/models"
	agentManagementPostgres "app/app/repository/postgres/agent_management"
	depositWithdrawPostgres "app/app/repository/postgres/deposit_withdraw"
	agentAuthService "app/app/service/agent_auth"
	"app/pkg/utils"
	"app/platform/database"
)

// ListDownlinesService — POST /api/v1/bo/pr/deposit-withdraw/downlines/list (DW1 · DW-40 – DW-45)
// 4 query ต่อหน้า: ลูกตรง (count + page) · Company หัวสายของแม่ · สกุล + ยอดของลูกฝั่ง agent แบบ IN (ไม่มี N+1)
func ListDownlinesService(ctx context.Context, actor agentAuthService.Actor, q depositWithdrawDto.DownlinesRequest,
	page utils.Page) ([]depositWithdrawDto.DownlineRow, int64, error) {
	db := database.DBConn.WithContext(ctx)
	rows, total, err := depositWithdrawPostgres.ListDWDownlinesRepository(db, actor.AgentID, q.Keyword, page.Offset(), page.Limit)
	if err != nil || len(rows) == 0 {
		return []depositWithdrawDto.DownlineRow{}, total, err
	}
	// ลูกทุกคนอยู่สายเดียวกับแม่ · แม่เป็น Superadmin (ไม่อยู่ใต้ Company) = ดูประเภทของ Company ลูกทีละแถว (DW-44)
	parentType, err := agentManagementPostgres.GetChainCompanyTypeRepository(db, actor.AgentID)
	if err != nil {
		return nil, 0, err
	}
	parentSeamless := isSeamless(parentType)

	var agentIDs []uint
	for _, r := range rows {
		if r.Role != depositWithdrawCore.RoleMember {
			agentIDs = append(agentIDs, r.ID)
		}
	}
	currencies, err := depositWithdrawPostgres.ListAgentCurrenciesOfRepository(db, agentIDs)
	if err != nil {
		return nil, 0, err
	}
	balances, err := depositWithdrawPostgres.ListAgentBalancesOfRepository(db, agentIDs)
	if err != nil {
		return nil, 0, err
	}
	amount := make(map[uint]map[string]float64, len(agentIDs))
	for _, b := range balances {
		if amount[b.AgentID] == nil {
			amount[b.AgentID] = map[string]float64{}
		}
		amount[b.AgentID][b.Currency] = b.Amount
	}
	views := make(map[uint][]depositWithdrawDto.BalanceView, len(agentIDs))
	for _, c := range currencies { // เรียง agent_id, currency · ไม่มีแถวยอด = 0 (DW-43)
		views[c.AgentID] = append(views[c.AgentID], depositWithdrawDto.BalanceView{Currency: c.Currency, Amount: amount[c.AgentID][c.Currency]})
	}

	out := make([]depositWithdrawDto.DownlineRow, 0, len(rows))
	for _, r := range rows {
		row := depositWithdrawDto.DownlineRow{ID: r.ID, Role: r.Role, Username: r.Username, Status: string(r.Status),
			HasBalance: !parentSeamless, Balances: []depositWithdrawDto.BalanceView{}}
		if r.Name != nil {
			row.Name = *r.Name
		}
		if r.Role == depositWithdrawCore.RoleMember {
			if r.Currency != nil {
				row.Balances = []depositWithdrawDto.BalanceView{{Currency: *r.Currency, Amount: r.Credit}}
			}
		} else {
			if r.Role == string(models.AgentRoleCompany) && isSeamless(r.AgentType) {
				row.HasBalance = false
			}
			if v := views[r.ID]; v != nil {
				row.Balances = v
			}
		}
		out = append(out, row)
	}
	return out, total, nil
}

// ListLogsService — POST /api/v1/bo/pr/deposit-withdraw/logs/list (DW3 · DW-46 – DW-49) · เห็นเฉพาะรายการที่ตัวเองเป็นแม่
func ListLogsService(ctx context.Context, actor agentAuthService.Actor, q depositWithdrawDto.LogsRequest,
	page utils.Page) ([]depositWithdrawDto.LogRow, int64, error) {
	f := depositWithdrawPostgres.DWLogFilter{ParentID: actor.AgentID, From: q.From, To: q.To, Type: q.Type, TargetType: q.Owner, TargetID: q.UserID}
	logs, total, err := depositWithdrawPostgres.ListDWLogsRepository(database.DBConn.WithContext(ctx), f, page.Offset(), page.Limit)
	if err != nil {
		return nil, 0, err
	}
	out := make([]depositWithdrawDto.LogRow, 0, len(logs))
	for _, l := range logs {
		out = append(out, depositWithdrawDto.LogRow{ID: l.ID, RequestID: l.RequestID, TransactionType: string(l.TransactionType),
			ActorUsername: l.ActorUsername, Role: l.TargetRole, UserID: l.TargetID, Username: l.TargetUsername,
			Currency: l.Currency, Amount: l.Amount, Note: l.Note, ParentUnlimited: l.ParentUnlimited,
			ParentBalanceBefore: l.ParentBalanceBefore, ParentBalanceAfter: l.ParentBalanceAfter,
			TargetBalanceBefore: l.TargetBalanceBefore, TargetBalanceAfter: l.TargetBalanceAfter,
			CreatedAt: formatTime(l.CreatedAt)})
	}
	return out, total, nil
}

// isSeamless — Company หัวสายเป็น Seamless (MGMT-15 · DW-06) · nil = ไม่อยู่ใต้ Company
func isSeamless(agentType *models.AgentType) bool {
	return agentType != nil && agentManagementCore.IsSeamless(agentManagementCore.UserTypeOf(models.AgentRoleCompany, agentType))
}
