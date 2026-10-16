package depositwithdraw

import (
	"context"
	"errors"
	"strings"

	agentManagementCore "app/app/core/agent_management"
	depositWithdrawCore "app/app/core/deposit_withdraw"
	depositWithdrawDto "app/app/internals/backoffice/dto/deposit_withdraw"
	depositWithdrawPostgres "app/app/repository/postgres/deposit_withdraw"
	agentAuthService "app/app/service/agent_auth"
	"app/pkg/apperr"
	"app/pkg/utils"
	"app/platform/database"
)

// ListDownlinesService — POST /api/v1/bo/pr/deposit-withdraw/downlines/list (DW1 · DW-40 – DW-45 · DW-44A)
// 5 query: ลูกตรง (count + page) · ยอดของแม่ + ลูกฝั่ง agent ในหน้า (IN ครั้งเดียว — ไม่มี N+1) · ยอดรวมแยก role
// สาย Seamless มียอดแล้ว (DW-06 / MGMT-15 แก้ 2026-10-15) — ทุกแถว has_balance = true · แสดงยอดจริง
func ListDownlinesService(ctx context.Context, actor agentAuthService.Actor, q depositWithdrawDto.DownlinesRequest,
	page utils.Page) ([]depositWithdrawDto.DownlineRow, int64, depositWithdrawDto.DownlinesSummary, error) {
	db := database.DBConn.WithContext(ctx)
	var summary depositWithdrawDto.DownlinesSummary
	rows, total, err := depositWithdrawPostgres.ListDWDownlinesRepository(db, actor.AgentID, q.Keyword, page.Offset(), page.Limit)
	if err != nil {
		return nil, 0, summary, err
	}

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
	summary.CreditDetail.Currency = newCurrencyAmounts()
	if amounts[actor.AgentID] != nil {
		summary.CreditDetail.Currency = amounts[actor.AgentID]
	}

	out := make([]depositWithdrawDto.DownlineRow, 0, len(rows))
	for _, r := range rows {
		row := depositWithdrawDto.DownlineRow{ID: r.ID, Role: r.Role, Username: r.Username, Status: string(r.Status),
			HasBalance: true, Currency: newCurrencyAmounts()}
		if r.Name != nil {
			row.Name = *r.Name
		}
		if r.LastLoginIP != nil {
			row.LastLoginIP = *r.LastLoginIP
		}
		if r.LastLoginAt != nil {
			row.LastLoginAt = formatTime(*r.LastLoginAt)
		}
		switch {
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
		out = append(out, depositWithdrawDto.LogRow{ID: l.ID, RequestID: l.RequestID, TransactionType: string(l.TransactionType), Status: string(l.Status),
			ActorUsername: l.ActorUsername, Role: l.TargetRole, UserID: l.TargetID, Username: l.TargetUsername,
			Currency: l.Currency, Amount: l.Amount, Note: l.Note, IP: l.IP, ParentUnlimited: l.ParentUnlimited,
			ParentBalanceBefore: l.ParentBalanceBefore, ParentBalanceAfter: l.ParentBalanceAfter,
			TargetBalanceBefore: l.TargetBalanceBefore, TargetBalanceAfter: l.TargetBalanceAfter,
			CreatedAt: formatTime(l.CreatedAt)})
	}
	return out, total, nil
}

// ListHistoryService — POST /api/v1/bo/pr/deposit-withdraw/history/list (DW4 · DW-55 – DW-60)
// หา username ใต้ตัวเองทุกชั้น (ไม่เจอ / นอกสาย / ตัวเอง = 404402) → log ที่ทำให้บัญชีนั้น · แสดงคนกดจริง
func ListHistoryService(ctx context.Context, actor agentAuthService.Actor, q depositWithdrawDto.HistoryRequest,
	page utils.Page) ([]depositWithdrawDto.HistoryRow, int64, error) {
	db := database.DBConn.WithContext(ctx)
	owner, id, err := depositWithdrawPostgres.FindDWDownlineAccountRepository(db, actor.AgentID, q.Username)
	if errors.Is(err, apperr.ErrNotFound) {
		return nil, 0, apperr.ErrDWAccountNotFound
	}
	if err != nil {
		return nil, 0, err
	}
	logs, total, err := depositWithdrawPostgres.ListDWHistoryRepository(db, owner, id, q.Start, q.End, page.Offset(), page.Limit)
	if err != nil {
		return nil, 0, err
	}
	out := make([]depositWithdrawDto.HistoryRow, 0, len(logs))
	for _, l := range logs {
		out = append(out, depositWithdrawDto.HistoryRow{ID: l.ID, AgentID: l.TargetID, AgentUsername: l.TargetUsername, Role: l.TargetRole,
			RequestBy: l.ActorID, RequestType: string(l.ActorType), RequestUsername: l.ActorUsername,
			Currency: l.Currency, TransactionType: string(l.TransactionType), Status: string(l.Status), Amount: l.Amount, Remark: l.Note,
			CreatedAt: formatTime(l.CreatedAt)})
	}
	return out, total, nil
}
