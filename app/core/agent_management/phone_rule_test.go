package agentmanagement

import "testing"

func TestCheckPhone(t *testing.T) { // MGMT-08 (lead E1-5)
	tests := []struct {
		name, code, phone string
		want              PhoneViolation
	}{
		{"ไม่กรอก", "", "", PhoneOK},
		{"ไทยมือถือ", "66", "812345678", PhoneOK},
		{"ไทยบ้าน", "66", "21234567", PhoneOK},
		{"US", "1", "2025550123", PhoneOK},
		{"ลาว 3 หลัก", "856", "2055123456", PhoneOK},
		{"รวม 15 หลักพอดี", "66", "1234567890123", PhoneOK},
		{"ส่งแค่รหัส", "66", "", PhoneNotPaired},
		{"ส่งแค่เบอร์", "", "812345678", PhoneNotPaired},
		{"รหัสไม่มีจริง", "999", "812345678", PhoneCodeInvalid},
		{"รหัสมี +", "+66", "812345678", PhoneCodeInvalid},
		{"รหัสไม่ใช่ประเทศ 800", "800", "12345678", PhoneCodeInvalid},
		{"0 นำหน้า", "66", "0812345678", PhoneNumberInvalid},
		{"มีขีด", "66", "81-234-5678", PhoneNumberInvalid},
		{"มีช่องว่าง", "66", "81 2345678", PhoneNumberInvalid},
		{"รวม 16 หลัก", "66", "12345678901234", PhoneTooLong},
	}
	for _, tt := range tests {
		if got := CheckPhone(tt.code, tt.phone); got != tt.want {
			t.Errorf("%s: CheckPhone(%q, %q) = %v, want %v", tt.name, tt.code, tt.phone, got, tt.want)
		}
	}
}
