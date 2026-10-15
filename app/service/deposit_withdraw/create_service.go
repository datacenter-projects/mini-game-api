// Package depositwithdraw คือ business logic ของเส้น /deposit-withdraw/* — spec: docs/modules/deposit_withdraw.md
package depositwithdraw

import (
	"context"
	"errors"
	"slices"
	"time"

	depositWithdrawCore "app/app/core/deposit_withdraw"
	depositWithdrawDto "app/app/internals/backoffice/dto/deposit_withdraw"
	"app/app/models"
	agentManagementPostgres "app/app/repository/postgres/agent_management"
	depositWithdrawPostgres "app/app/repository/postgres/deposit_withdraw"
	depositWithdrawRedis "app/app/repository/redis/deposit_withdraw"
	agentAuthService "app/app/service/agent_auth"
	"app/pkg/apperr"
	"app/pkg/utils"
	"app/platform/database"
	"app/platform/logger"

	"gorm.io/gorm"
)

const (
	lockTTL       = 10 * time.Second // DW-20 · lead DQ2
	txTimeout     = 8 * time.Second  // DW-23 ต่ำกว่า lockTTL (lead ยืนยัน 2026-10-14)
	releaseWait   = 2 * time.Second  // เวลารอปลด lock หลัง request จบ
	tokenAlphabet = "0123456789abcdefghijklmnopqrstuvwxyz"
	tokenLen      = 32
)

// CheckNotAdminService — ใช้โดย middleware deposit_withdraw.RejectAdmin (DW-03) · ADMIN = 404305
// ต้องอยู่ก่อน RequirePermission (ADMIN ไม่มีเมนูใดเลย — ถ้าเช็คสิทธิ์ก่อนจะได้ 402303)
func CheckNotAdminService(actor agentAuthService.Actor) error {
	if actor.Role == models.AgentRoleAdmin {
		return apperr.ErrDWAdminNotAllowed
	}
	return nil
}

func errRequestIDUsed() error {
	return apperr.ErrValidation.WithMessage("request_id นี้ถูกใช้แล้ว", "request_id has already been used")
}

// CreateDepositWithdrawService — POST /api/v1/bo/pr/deposit-withdraw/create (DW2 · DW-01 – DW-34)
// Redis lock ที่บัญชีปลายทาง → tx (request_id → ลูก → สถานะ / สกุล → upsert → lock ยอด → อัปเดต → log → ledger → create_requests) → ปลด lock
func CreateDepositWithdrawService(ctx context.Context, actor agentAuthService.Actor, req depositWithdrawDto.CreateRequest,
	meta agentAuthService.RequestMeta) (depositWithdrawDto.CreateResponse, error) {
	var res depositWithdrawDto.CreateResponse
	token, err := utils.RandomString(tokenAlphabet, tokenLen)
	if err != nil {
		return res, err
	}
	got, err := depositWithdrawRedis.AcquireDWLockRepository(ctx, req.Owner, req.UserID, token, lockTTL)
	if err != nil {
		return res, apperr.ErrDWLockUnavailable.Wrap(err) // DW-21 fail closed
	}
	if !got {
		return res, apperr.ErrDWInProgress
	}
	defer releaseLock(ctx, req.Owner, req.UserID, token)

	txCtx, cancel := context.WithTimeout(ctx, txTimeout)
	defer cancel()
	err = database.DBConn.WithContext(txCtx).Transaction(func(tx *gorm.DB) error {
		var err error
		res, err = createInTx(tx, actor, req, meta)
		return err
	})
	if errors.Is(err, gorm.ErrDuplicatedKey) { // DW-27: request_id ชนกับรายการที่ commit ไปแล้ว — อ่านผลเดิม
		return replayAfterConflict(ctx, actor, req)
	}
	return res, err
}

// releaseLock — compare-and-delete (DW-22) · ปลดไม่ได้ = log แล้วปล่อยให้ TTL หมดเอง ไม่ทำให้ request ล้ม
func releaseLock(ctx context.Context, owner models.BalanceOwnerType, id uint, token string) {
	rctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), releaseWait)
	defer cancel()
	if err := depositWithdrawRedis.ReleaseDWLockRepository(rctx, owner, id, token); err != nil {
		logger.Ctx(ctx).Warnw("release deposit_withdraw lock failed", "owner_type", owner, "owner_id", id, "error", err)
	}
}

// target — บัญชีปลายทางที่ผ่าน DW-01 / DW-02 แล้ว
type target struct {
	owner      models.BalanceOwnerType
	id         uint
	role       string
	username   string
	status     models.AgentStatus
	chainID    uint // บัญชีที่ใช้หา Company หัวสาย: agent = ตัวเอง · Member = ผู้สร้าง
	currencies []string
}

func createInTx(tx *gorm.DB, actor agentAuthService.Actor, req depositWithdrawDto.CreateRequest,
	meta agentAuthService.RequestMeta) (depositWithdrawDto.CreateResponse, error) {
	var res depositWithdrawDto.CreateResponse
	// DW-24: request_id ก่อนอย่างอื่น · key เดียวกับ FindReplay ของ agents / members create
	if err := agentManagementPostgres.AdvisoryXactLockRepository(tx, "create_request:"+req.RequestID); err != nil {
		return res, err
	}
	if r, replay, err := findReplay(tx, actor, req); err != nil || replay {
		return r, err
	}

	parent, err := agentManagementPostgres.GetAgentProfileRepository(tx, actor.AgentID)
	if err != nil {
		return res, err
	}
	t, err := loadTarget(tx, parent.ID, req)
	if err != nil {
		return res, err
	}
	if err := checkTarget(tx, t, req, parent.ID); err != nil {
		return res, err
	}

	// DW-31 / DW-32: upsert → agent_balances FOR UPDATE เรียง agent_id → user_members FOR NO KEY UPDATE
	// แม่ทุกประเภทรวม Superadmin มียอดใน agent_balances (DW-12 / DW-14 · lead 2026-10-15)
	now := time.Now().Truncate(time.Microsecond) // เท่าความละเอียดของ timestamptz — ส่งซ้ำได้ created_at เดิม (DW-25)
	agentIDs := []uint{parent.ID}
	if t.owner == models.BalanceOwnerAgent {
		agentIDs = append(agentIDs, t.id)
	}
	slices.Sort(agentIDs)
	if err := depositWithdrawPostgres.EnsureAgentBalanceRowsRepository(tx, agentIDs, req.Currency, now); err != nil {
		return res, err
	}
	locked, err := depositWithdrawPostgres.LockDWAgentBalancesRepository(tx, agentIDs, req.Currency)
	if err != nil {
		return res, err
	}
	have := make(map[uint]float64, len(locked))
	for _, b := range locked {
		have[b.AgentID] = b.Amount
	}
	targetBefore := have[t.id]
	if t.owner == models.BalanceOwnerMember {
		m, err := depositWithdrawPostgres.LockDWMemberRepository(tx, t.id)
		if err != nil || m.AgentID != parent.ID { // ย้ายสาย / ถูกลบระหว่างตรวจ
			return res, notFoundAsTarget(err)
		}
		if req.Type == models.DWDeposit && !depositWithdrawCore.CanDeposit(m.Status) { // สถานะหลัง lock (DW-05)
			return res, apperr.ErrDWTargetInactive
		}
		targetBefore = m.Credit
	}

	bal, v := depositWithdrawCore.Settle(req.Type, have[parent.ID], targetBefore, req.Amount)
	switch v {
	case depositWithdrawCore.ParentInsufficient:
		return res, apperr.ErrDWParentInsufficient
	case depositWithdrawCore.TargetInsufficient:
		return res, apperr.ErrDWTargetInsufficient
	}
	if err := applyBalances(tx, req, t, parent.ID, bal, now); err != nil {
		return res, err
	}

	l := models.DepositWithdrawLog{RequestID: req.RequestID, TransactionType: req.Type,
		ActorType: actor.AccountType, ActorID: actor.AccountID(), ActorUsername: actor.Username,
		ParentID: parent.ID, ParentUsername: parent.Username,
		TargetType: t.owner, TargetID: t.id, TargetRole: t.role, TargetUsername: t.username,
		Currency: req.Currency, Amount: req.Amount, // ParentUnlimited = false เสมอ (เลิกใช้ 2026-10-15)
		ParentBalanceBefore: bal.ParentBefore, ParentBalanceAfter: bal.ParentAfter,
		TargetBalanceBefore: bal.TargetBefore, TargetBalanceAfter: bal.TargetAfter,
		Note: req.Note, IP: meta.IP, CreatedAt: now}
	if err := depositWithdrawPostgres.CreateDWLogRepository(tx, &l); err != nil {
		return res, err
	}
	if err := agentManagementPostgres.CreateBalanceLedgerRepository(tx, ledgerRows(l, actor)); err != nil {
		return res, err
	}
	if err := agentManagementPostgres.CreateCreateRequestRepository(tx, &models.CreateRequest{RequestID: req.RequestID,
		CreatorType: actor.AccountType, CreatorID: actor.AccountID(), TargetType: models.CreateRequestTargetDW, TargetID: l.ID, CreatedAt: now}); err != nil {
		return res, err
	}
	return createResponse(l), nil
}

// loadTarget — ลูกตรงของแม่ (DW-01) ที่ role ตรงกับในตาราง (DW-02) · ไม่ใช่ทุกกรณี = 404401 ข้อความเดียวกัน
func loadTarget(tx *gorm.DB, parentID uint, req depositWithdrawDto.CreateRequest) (target, error) {
	if req.Owner == models.BalanceOwnerMember {
		m, err := depositWithdrawPostgres.GetDWMemberRepository(tx, req.UserID)
		if err != nil || m.AgentID != parentID {
			return target{}, notFoundAsTarget(err)
		}
		return target{owner: models.BalanceOwnerMember, id: m.ID, role: depositWithdrawCore.RoleMember, username: m.Username,
			status: m.Status, chainID: m.AgentID, currencies: []string{m.Currency}}, nil
	}
	a, err := agentManagementPostgres.GetAgentProfileRepository(tx, req.UserID)
	if err != nil || a.ParentID == nil || *a.ParentID != parentID || string(a.Role) != req.Role {
		return target{}, notFoundAsTarget(err)
	}
	currencies, err := agentManagementPostgres.ListAgentCurrenciesRepository(tx, a.ID)
	if err != nil {
		return target{}, err
	}
	return target{owner: models.BalanceOwnerAgent, id: a.ID, role: string(a.Role), username: a.Username,
		status: a.Status, chainID: a.ID, currencies: currencies}, nil
}

// notFoundAsTarget — ไม่พบ / ไม่ใช่ลูกตรง = 404401 · error อื่นของ DB ส่งต่อ
func notFoundAsTarget(err error) error {
	if err == nil || errors.Is(err, apperr.ErrNotFound) {
		return apperr.ErrDWTargetNotFound
	}
	return err
}

// checkTarget — Seamless (DW-06) → สถานะเฉพาะฝาก (DW-05) → สกุลของลูกและของแม่ (DW-11)
func checkTarget(tx *gorm.DB, t target, req depositWithdrawDto.CreateRequest, parentID uint) error {
	agentType, err := agentManagementPostgres.GetChainCompanyTypeRepository(tx, t.chainID)
	if err != nil {
		return err
	}
	if isSeamless(agentType) {
		return apperr.ErrDWSeamless
	}
	if req.Type == models.DWDeposit && !depositWithdrawCore.CanDeposit(t.status) {
		return apperr.ErrDWTargetInactive
	}
	if !slices.Contains(t.currencies, req.Currency) {
		return apperr.ErrDWCurrency
	}
	parentCurrencies, err := agentManagementPostgres.ListAgentCurrenciesRepository(tx, parentID)
	if err != nil {
		return err
	}
	if !slices.Contains(parentCurrencies, req.Currency) {
		return apperr.ErrDWCurrency
	}
	return nil
}

// applyBalances — เขียนยอดใหม่ทั้ง 2 ฝั่ง · CHECK ยอดติดลบ (DW-34) แปลงตามทิศ: ฝาก = แม่ไม่พอ · ถอน = ลูกไม่พอ
func applyBalances(tx *gorm.DB, req depositWithdrawDto.CreateRequest, t target, parentID uint,
	bal depositWithdrawCore.Balances, now time.Time) error {
	err := depositWithdrawPostgres.UpdateDWAgentBalanceRepository(tx, parentID, req.Currency, bal.ParentAfter, now)
	if err == nil {
		if t.owner == models.BalanceOwnerMember {
			err = depositWithdrawPostgres.UpdateDWMemberCreditRepository(tx, t.id, bal.TargetAfter, now)
		} else {
			err = depositWithdrawPostgres.UpdateDWAgentBalanceRepository(tx, t.id, req.Currency, bal.TargetAfter, now)
		}
	}
	if errors.Is(err, depositWithdrawPostgres.ErrBalanceNegative) {
		if req.Type == models.DWDeposit {
			return apperr.ErrDWParentInsufficient
		}
		return apperr.ErrDWTargetInsufficient
	}
	return err
}

// ledgerRows — balance_ledger ของรายการ (DW-33) · แม่ 1 แถว + ลูก 1 แถว
func ledgerRows(l models.DepositWithdrawLog, actor agentAuthService.Actor) []models.BalanceLedger {
	parentReason, targetReason := depositWithdrawCore.LedgerReasons(l.TransactionType)
	parentAmount, targetAmount := depositWithdrawCore.SignedAmounts(l.TransactionType, l.Amount)
	refType, refID := models.LedgerRefDW, l.ID
	row := func(owner models.BalanceOwnerType, id uint, amount, after float64, reason models.LedgerReason) models.BalanceLedger {
		return models.BalanceLedger{OwnerType: owner, OwnerID: id, Currency: l.Currency, Amount: amount, BalanceAfter: after,
			Reason: reason, RefType: &refType, RefID: &refID, RequestID: l.RequestID,
			ActorType: actor.AccountType, ActorID: actor.AccountID(), CreatedAt: l.CreatedAt}
	}
	return []models.BalanceLedger{
		row(models.BalanceOwnerAgent, l.ParentID, parentAmount, l.ParentBalanceAfter, parentReason),
		row(l.TargetType, l.TargetID, targetAmount, l.TargetBalanceAfter, targetReason),
	}
}

// findReplay — request_id ที่ลงทะเบียนแล้วใน create_requests (DW-25 / DW-26)
// ของผู้เรียกเดิม + เส้นนี้ + ค่าเดิมทุก field = คืนผลเดิม · อื่นๆ = 422
func findReplay(db *gorm.DB, actor agentAuthService.Actor, req depositWithdrawDto.CreateRequest) (depositWithdrawDto.CreateResponse, bool, error) {
	var res depositWithdrawDto.CreateResponse
	r, err := agentManagementPostgres.GetCreateRequestRepository(db, req.RequestID)
	if errors.Is(err, apperr.ErrNotFound) {
		return res, false, nil
	}
	if err != nil {
		return res, false, err
	}
	if r.TargetType != models.CreateRequestTargetDW || r.CreatorType != actor.AccountType || r.CreatorID != actor.AccountID() {
		return res, false, errRequestIDUsed()
	}
	l, err := depositWithdrawPostgres.GetDWLogRepository(db, r.TargetID)
	if err != nil {
		return res, false, err
	}
	if l.TargetRole != req.Role || l.TargetID != req.UserID || l.Currency != req.Currency || l.Amount != req.Amount ||
		l.TransactionType != req.Type || l.Note != req.Note {
		return res, false, errRequestIDUsed()
	}
	return createResponse(l), true, nil
}

// replayAfterConflict — insert ชน unique ของ request_id (DW-27) · tx เดิม rollback แล้ว อ่านผลของรายการที่ชนะ
func replayAfterConflict(ctx context.Context, actor agentAuthService.Actor, req depositWithdrawDto.CreateRequest) (depositWithdrawDto.CreateResponse, error) {
	res, replay, err := findReplay(database.DBConn.WithContext(ctx), actor, req)
	if err != nil {
		return res, err
	}
	if !replay {
		return res, errRequestIDUsed()
	}
	return res, nil
}

func createResponse(l models.DepositWithdrawLog) depositWithdrawDto.CreateResponse {
	return depositWithdrawDto.CreateResponse{ID: l.ID, RequestID: l.RequestID, TransactionType: string(l.TransactionType),
		Role: l.TargetRole, UserID: l.TargetID, Username: l.TargetUsername, Currency: l.Currency, Amount: l.Amount, Note: l.Note,
		ParentUnlimited: l.ParentUnlimited, ParentBalanceAfter: l.ParentBalanceAfter, TargetBalanceAfter: l.TargetBalanceAfter,
		CreatedAt: formatTime(l.CreatedAt)}
}

func formatTime(t time.Time) string { return t.Format(time.RFC3339Nano) }
