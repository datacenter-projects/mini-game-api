package membermanagement

import (
	"strings"
	"testing"

	"app/app/models"
)

func TestParsePaymentType(t *testing.T) { // MGMT-63
	tests := []struct {
		in   string
		want models.PaymentType
		ok   bool
	}{
		{"CREDIT", models.PaymentTypeCredit, true},
		{" cash ", models.PaymentTypeCash, true},
		{"credit", models.PaymentTypeCredit, true},
		{"", "", false},
		{"DEBIT", "DEBIT", false},
	}
	for _, tt := range tests {
		if got, ok := ParsePaymentType(tt.in); got != tt.want || ok != tt.ok {
			t.Errorf("ParsePaymentType(%q) = %q, %v · want %q, %v", tt.in, got, ok, tt.want, tt.ok)
		}
	}
}

func TestStoredPaymentType(t *testing.T) { // MGMT-63 · lead P1
	tests := []struct {
		in       models.PaymentType
		seamless bool
		want     models.PaymentType
	}{
		{models.PaymentTypeCredit, false, models.PaymentTypeCredit},
		{models.PaymentTypeCash, false, models.PaymentTypeCash},
		{models.PaymentTypeCredit, true, models.PaymentTypeCash},
		{models.PaymentTypeCash, true, models.PaymentTypeCash},
	}
	for _, tt := range tests {
		if got := StoredPaymentType(tt.in, tt.seamless); got != tt.want {
			t.Errorf("StoredPaymentType(%q, %v) = %q, want %q", tt.in, tt.seamless, got, tt.want)
		}
	}
}

func TestCheckPaymentAccount(t *testing.T) { // MGMT-64 · lead P3 – P7
	acc := func(edit func(a *models.MemberPaymentAccount)) models.MemberPaymentAccount {
		a := models.MemberPaymentAccount{BankCode: "KBANK", AccountNumber: "7876766654"}
		if edit != nil {
			edit(&a)
		}
		return NormalizePaymentAccount(a)
	}
	tests := []struct {
		name string
		a    models.MemberPaymentAccount
		want PaymentAccountIssue
	}{
		{"ไม่มีบัญชี", models.MemberPaymentAccount{}, PaymentAccountIssue{Violation: PaymentAccountOK}},
		{"ช่องว่างล้วน = ไม่มีบัญชี", acc(func(a *models.MemberPaymentAccount) { a.BankCode, a.AccountNumber = " ", " " }), PaymentAccountIssue{Violation: PaymentAccountOK}},
		{"บัญชีครบ", acc(nil), PaymentAccountIssue{Violation: PaymentAccountOK}},
		{"bank ตัวเล็ก + ช่องว่าง", acc(func(a *models.MemberPaymentAccount) { a.BankCode = " scb " }), PaymentAccountIssue{Violation: PaymentAccountOK}},
		{"เลข 15 หลัก", acc(func(a *models.MemberPaymentAccount) { a.AccountNumber = "123456789012345" }), PaymentAccountIssue{Violation: PaymentAccountOK}},
		{"ชื่อครบ 6 field", acc(func(a *models.MemberPaymentAccount) {
			a.NameTH, a.MidnameTH, a.SurnameTH = "ใจดี", "ณ", "มีสุข-ยิ่ง"
			a.NameEN, a.MidnameEN, a.SurnameEN = "Jai Dee", "Na", "Mee-Suk"
		}), PaymentAccountIssue{Violation: PaymentAccountOK}},
		{"ชื่อไทยขึ้นต้นสระหน้า", acc(func(a *models.MemberPaymentAccount) { a.NameTH = "เอกชัย" }), PaymentAccountIssue{Violation: PaymentAccountOK}},
		{"ชื่อ 50 ตัวพอดี", acc(func(a *models.MemberPaymentAccount) { a.NameEN = strings.Repeat("a", 50) }), PaymentAccountIssue{Violation: PaymentAccountOK}},
		{"ส่งแค่ bank", acc(func(a *models.MemberPaymentAccount) { a.AccountNumber = "" }), PaymentAccountIssue{AccountNotPaired, "bank_code / account_number"}},
		{"ส่งแค่เลข", acc(func(a *models.MemberPaymentAccount) { a.BankCode = "" }), PaymentAccountIssue{AccountNotPaired, "bank_code / account_number"}},
		{"ธนาคารไม่อยู่ในรายการ", acc(func(a *models.MemberPaymentAccount) { a.BankCode = "XYZ" }), PaymentAccountIssue{BankCodeInvalid, "bank_code"}},
		{"ธนาคารต่างประเทศ", acc(func(a *models.MemberPaymentAccount) { a.BankCode = "HSBC" }), PaymentAccountIssue{BankCodeInvalid, "bank_code"}},
		{"เลข 9 หลัก", acc(func(a *models.MemberPaymentAccount) { a.AccountNumber = "123456789" }), PaymentAccountIssue{AccountNumberInvalid, "account_number"}},
		{"เลข 16 หลัก", acc(func(a *models.MemberPaymentAccount) { a.AccountNumber = "1234567890123456" }), PaymentAccountIssue{AccountNumberInvalid, "account_number"}},
		{"เลขมีขีด", acc(func(a *models.MemberPaymentAccount) { a.AccountNumber = "787-6-76665-4" }), PaymentAccountIssue{AccountNumberInvalid, "account_number"}},
		{"bank ผิดก่อนเลข", acc(func(a *models.MemberPaymentAccount) { a.BankCode, a.AccountNumber = "XYZ", "12" }), PaymentAccountIssue{BankCodeInvalid, "bank_code"}},
		{"ชื่อโดยไม่มีบัญชี", NormalizePaymentAccount(models.MemberPaymentAccount{SurnameEN: "Meesuk"}), PaymentAccountIssue{NameWithoutAccount, "surname_en"}},
		{"name_th เป็นอังกฤษ", acc(func(a *models.MemberPaymentAccount) { a.NameTH = "Jaidee" }), PaymentAccountIssue{NameInvalid, "name_th"}},
		{"name_th มีเลขไทย", acc(func(a *models.MemberPaymentAccount) { a.NameTH = "ใจดี๑" }), PaymentAccountIssue{NameInvalid, "name_th"}},
		{"name_en เป็นไทย", acc(func(a *models.MemberPaymentAccount) { a.NameEN = "ใจดี" }), PaymentAccountIssue{NameInvalid, "name_en"}},
		{"name_en มีตัวเลข", acc(func(a *models.MemberPaymentAccount) { a.MidnameEN = "Jai2" }), PaymentAccountIssue{NameInvalid, "midname_en"}},
		{"ขึ้นต้นด้วย -", acc(func(a *models.MemberPaymentAccount) { a.SurnameEN = "-Meesuk" }), PaymentAccountIssue{NameInvalid, "surname_en"}},
		{"ชื่อ 51 ตัว", acc(func(a *models.MemberPaymentAccount) { a.SurnameTH = strings.Repeat("ก", 51) }), PaymentAccountIssue{NameInvalid, "surname_th"}},
		{"ชื่อไทยผิดก่อนอังกฤษ", acc(func(a *models.MemberPaymentAccount) { a.NameTH, a.NameEN = "A", "ก" }), PaymentAccountIssue{NameInvalid, "name_th"}},
	}
	for _, tt := range tests {
		if got := CheckPaymentAccount(tt.a); got != tt.want {
			t.Errorf("%s: got %+v, want %+v", tt.name, got, tt.want)
		}
	}
}

func TestNormalizePaymentAccount(t *testing.T) { // MGMT-64
	got := NormalizePaymentAccount(models.MemberPaymentAccount{BankCode: " kbank ", AccountNumber: " 7876766654 ", NameTH: " ใจดี ", SurnameEN: " Meesuk "})
	want := models.MemberPaymentAccount{BankCode: "KBANK", AccountNumber: "7876766654", NameTH: "ใจดี", SurnameEN: "Meesuk"}
	if got != want {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestMaskAccountNumber(t *testing.T) { // MGMT-64 · lead P8
	tests := []struct{ in, want string }{
		{"7876766654", "787•••6654"},
		{"123456789012345", "123•••2345"},
		{"", ""},
		{"1234567", "•••"},
	}
	for _, tt := range tests {
		if got := MaskAccountNumber(tt.in); got != tt.want {
			t.Errorf("MaskAccountNumber(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}
