package agentmanagement

// สกุลเงินของบัญชีใหม่ (MGMT-10 – MGMT-14)

// Currencies — 27 สกุลที่ระบบรองรับ เรียง A→Z (ตรงกับ account ACC-19)
var Currencies = []string{
	"ARS", "AUD", "BDT", "BOB", "BRL", "CLP", "CNY", "EUR", "GBP",
	"HKD", "IDR", "INR", "JPY", "KHR", "KRW", "LAK", "MMK", "MXN", "MYR", "NGN", "PHP",
	"PKR", "THB", "TWD", "USD", "USDT", "VND",
}

// IsSupportedCurrency — อยู่ใน 27 สกุลไหม (ตัวพิมพ์ใหญ่เท่านั้น)
func IsSupportedCurrency(c string) bool {
	for _, x := range Currencies {
		if x == c {
			return true
		}
	}
	return false
}

// CurrencyViolation — ผลการตรวจสกุล · service แปลงเป็น apperr
type CurrencyViolation int

const (
	CurrencyOK            CurrencyViolation = iota
	CurrencyInvalidInput                    // 422 — จำนวนไม่ตรงกฎ / สกุลไม่รู้จัก / ซ้ำ / ส่งมาทั้งที่ไม่ต้องส่ง
	CurrencyNotInCreators                   // 402310 (MGMT-14)
)

type currencyMode int

const (
	currencyAll         currencyMode = iota // ครบทุกสกุล ไม่ต้องส่ง
	currencyPickOne                         // เลือก 1 สกุล
	currencyPickMany                        // เลือกอย่างน้อย 1
	currencyFromCreator                     // ใช้สกุลของผู้สร้าง ไม่ต้องส่ง
)

func currencyModeOf(creator, newType UserType) currencyMode {
	switch newType {
	case UserTypeCompanyTransfer, UserTypeCompanySeamlessReseller, UserTypeCompanySeamlessMaster:
		return currencyAll // MGMT-10
	case UserTypeCompanySeamless1to1, UserTypeShareB2C, UserTypeShareReseller, UserTypeShareMaster:
		return currencyPickOne // MGMT-11, MGMT-12
	case UserTypeShareB2B:
		return currencyPickMany // MGMT-12
	case UserTypeAgent:
		if creator == UserTypeShareB2B {
			return currencyPickOne // MGMT-13
		}
	}
	return currencyFromCreator // MGMT-13 · Member
}

// ResolveCurrencies — สกุลของบัญชีใหม่ (MGMT-10 – MGMT-14)
// newType = ประเภทที่เก็บจริง (UserTypeMember สำหรับ Member) · requested = ค่าที่ส่งมา (ไม่ส่ง = ว่าง)
func ResolveCurrencies(creator, newType UserType, requested, creatorCurrencies []string) ([]string, CurrencyViolation) {
	switch currencyModeOf(creator, newType) {
	case currencyAll:
		if len(requested) > 0 {
			return nil, CurrencyInvalidInput
		}
		return append([]string(nil), Currencies...), CurrencyOK
	case currencyFromCreator:
		if len(requested) > 0 {
			return nil, CurrencyInvalidInput
		}
		return append([]string(nil), creatorCurrencies...), CurrencyOK
	case currencyPickOne:
		if len(requested) != 1 {
			return nil, CurrencyInvalidInput
		}
	case currencyPickMany:
		if len(requested) == 0 {
			return nil, CurrencyInvalidInput
		}
	}
	has := make(map[string]bool, len(creatorCurrencies))
	for _, c := range creatorCurrencies {
		has[c] = true
	}
	seen := map[string]bool{}
	for _, c := range requested {
		if !IsSupportedCurrency(c) || seen[c] {
			return nil, CurrencyInvalidInput
		}
		seen[c] = true
	}
	for _, c := range requested {
		if !has[c] {
			return nil, CurrencyNotInCreators
		}
	}
	return append([]string(nil), requested...), CurrencyOK
}
