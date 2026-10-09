package agentmanagement

import (
	"encoding/json"
	"errors"
	"sort"
	"time"

	agentManagementCore "app/app/core/agent_management"
	"app/app/models"
	agentManagementPostgres "app/app/repository/postgres/agent_management"
	memberManagementPostgres "app/app/repository/postgres/member_management"
	agentAuthService "app/app/service/agent_auth"
	"app/pkg/apperr"
	"app/pkg/utils"

	"gorm.io/gorm"
)

// Creator — ผู้สร้างบัญชี = บัญชีหลักที่ request ทำงานในนาม (sub = เจ้าของ — AUTH-26)
type Creator struct {
	Agent       models.UserAgent
	UserType    agentManagementCore.UserType
	CompanyType agentManagementCore.UserType // Company หัวสาย รวมตัวเอง · "" = ไม่อยู่ใต้ Company (Superadmin)
	Currencies  []string
	ReceivedBP  map[agentManagementCore.PTGroup]int // ค่าที่ผู้สร้างได้รับต่อกลุ่ม (MGMT-22)
	Chain       agentManagementCore.Chain           // สายชั้นบนของผู้สร้าง (MGMT-61)
}

func LoadCreator(db *gorm.DB, actor agentAuthService.Actor) (Creator, error) {
	var c Creator
	a, err := agentManagementPostgres.GetAgentProfileRepository(db, actor.AgentID)
	if err != nil {
		return c, err
	}
	c.Agent = a
	c.UserType = agentManagementCore.UserTypeOf(a.Role, a.AgentType)
	cnf, err := agentManagementPostgres.GetAgentChainRepository(db, a.ID)
	if err != nil {
		return c, err
	}
	if err := json.Unmarshal([]byte(cnf), &c.Chain); err != nil {
		return c, err
	}
	companyAgentType, err := agentManagementPostgres.GetChainCompanyTypeRepository(db, a.ID)
	if err != nil {
		return c, err
	}
	if companyAgentType != nil {
		c.CompanyType = agentManagementCore.UserTypeOf(models.AgentRoleCompany, companyAgentType)
	}
	if c.Currencies, err = agentManagementPostgres.ListAgentCurrenciesRepository(db, a.ID); err != nil {
		return c, err
	}
	// FOR SHARE: กันชั้นบนลดค่าที่ให้ผู้สร้างระหว่างสร้างลูก (MGMT-24)
	settings, err := agentManagementPostgres.LockAgentGameSettingsRepository(db, []uint{a.ID}, "SHARE")
	if err != nil {
		return c, err
	}
	c.ReceivedBP = map[agentManagementCore.PTGroup]int{}
	for _, s := range settings { // ค่าของเกมในกลุ่มเท่ากันเสมอ (MGMT-16)
		if g, ok := agentManagementCore.GroupOfGame(s.GameCode); ok {
			c.ReceivedBP[g] = s.PTFromParentBP
		}
	}
	return c, nil
}

// newAccountSeamless — บัญชีใหม่อยู่ฝั่ง Seamless ไหม (MGMT-15) · ใต้ Superadmin ตัดสินจากประเภทของ Company ใหม่
func (c Creator) newAccountSeamless(newType agentManagementCore.UserType) bool {
	if c.UserType == agentManagementCore.UserTypeSuperadmin {
		return agentManagementCore.IsSeamless(newType)
	}
	return agentManagementCore.IsSeamless(c.CompanyType)
}

// CheckInitialBalance — ตรวจยอดเงินตั้งต้นก่อนเปิด tx (MGMT-15, MGMT-15A, MGMT-51)
func CheckInitialBalance(amounts map[string]int64, seamless bool, accountCurrencies []string) error {
	if len(amounts) == 0 {
		return nil
	}
	if seamless {
		return apperr.ErrValidation.WithMessage("balance บัญชีฝั่ง Seamless ไม่มียอดเงิน ห้ามส่ง", "balance is not allowed for Seamless accounts")
	}
	has := map[string]bool{}
	for _, c := range accountCurrencies {
		has[c] = true
	}
	for cur := range amounts {
		if !has[cur] {
			return apperr.ErrValidation.WithMessage("balance."+cur+" ไม่ใช่สกุลของบัญชีใหม่", "balance."+cur+" is not a currency of the new account")
		}
	}
	return nil
}

// TransferInitialBalance — โอนยอดเงินตั้งต้นจากผู้สร้างให้บัญชีใหม่ใน tx เดียวกับการสร้าง (MGMT-15A · กฎข้อ 10–12)
// Superadmin วงเงินไม่จำกัด: ไม่หักยอด · ledger เฉพาะฝั่งที่ได้รับ
func TransferInitialBalance(tx *gorm.DB, c Creator, owner models.BalanceOwnerType, ownerID uint, amounts map[string]int64,
	requestID string, actor agentAuthService.Actor, now time.Time) error {
	if len(amounts) == 0 {
		return nil
	}
	currencies := make([]string, 0, len(amounts))
	for cur := range amounts {
		currencies = append(currencies, cur)
	}
	sort.Strings(currencies)

	var ledger []models.BalanceLedger
	entry := func(ot models.BalanceOwnerType, id uint, cur string, amount, after int64, reason models.LedgerReason, refType string, refID uint) {
		ledger = append(ledger, models.BalanceLedger{OwnerType: ot, OwnerID: id, Currency: cur, Amount: amount, BalanceAfter: after,
			Reason: reason, RefType: &refType, RefID: &refID, RequestID: requestID,
			ActorType: actor.AccountType, ActorID: actor.AccountID(), CreatedAt: now})
	}

	inReason := models.LedgerInitialFromSuperadmin
	if c.UserType != agentManagementCore.UserTypeSuperadmin {
		inReason = models.LedgerInitialTransferIn
		locked, err := agentManagementPostgres.LockAgentBalancesRepository(tx, c.Agent.ID, currencies)
		if err != nil {
			return err
		}
		have := map[string]int64{}
		for _, b := range locked {
			have[b.Currency] = b.Amount
		}
		for _, cur := range currencies {
			after := have[cur] - amounts[cur]
			if after < 0 {
				return apperr.ErrInsufficientInitial
			}
			if err := agentManagementPostgres.UpdateAgentBalanceRepository(tx, c.Agent.ID, cur, after, now); err != nil {
				return err
			}
			entry(models.BalanceOwnerAgent, c.Agent.ID, cur, -amounts[cur], after, models.LedgerInitialTransferOut, string(owner), ownerID)
		}
	}

	switch owner {
	case models.BalanceOwnerAgent:
		rows := make([]models.AgentBalance, 0, len(currencies))
		for _, cur := range currencies {
			rows = append(rows, models.AgentBalance{AgentID: ownerID, Currency: cur, Amount: amounts[cur], UpdatedAt: now})
		}
		if err := agentManagementPostgres.CreateAgentBalancesRepository(tx, rows); err != nil {
			return err
		}
	case models.BalanceOwnerMember: // ยอดอยู่ที่ user_members.credit (Member มี 1 สกุล — MGMT-13) · แถว Member เพิ่งสร้างใน tx นี้
		for _, cur := range currencies {
			if err := memberManagementPostgres.UpdateUserMemberCreditRepository(tx, ownerID, utils.MinorToCredit(amounts[cur]), now); err != nil {
				return err
			}
		}
	}
	for _, cur := range currencies {
		entry(owner, ownerID, cur, amounts[cur], amounts[cur], inReason, string(models.BalanceOwnerAgent), c.Agent.ID)
	}
	return agentManagementPostgres.CreateBalanceLedgerRepository(tx, ledger)
}

// ChangeLog — แถว account_change_logs (MGMT-60) · value ห้ามมีรหัสผ่าน
func ChangeLog(actor agentAuthService.Actor, meta agentAuthService.RequestMeta, requestID, targetType string, targetID uint,
	targetUsername string, action models.AccountChangeAction, oldValue, newValue any, now time.Time) (models.AccountChangeLog, error) {
	l := models.AccountChangeLog{ActorType: actor.AccountType, ActorID: actor.AccountID(), ActorUsername: actor.Username,
		TargetType: targetType, TargetID: targetID, TargetUsername: targetUsername, Action: action, CreatedAt: now}
	for _, p := range []struct {
		v   any
		dst **string
	}{{oldValue, &l.OldValue}, {newValue, &l.NewValue}} {
		if p.v == nil {
			continue
		}
		b, err := json.Marshal(p.v)
		if err != nil {
			return l, err
		}
		s := string(b)
		*p.dst = &s
	}
	if meta.IP != "" {
		l.IP = &meta.IP
	}
	if requestID != "" {
		l.RequestID = &requestID
	}
	return l, nil
}

// LockAndCheckIdentity — กันสร้างพร้อมกัน แล้วเช็ค username / เบอร์ซ้ำ (MGMT-05, MGMT-08)
func LockAndCheckIdentity(tx *gorm.DB, username, phone string, member bool) error {
	if err := agentManagementPostgres.AdvisoryXactLockRepository(tx, "username:"+username); err != nil {
		return err
	}
	taken, err := agentManagementPostgres.UsernameExistsRepository(tx, username)
	if err != nil {
		return err
	}
	if taken {
		return apperr.ErrUsernameTaken
	}
	if phone == "" {
		return nil
	}
	if err := agentManagementPostgres.AdvisoryXactLockRepository(tx, "phone:"+phone); err != nil {
		return err
	}
	if member {
		taken, err = memberManagementPostgres.UserMemberPhoneExistsRepository(tx, phone)
	} else {
		taken, err = agentManagementPostgres.AgentPhoneExistsRepository(tx, phone)
	}
	if err != nil {
		return err
	}
	if taken {
		return apperr.ErrPhoneTaken
	}
	return nil
}

// FindReplay — request_id ที่ใช้แล้ว (MGMT-15A) · ของผู้เรียกคนเดิมและเส้นเดิม = คืนผลเดิม · อื่นๆ = 422
func FindReplay(tx *gorm.DB, requestID string, actor agentAuthService.Actor, targetType string) (uint, bool, error) {
	if err := agentManagementPostgres.AdvisoryXactLockRepository(tx, "create_request:"+requestID); err != nil {
		return 0, false, err
	}
	r, err := agentManagementPostgres.GetCreateRequestRepository(tx, requestID)
	if err != nil {
		if errors.Is(err, apperr.ErrNotFound) {
			return 0, false, nil
		}
		return 0, false, err
	}
	if r.CreatorType != actor.AccountType || r.CreatorID != actor.AccountID() || r.TargetType != targetType {
		return 0, false, apperr.ErrValidation.WithMessage("request_id นี้ถูกใช้แล้ว", "request_id has already been used")
	}
	return r.TargetID, true, nil
}

func OptionalString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// PTError — แปลงผลตรวจ Commission ของ Member เป็น error (MGMT-21) · ฝั่ง agent ใช้ ChildPTError / OwnPTError
func PTError(group string, v agentManagementCore.PTViolation) error {
	switch v {
	case agentManagementCore.PTOK:
		return nil
	case agentManagementCore.PTInvalidStep: // ใช้กับ Commission ของ Member (member_management)
		return apperr.ErrValidation.WithMessage("pt."+group+".commission_percent ตั้งได้ 0 – 1 ทีละ 0.1",
			"pt."+group+".commission_percent must be 0 – 1 in steps of 0.1")
	case agentManagementCore.PTExceedsReceived:
		return apperr.ErrPTExceedsReceived
	case agentManagementCore.PTSeamlessMasterLock:
		return apperr.ErrSeamlessMasterPTLocked
	case agentManagementCore.PTForceRemainExceeded:
		return apperr.ErrForceRemainExceeded
	default:
		return apperr.ErrCommissionExceeded.WithMessage("pt."+group+".commission_percent ตั้งได้ 0 – 1", "pt."+group+".commission_percent must be 0 – 1")
	}
}
