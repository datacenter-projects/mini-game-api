// Package membermanagement — กฎ business ของ Member ที่ไม่ใช้ร่วมกับฝั่ง agent (pure function)
// spec: docs/modules/member_management.md MGMT-63 – MGMT-65 · กฎที่ใช้ร่วมกับ agent อยู่ใน core/agent_management
package membermanagement

import (
	"regexp"
	"strings"
	"unicode/utf8"

	"app/app/models"
)

// thaiBanks — รายการธนาคาร (หัวข้อ 2.2 · lead P5 ธนาคารไทยเท่านั้น) · เพิ่มต้องแก้ spec ก่อน
var thaiBanks = map[string]bool{
	"BBL": true, "KBANK": true, "KTB": true, "SCB": true, "BAY": true, "TTB": true, "KKP": true, "CIMBT": true, "TISCO": true,
	"UOBT": true, "LHBANK": true, "ICBCT": true, "TCRB": true, "GSB": true, "BAAC": true, "GHB": true, "ISBT": true,
}

// IsThaiBankCode — รหัสอยู่ในรายการธนาคาร (ตัวใหญ่แล้ว)
func IsThaiBankCode(code string) bool { return thaiBanks[code] }

// ParsePaymentType — ตัดช่องว่าง แปลงตัวใหญ่ · ต้องเป็น CREDIT / CASH (MGMT-63)
func ParsePaymentType(s string) (models.PaymentType, bool) {
	t := models.PaymentType(strings.ToUpper(strings.TrimSpace(s)))
	return t, t == models.PaymentTypeCredit || t == models.PaymentTypeCash
}

// StoredPaymentType — ค่าที่เก็บจริง: ผู้สร้างฝั่ง Seamless = CASH เสมอ (MGMT-63 · lead P1)
func StoredPaymentType(t models.PaymentType, seamless bool) models.PaymentType {
	if seamless {
		return models.PaymentTypeCash
	}
	return t
}

// NameMaxLen — ชื่อเจ้าของบัญชีแต่ละ field ไม่เกินกี่ตัวอักษร (lead P7)
const NameMaxLen = 50

var (
	accountNumberRe = regexp.MustCompile(`^[0-9]{10,15}$`)
	// อักษรไทย (ก – ฺ · เ – ๎) ไม่รวมเลขไทย · ขึ้นต้นด้วยตัวอักษร · ตามด้วยตัวอักษร / ช่องว่าง / -
	thaiNameRe = regexp.MustCompile(`^[\x{0E01}-\x{0E3A}\x{0E40}-\x{0E4E}][\x{0E01}-\x{0E3A}\x{0E40}-\x{0E4E} \-]*$`)
	engNameRe  = regexp.MustCompile(`^[A-Za-z][A-Za-z \-]*$`)
)

// PaymentAccountViolation — ผลตรวจบัญชีธนาคาร (MGMT-64)
type PaymentAccountViolation int

const (
	PaymentAccountOK     PaymentAccountViolation = iota
	AccountNotPaired                             // ส่ง bank_code หรือ account_number ค่าเดียว
	BankCodeInvalid                              // ไม่อยู่ในรายการธนาคาร
	AccountNumberInvalid                         // ไม่ใช่ตัวเลขล้วน 10–15 หลัก
	NameWithoutAccount                           // ส่งชื่อโดยไม่มี bank_code + account_number
	NameInvalid                                  // ชื่อผิดรูปแบบ / ยาวเกิน
)

// PaymentAccountIssue — ผิดข้อไหนที่ field ไหน (ชื่อ field ใน JSON)
type PaymentAccountIssue struct {
	Violation PaymentAccountViolation
	Field     string
}

// NormalizePaymentAccount — ตัดช่องว่างหัวท้ายทุก field · bank_code ตัวใหญ่
func NormalizePaymentAccount(a models.MemberPaymentAccount) models.MemberPaymentAccount {
	for _, p := range []*string{&a.BankCode, &a.AccountNumber, &a.NameTH, &a.MidnameTH, &a.SurnameTH, &a.NameEN, &a.MidnameEN, &a.SurnameEN} {
		*p = strings.TrimSpace(*p)
	}
	a.BankCode = strings.ToUpper(a.BankCode)
	return a
}

// HasPaymentAccount — มีบัญชีไหม (ผ่าน CheckPaymentAccount แล้ว = มีครบคู่หรือว่างทั้งหมด)
func HasPaymentAccount(a models.MemberPaymentAccount) bool {
	return a.BankCode != "" || a.AccountNumber != ""
}

// CheckPaymentAccount — ค่าที่ normalize แล้ว · ว่างทั้งหมด = ไม่มีบัญชี (OK)
// ตรวจตามลำดับ field ใน body: คู่ → bank_code → account_number → ชื่อ (name_th → … → surname_en)
func CheckPaymentAccount(a models.MemberPaymentAccount) PaymentAccountIssue {
	names := []struct {
		field, value string
		thai         bool
	}{
		{"name_th", a.NameTH, true}, {"midname_th", a.MidnameTH, true}, {"surname_th", a.SurnameTH, true},
		{"name_en", a.NameEN, false}, {"midname_en", a.MidnameEN, false}, {"surname_en", a.SurnameEN, false},
	}
	if !HasPaymentAccount(a) {
		for _, n := range names {
			if n.value != "" {
				return PaymentAccountIssue{NameWithoutAccount, n.field}
			}
		}
		return PaymentAccountIssue{Violation: PaymentAccountOK}
	}
	if a.BankCode == "" || a.AccountNumber == "" {
		return PaymentAccountIssue{AccountNotPaired, "bank_code / account_number"}
	}
	if !IsThaiBankCode(a.BankCode) {
		return PaymentAccountIssue{BankCodeInvalid, "bank_code"}
	}
	if !accountNumberRe.MatchString(a.AccountNumber) {
		return PaymentAccountIssue{AccountNumberInvalid, "account_number"}
	}
	for _, n := range names {
		if n.value != "" && !isValidAccountName(n.value, n.thai) {
			return PaymentAccountIssue{NameInvalid, n.field}
		}
	}
	return PaymentAccountIssue{Violation: PaymentAccountOK}
}

func isValidAccountName(s string, thai bool) bool {
	if utf8.RuneCountInString(s) > NameMaxLen {
		return false
	}
	if thai {
		return thaiNameRe.MatchString(s)
	}
	return engNameRe.MatchString(s)
}

// MaskAccountNumber — เลขบัญชีใน log: 3 ตัวแรก + ••• + 4 ตัวท้าย (เช่น 787•••6654 · lead P8) · ว่าง = ""
func MaskAccountNumber(n string) string {
	if n == "" {
		return ""
	}
	if len(n) < 8 { // ผ่านการตรวจแล้วยาว 10–15 เสมอ · กันข้อมูลผิดรูปไม่ให้หลุดเต็ม
		return "•••"
	}
	return n[:3] + "•••" + n[len(n)-4:]
}
