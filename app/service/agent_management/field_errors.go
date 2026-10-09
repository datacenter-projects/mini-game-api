package agentmanagement

import (
	"strings"

	agentManagementCore "app/app/core/agent_management"
	agentManagementPostgres "app/app/repository/postgres/agent_management"
	"app/pkg/apperr"
	"app/pkg/utils"

	"gorm.io/gorm"
)

// error ที่บอกชื่อ field และค่าที่ตั้งได้ (แก้ 2026-10-09) — code เดิม เปลี่ยนแค่ msg

func pct(v float64) string { return utils.FormatNum(v) }

func joinTypes(ts []agentManagementCore.UserType) string {
	s := make([]string, len(ts))
	for i, t := range ts {
		s[i] = string(t)
	}
	return strings.Join(s, ", ")
}

// CannotCreateTypeError — 402301 บอกค่าที่ส่งมา · เหตุผลที่ส่งไม่ได้ · รายการ user_type ที่ผู้สร้างส่งได้ (MGMT-02)
func CannotCreateTypeError(creator agentManagementCore.UserType, sent string) error {
	whyTH, whyEN := cannotCreateReason(creator, agentManagementCore.UserType(sent))
	c := string(creator)
	allowedTH, allowedEN := c+" สร้างบัญชีฝั่ง agent ไม่ได้", c+" cannot create agent-side accounts"
	if allowed := agentManagementCore.CreatableTypes(creator); len(allowed) > 0 {
		allowedTH, allowedEN = c+" สร้างได้เฉพาะ "+joinTypes(allowed), c+" can only create "+joinTypes(allowed)
	}
	return apperr.ErrCannotCreateType.WithMessage(
		"user_type: ส่ง "+sent+" ไม่ได้ · "+whyTH+" · "+allowedTH,
		"user_type: "+sent+" is not allowed · "+whyEN+" · "+allowedEN)
}

func cannotCreateReason(creator, sent agentManagementCore.UserType) (string, string) {
	s := string(sent)
	if creators := agentManagementCore.CreatorsOf(sent); len(creators) > 0 {
		return s + " สร้างได้โดย " + joinTypes(creators) + " เท่านั้น แต่คุณเป็น " + string(creator),
			s + " can only be created by " + joinTypes(creators) + ", you are " + string(creator)
	}
	switch sent {
	case agentManagementCore.UserTypeMember:
		return "สร้าง MEMBER ใช้เส้น /api/v1/bo/pr/manage/members/create", "create MEMBER via /api/v1/bo/pr/manage/members/create"
	case agentManagementCore.UserTypeShareReseller, agentManagementCore.UserTypeShareMaster:
		return "ให้ส่ง SHARE_B2C ระบบตั้งเป็น " + s + " เองตามประเภท Company ที่สร้าง",
			"send SHARE_B2C, the system sets " + s + " from the creating Company type"
	case agentManagementCore.UserTypeSuperadmin, agentManagementCore.UserTypeAdmin:
		return s + " สร้างผ่าน API ไม่ได้", s + " cannot be created via the API"
	}
	return "ไม่มีประเภท " + s, s + " is not a user type"
}

// CurrencyError — 422 / 402310 พร้อมกฎ currencies ของประเภทบัญชีใหม่ (MGMT-10 – MGMT-14)
func CurrencyError(creator, newType agentManagementCore.UserType, cv agentManagementCore.CurrencyViolation, creatorCurrencies []string) error {
	if cv == agentManagementCore.CurrencyNotInCreators {
		return apperr.ErrCurrencyNotInCreator.WithMessage(
			"currencies: เลือกได้เฉพาะสกุลของคุณ ("+strings.Join(creatorCurrencies, ", ")+")",
			"currencies: choose only from your currencies ("+strings.Join(creatorCurrencies, ", ")+")")
	}
	t := string(newType)
	switch agentManagementCore.CurrencyRequirementOf(creator, newType) {
	case agentManagementCore.CurrencyAllNoSend:
		return apperr.ErrValidation.WithMessage("currencies: "+t+" ได้ครบทุกสกุล ห้ามส่ง currencies",
			"currencies: "+t+" gets every currency, do not send currencies")
	case agentManagementCore.CurrencyFromCreatorNoSend:
		return apperr.ErrValidation.WithMessage("currencies: "+t+" ใช้สกุลของผู้สร้าง ห้ามส่ง currencies",
			"currencies: "+t+" uses the creator's currency, do not send currencies")
	case agentManagementCore.CurrencyPickOne:
		return apperr.ErrValidation.WithMessage("currencies: "+t+" ต้องเลือก 1 สกุล (ไม่ซ้ำ)",
			"currencies: "+t+" needs exactly 1 currency")
	}
	return apperr.ErrValidation.WithMessage("currencies: "+t+" ต้องเลือกอย่างน้อย 1 สกุล (ไม่ซ้ำ)",
		"currencies: "+t+" needs at least 1 currency (no duplicates)")
}

// ChildPTError — ผลตรวจค่าที่ผู้สร้างตั้งให้ลูก → error ที่บอก field และค่าที่ตั้งได้ (MGMT-18, MGMT-19, MGMT-25)
func ChildPTError(group string, is agentManagementCore.PTIssue) error {
	f := "pt." + group + "." + is.Field
	switch is.Violation {
	case agentManagementCore.PTOK:
		return nil
	case agentManagementCore.PTInvalidStep:
		if is.Field == "commission_percent" {
			return apperr.ErrValidation.WithMessage(f+" ตั้งได้ 0 – 1 ทีละ 0.1", f+" must be 0 – 1 in steps of 0.1")
		}
		return apperr.ErrValidation.WithMessage(f+" ตั้งได้ 0 – 100 ทีละ 0.5", f+" must be 0 – 100 in steps of 0.5")
	case agentManagementCore.PTExceedsReceived:
		return apperr.ErrPTExceedsReceived.WithMessage(f+" ตั้งได้ไม่เกิน "+pct(is.Limit)+" (ค่าที่คุณได้รับ)",
			f+" must not exceed "+pct(is.Limit)+" (what you received)")
	case agentManagementCore.PTSeamlessMasterLock:
		if is.Field == "pt_from_parent" {
			return apperr.ErrSeamlessMasterPTLocked.WithMessage(f+" ของ Company Seamless Master ต้องเท่ากับ "+pct(is.Limit)+" (ค่าที่ได้รับทั้งหมด)",
				f+" must equal "+pct(is.Limit)+" for Company Seamless Master")
		}
		return apperr.ErrSeamlessMasterPTLocked.WithMessage(f+" ของ Company Seamless Master ต้องเป็น 0", f+" must be 0 for Company Seamless Master")
	case agentManagementCore.PTForceRemainExceeded:
		return apperr.ErrForceRemainExceeded.WithMessage(f+" ตั้งได้ไม่เกิน "+pct(is.Limit)+" (ค่าที่ให้ลูก)",
			f+" must not exceed "+pct(is.Limit)+" (pt_from_parent)")
	default:
		return apperr.ErrCommissionExceeded.WithMessage(f+" ตั้งได้ 0 – 1", f+" must be 0 – 1")
	}
}

// MemberPTError — ค่าที่ผู้สร้างถือสู้กับ Member (member_management MGMT-21) · ไม่เกินค่าที่ผู้สร้างได้รับ ทีละ 0.5
func MemberPTError(group string, v agentManagementCore.PTViolation, received float64) error {
	f := "pt." + group + ".pt"
	switch v {
	case agentManagementCore.PTOK:
		return nil
	case agentManagementCore.PTInvalidStep:
		return apperr.ErrValidation.WithMessage(f+" ตั้งได้ 0 – "+pct(received)+" ทีละ 0.5", f+" must be 0 – "+pct(received)+" in steps of 0.5")
	default:
		return apperr.ErrPTExceedsReceived.WithMessage(f+" ตั้งได้ไม่เกิน "+pct(received)+" (ค่าที่คุณได้รับ)",
			f+" must not exceed "+pct(received)+" (what you received)")
	}
}

// CheckCreatorBalance — ยอดของผู้สร้างพอสำหรับยอดเงินตั้งต้นไหม (MGMT-15A) · เช็คก่อนค่าหุ้นส่วนตามลำดับ field
// lock แถวยอดของผู้สร้างใน tx เดียวกัน · TransferInitialBalance เช็คซ้ำตอนโอนจริง · Superadmin ไม่จำกัด
func CheckCreatorBalance(tx *gorm.DB, c Creator, amounts map[string]float64) error {
	if len(amounts) == 0 || c.UserType == agentManagementCore.UserTypeSuperadmin {
		return nil
	}
	currencies := SortedGroups(amounts) // เรียงชื่อ — lock ตามลำดับเดียวกันทุกครั้ง
	locked, err := agentManagementPostgres.LockAgentBalancesRepository(tx, c.Agent.ID, currencies)
	if err != nil {
		return err
	}
	have := map[string]float64{}
	for _, b := range locked {
		have[b.Currency] = b.Amount
	}
	for _, cur := range currencies {
		if have[cur] < amounts[cur] {
			b := utils.FormatNum(have[cur])
			return apperr.ErrInsufficientInitial.WithMessage("balance."+cur+" ยอดของคุณไม่พอ (มี "+b+")",
				"balance."+cur+" exceeds your balance ("+b+")")
		}
	}
	return nil
}
