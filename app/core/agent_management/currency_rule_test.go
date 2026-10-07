package agentmanagement

import (
	"reflect"
	"testing"
)

func TestCurrencies(t *testing.T) {
	if len(Currencies) != 27 {
		t.Fatalf("len(Currencies) = %d, want 27", len(Currencies))
	}
	for i := 1; i < len(Currencies); i++ {
		if Currencies[i-1] >= Currencies[i] {
			t.Errorf("Currencies not sorted at %s", Currencies[i])
		}
	}
	if !IsSupportedCurrency("THB") || IsSupportedCurrency("thb") || IsSupportedCurrency("XXX") {
		t.Error("IsSupportedCurrency")
	}
}

func TestResolveCurrencies(t *testing.T) { // MGMT-10 – MGMT-14
	all := Currencies
	b2b := []string{"THB", "USD"}
	thb := []string{"THB"}
	tests := []struct {
		name            string
		creator, typ    UserType
		req, creatorCur []string
		want            []string
		v               CurrencyViolation
	}{
		{"Company Transfer ได้ครบ", UserTypeSuperadmin, UserTypeCompanyTransfer, nil, all, all, CurrencyOK},
		{"Company Transfer ส่งสกุลมา", UserTypeSuperadmin, UserTypeCompanyTransfer, thb, all, nil, CurrencyInvalidInput},
		{"Seamless Master ได้ครบ", UserTypeSuperadmin, UserTypeCompanySeamlessMaster, nil, all, all, CurrencyOK},
		{"1 to 1 เลือก 1", UserTypeSuperadmin, UserTypeCompanySeamless1to1, []string{"JPY"}, all, []string{"JPY"}, CurrencyOK},
		{"1 to 1 ไม่ส่ง", UserTypeSuperadmin, UserTypeCompanySeamless1to1, nil, all, nil, CurrencyInvalidInput},
		{"Share B2B ไม่ส่ง", UserTypeCompanyTransfer, UserTypeShareB2B, nil, all, nil, CurrencyInvalidInput},
		{"Share B2B หลายสกุล", UserTypeCompanyTransfer, UserTypeShareB2B, b2b, all, b2b, CurrencyOK},
		{"Share B2B ซ้ำ", UserTypeCompanyTransfer, UserTypeShareB2B, []string{"THB", "THB"}, all, nil, CurrencyInvalidInput},
		{"Share B2C 2 สกุล", UserTypeCompanyTransfer, UserTypeShareB2C, b2b, all, nil, CurrencyInvalidInput},
		{"Share Reseller 1 สกุล", UserTypeCompanySeamlessReseller, UserTypeShareReseller, thb, all, thb, CurrencyOK},
		{"สกุลไม่รู้จัก", UserTypeCompanyTransfer, UserTypeShareB2C, []string{"XXX"}, all, nil, CurrencyInvalidInput},
		{"Agent ใต้ B2B เลือก 1", UserTypeShareB2B, UserTypeAgent, []string{"USD"}, b2b, []string{"USD"}, CurrencyOK},
		{"Agent ใต้ B2B เลือก JPY", UserTypeShareB2B, UserTypeAgent, []string{"JPY"}, b2b, nil, CurrencyNotInCreators},
		{"Agent ใต้ B2B ไม่ส่ง", UserTypeShareB2B, UserTypeAgent, nil, b2b, nil, CurrencyInvalidInput},
		{"Agent ใต้ B2C ใช้ของผู้สร้าง", UserTypeShareB2C, UserTypeAgent, nil, thb, thb, CurrencyOK},
		{"Agent ใต้ Agent ส่งมา", UserTypeAgent, UserTypeAgent, thb, thb, nil, CurrencyInvalidInput},
		{"Member ใช้ของผู้สร้าง", UserTypeAgent, UserTypeMember, nil, thb, thb, CurrencyOK},
		{"Member ใต้ 1 to 1", UserTypeCompanySeamless1to1, UserTypeMember, nil, []string{"JPY"}, []string{"JPY"}, CurrencyOK},
	}
	for _, tt := range tests {
		got, v := ResolveCurrencies(tt.creator, tt.typ, tt.req, tt.creatorCur)
		if v != tt.v || !reflect.DeepEqual(got, tt.want) {
			t.Errorf("%s: ResolveCurrencies = %v, %v want %v, %v", tt.name, got, v, tt.want, tt.v)
		}
	}
}
