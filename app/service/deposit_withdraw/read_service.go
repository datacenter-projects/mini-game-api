package depositwithdraw

import (
	"context"
	"strings"

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

// ListDownlinesService — POST /api/v1/bo/pr/deposit-withdraw/downlines/list (DW1 · DW-40 – DW-45 · DW-44A)
// 6 query: ลูกตรง (count + page) · Company หัวสายของแม่ · ยอดของแม่ + ลูกฝั่ง agent ในหน้า (IN ครั้งเดียว — ไม่มี N+1) · ยอดรวมแยก role
func ListDownlinesService(ctx context.Context, actor agentAuthService.Actor, q depositWithdrawDto.DownlinesRequest,
	page utils.Page) ([]depositWithdrawDto.DownlineRow, int64, depositWithdrawDto.DownlinesSummary, error) {
	db := database.DBConn.WithContext(ctx)
	var summary depositWithdrawDto.DownlinesSummary
	rows, total, err := depositWithdrawPostgres.ListDWDownlinesRepository(db, actor.AgentID, q.Keyword, page.Offset(), page.Limit)
	if err != nil {
		return nil, 0, summary, err
	}
	// ลูกทุกคนอยู่สายเดียวกับแม่ · แม่เป็น Superadmin (ไม่อยู่ใต้ Company) = ดูประเภทของ Company ลูกทีละแถว (DW-44)
	parentType, err := agentManagementPostgres.GetChainCompanyTypeRepository(db, actor.AgentID)
	if err != nil {
		return nil, 0, summary, err
	}
	parentSeamless := isSeamless(parentType)

	agentIDs := []uint{actor.AgentID} // แม่ + ลูกฝั่ง agent ในหน้านี้
	for _, r := range rows {
		if r.Role != depositWithdrawCore.RoleMember {
			agentIDs = append(agentIDs, r.ID)
		}
	}
	balances, err := depositWithdrawPostgres.ListAgentBalancesOfRepository(db, agentIDs)
	if err != nil {
		return nil, 0, summary, err
	}
	amounts := make(map[uint]depositWithdrawDto.CurrencyAmounts, len(agentIDs))
	for _, b := range balances {
		if amounts[b.AgentID] == nil {
			amounts[b.AgentID] = newCurrencyAmounts()
		}
		addAmount(amounts[b.AgentID], b.Currency, b.Amount)
	}
	// ฝั่ง Seamless ไม่มียอดเงิน ส่ง 0 ทุกสกุล (MGMT-15 · ACC-19)
	summary.CreditDetail.Currency = newCurrencyAmounts()
	if !parentSeamless && amounts[actor.AgentID] != nil {
		summary.CreditDetail.Currency = amounts[actor.AgentID]
	}

	out := make([]depositWithdrawDto.DownlineRow, 0, len(rows))
	for _, r := range rows {
		row := depositWithdrawDto.DownlineRow{ID: r.ID, Role: r.Role, Username: r.Username, Status: string(r.Status),
			HasBalance: !parentSeamless, Currency: newCurrencyAmounts()}
		if r.Name != nil {
			row.Name = *r.Name
		}
		if r.LastLoginIP != nil {
			row.LastLoginIP = *r.LastLoginIP
		}
		if r.LastLoginAt != nil {
			row.LastLoginAt = formatTime(*r.LastLoginAt)
		}
		if r.Role == string(models.AgentRoleCompany) && isSeamless(r.AgentType) {
			row.HasBalance = false
		}
		switch {
		case !row.HasBalance: // คง 0 ทุกสกุล
		case r.Role == depositWithdrawCore.RoleMember:
			if r.Currency != nil {
				addAmount(row.Currency, *r.Currency, r.Credit)
			}
		case amounts[r.ID] != nil:
			row.Currency = amounts[r.ID]
		}
		out = append(out, row)
	}

	sums, err := depositWithdrawPostgres.SumDWDownlinesRepository(db, actor.AgentID, q.Keyword)
	if err != nil {
		return nil, 0, summary, err
	}
	summary.TotalSum = map[string]depositWithdrawDto.CurrencyAmounts{}
	for _, s := range sums {
		key := strings.ToLower(s.Role) // key ของ total_sum เป็น role ตัวเล็ก (agent / member …) · field role ในแถวยังตัวใหญ่ (maofoy 2026-10-14)
		if summary.TotalSum[key] == nil {
			summary.TotalSum[key] = newCurrencyAmounts()
		}
		if s.Currency != nil {
			addAmount(summary.TotalSum[key], *s.Currency, s.Total)
		}
	}
	return out, total, summary, nil
}

// newCurrencyAmounts — map ครบทุกสกุลที่ระบบรองรับ ค่าเริ่ม 0 (DW-43)
func newCurrencyAmounts() depositWithdrawDto.CurrencyAmounts {
	m := make(depositWithdrawDto.CurrencyAmounts, len(agentManagementCore.Currencies))
	for _, c := range agentManagementCore.Currencies {
		m[c] = 0
	}
	return m
}

// addAmount — บวกยอดเข้าสกุลแล้วปัด 4 ตำแหน่ง (กฎข้อ 9) · สกุลที่ไม่อยู่ในรายการของระบบ = ข้าม
func addAmount(m depositWithdrawDto.CurrencyAmounts, currency string, amount float64) {
	if v, ok := m[currency]; ok {
		m[currency] = utils.Round4(v + amount)
	}
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
