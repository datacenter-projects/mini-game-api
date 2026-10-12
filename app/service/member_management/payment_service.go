package membermanagement

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	agentManagementCore "app/app/core/agent_management"
	memberManagementCore "app/app/core/member_management"
	memberManagementDto "app/app/internals/backoffice/dto/member_management"
	"app/app/models"
	agentManagementPostgres "app/app/repository/postgres/agent_management"
	memberManagementPostgres "app/app/repository/postgres/member_management"
	agentAuthService "app/app/service/agent_auth"
	agentManagementService "app/app/service/agent_management"
	"app/pkg/apperr"

	"gorm.io/gorm"
)

// วิธีจ่ายเงิน · บัญชีธนาคาร · ลิมิตชนะต่อวัน — spec: docs/modules/member_management.md MGMT-63 – MGMT-65 (lead review P1–P15)

// paymentLogValue — ค่าใน log CREATE / UPDATE_PAYMENT · เลขบัญชีปิดบางส่วน (lead P8)
type paymentLogValue struct {
	PaymentType    models.PaymentType                 `json:"payment_type"`
	PaymentAccount memberManagementDto.PaymentAccount `json:"payment_account"`
}

// limitLogValue — ค่าใน log UPDATE_LIMIT (MGMT-65)
type limitLogValue struct {
	WinLimitPerDay float64 `json:"win_limit_per_day"`
}

// maskedPaymentAccount — บัญชีสำหรับ log: เลขบัญชีปิดบางส่วน · ไม่มีบัญชี = ทุก field ""
func maskedPaymentAccount(a models.MemberPaymentAccount) memberManagementDto.PaymentAccount {
	v := memberManagementDto.PaymentAccountView(a)
	v.AccountNumber = memberManagementCore.MaskAccountNumber(a.AccountNumber)
	return v
}

// encodePaymentAccount — ค่าที่เก็บใน user_members.payment_account · ไม่มีบัญชี = nil (NULL)
func encodePaymentAccount(a models.MemberPaymentAccount) (*string, error) {
	if !memberManagementCore.HasPaymentAccount(a) {
		return nil, nil
	}
	b, err := json.Marshal(a)
	if err != nil {
		return nil, err
	}
	s := string(b)
	return &s, nil
}

// decodePaymentAccount — อ่านค่าที่เก็บ · NULL = ไม่มีบัญชี
func decodePaymentAccount(s *string) (models.MemberPaymentAccount, error) {
	var a models.MemberPaymentAccount
	if s == nil {
		return a, nil
	}
	err := json.Unmarshal([]byte(*s), &a)
	return a, err
}

// lockMemberPaymentAccountFree — lock บัญชี (ธนาคาร + เลข) ภายใต้ผู้สร้าง แล้วเช็คว่า Member อื่นของผู้สร้างคนเดียวกันใช้แล้วไหม (MGMT-64 · lead P4)
// ไม่มีบัญชี = ไม่เช็ค · คนละผู้สร้างซ้ำได้ · selfID 0 = Member ใหม่
func lockMemberPaymentAccountFree(tx *gorm.DB, agentID uint, a models.MemberPaymentAccount, selfID uint) error {
	if !memberManagementCore.HasPaymentAccount(a) {
		return nil
	}
	key := fmt.Sprintf("member_bank:%d:%s-%s", agentID, a.BankCode, a.AccountNumber)
	if err := agentManagementPostgres.AdvisoryXactLockRepository(tx, key); err != nil {
		return err
	}
	taken, err := memberManagementPostgres.UserMemberPaymentAccountTakenByOtherRepository(tx, agentID, a.BankCode, a.AccountNumber, selfID)
	if err != nil {
		return err
	}
	if taken {
		return apperr.ErrPaymentAccountTaken
	}
	return nil
}

// memberCreatorSeamless — ผู้สร้าง Member อยู่สายฝั่ง Seamless ไหม (Company หัวสาย) — ใช้ตั้ง payment_type = CASH (MGMT-63)
func memberCreatorSeamless(tx *gorm.DB, agentID uint) (bool, error) {
	agentType, err := agentManagementPostgres.GetChainCompanyTypeRepository(tx, agentID)
	if err != nil || agentType == nil {
		return false, err
	}
	return agentManagementCore.IsSeamless(agentManagementCore.UserTypeOf(models.AgentRoleCompany, agentType)), nil
}

// paymentState — วิธีจ่ายเงิน + บัญชี ที่ใช้เทียบค่าเดิม / ใหม่ (เลขบัญชีเต็ม)
type paymentState struct {
	Type    models.PaymentType
	Account models.MemberPaymentAccount
}

// applyMemberPayment — section payment (MGMT-32M · MGMT-63 / 64 · lead P13) · field ที่ไม่ส่งคงค่าเดิม · ไม่เปลี่ยน = ไม่เขียน log
// สิทธิ์ payment edit เช็คก่อนเปิด tx (UpdateMemberDetailService)
func applyMemberPayment(ctx context.Context, tx *gorm.DB, actor agentAuthService.Actor, meta agentAuthService.RequestMeta,
	m models.UserMember, in memberManagementDto.MemberPaymentSection, now time.Time) error {
	oldAccount, err := decodePaymentAccount(m.PaymentAccount)
	if err != nil {
		return err
	}
	old := paymentState{Type: m.PaymentType, Account: oldAccount}
	next := old
	if in.Type != "" {
		seamless, err := memberCreatorSeamless(tx, m.AgentID)
		if err != nil {
			return err
		}
		next.Type = memberManagementCore.StoredPaymentType(in.Type, seamless)
	}
	if in.Account != nil {
		next.Account = *in.Account
	}
	if next == old {
		return nil
	}
	if next.Account.BankCode != old.Account.BankCode || next.Account.AccountNumber != old.Account.AccountNumber {
		if err := lockMemberPaymentAccountFree(tx, m.AgentID, next.Account, m.ID); err != nil {
			return err
		}
	}
	stored, err := encodePaymentAccount(next.Account)
	if err != nil {
		return err
	}
	err = memberManagementPostgres.UpdateUserMemberPaymentRepository(tx, m.ID, next.Type, stored, now)
	if errors.Is(err, apperr.ErrConflict) { // unique เดียวของคอลัมน์ที่แก้คือบัญชี (หลุด advisory lock — เช่น แถวที่ยังไม่ commit ตอนเช็ค)
		return apperr.ErrPaymentAccountTaken
	}
	if err != nil {
		return err
	}
	return agentManagementService.WriteLog(ctx, tx, actor, meta, agentManagementService.TargetMember, m.ID, m.Username, models.ChangeUpdatePayment,
		paymentLogValue{old.Type, maskedPaymentAccount(old.Account)}, paymentLogValue{next.Type, maskedPaymentAccount(next.Account)}, now)
}

// applyMemberLimit — section limit (MGMT-65 · lead P13) · ค่าเดิม = ไม่เขียน log
func applyMemberLimit(ctx context.Context, tx *gorm.DB, actor agentAuthService.Actor, meta agentAuthService.RequestMeta,
	m models.UserMember, in memberManagementDto.MemberLimitSection, now time.Time) error {
	if *in.WinLimitPerDay == m.WinLimitPerDay { // ทั้งคู่ปัด 4 ตำแหน่งแล้ว (กฎข้อ 9)
		return nil
	}
	if err := memberManagementPostgres.UpdateUserMemberWinLimitRepository(tx, m.ID, *in.WinLimitPerDay, now); err != nil {
		return err
	}
	return agentManagementService.WriteLog(ctx, tx, actor, meta, agentManagementService.TargetMember, m.ID, m.Username, models.ChangeUpdateLimit,
		limitLogValue{m.WinLimitPerDay}, limitLogValue{*in.WinLimitPerDay}, now)
}
