package utils

import (
	"strings"
	"testing"
)

func TestFieldLabel(t *testing.T) {
	tests := []struct {
		path, th, en string
	}{
		{"pt.minigame", "PT (Minigame)", "PT (Minigame)"},
		{"pt.minigame.pt_from_parent", "PT ที่ให้ (Minigame)", "PT given (Minigame)"},
		{"pt.minigame.force", "Force (Minigame)", "Force (Minigame)"},
		{"pt.minigame.remain_quota", "Remain (Minigame)", "Remain (Minigame)"},
		{"pt.minigame.commission_percent", "Commission (Minigame)", "Commission (Minigame)"},
		{"pt.minigame.status", "รับ PT (Minigame)", "Receives PT (Minigame)"},
		{"pt.minigame.pt", "PT ที่ถือสู้ (Minigame)", "PT held (Minigame)"},
		{"info.name", "ชื่อ", "Name"},
		{"info.phone_country_code / info.phone", "รหัสประเทศ / เบอร์โทร", "Country code / Phone"},
		{"status_game.rock_paper_scissors", "สถานะเกม (Rock Paper Scissors)", "Game status (Rock Paper Scissors)"},
		{"balance.THB", "ยอดเงินตั้งต้น (THB)", "Initial balance (THB)"},
		{"permissions.member", "สิทธิ์เมนู member", "Permission: member"},
		{"payment.payment_type", "ประเภทการชำระ", "Payment type"},
		{"payment.payment_account", "บัญชีรับเงิน", "Payment account"},
		{"payment_account.bank_code / payment_account.account_number", "ธนาคาร / เลขบัญชี", "Bank / Account number"},
		{"payment.payment_account.surname_en", "นามสกุล (อังกฤษ)", "Last name (English)"},
		{"limit.win_limit_per_day", "ยอดชนะสูงสุดต่อวัน", "Daily win limit"},
		// ระดับบนสุด · มี index
		{"win_limit_per_day", "ยอดชนะสูงสุดต่อวัน", "Daily win limit"},
		{"phone_country_code / phone", "รหัสประเทศ / เบอร์โทร", "Country code / Phone"},
		{"currencies[1]", "สกุลเงิน ลำดับที่ 2", "Currencies #2"},
		// ไม่อยู่ในตาราง (คำบนหน้าจอ / field ที่ผู้ใช้ไม่ได้กรอก / module อื่น) = คืน path เดิม
		{"username", "username", "username"},
		{"password", "password", "password"},
		{"allowed_ips[0]", "allowed_ips[0]", "allowed_ips[0]"},
		{"request_id", "request_id", "request_id"},
		{"pt", "pt", "pt"},
		{"unknown[1]", "unknown[1]", "unknown[1]"},
		{"currencies[x]", "currencies[x]", "currencies[x]"},
		{"pt.minigame.unknown", "pt.minigame.unknown", "pt.minigame.unknown"},
		{"info.unknown", "info.unknown", "info.unknown"},
		{"payment_account.unknown", "payment_account.unknown", "payment_account.unknown"},
		{"", "", ""},
	}
	for _, tt := range tests {
		th, en := FieldLabel(tt.path)
		if th != tt.th || en != tt.en {
			t.Errorf("FieldLabel(%q) = %q, %q want %q, %q", tt.path, th, en, tt.th, tt.en)
		}
	}
}

func TestSubFieldLabels(t *testing.T) {
	tests := []struct {
		section string
		fields  []string
		th, en  string
	}{
		{"pt", []string{"pt_from_parent", "commission_percent"}, "PT ที่ให้|Commission", "PT given|Commission"},
		{"payment_account", []string{"bank_code", "account_number"}, "ธนาคาร|เลขบัญชี", "Bank|Account number"},
		{"info", []string{"name", "unknown"}, "ชื่อ|unknown", "Name|unknown"},
		{"nope", []string{"name"}, "name", "name"},
	}
	for _, tt := range tests {
		th, en := SubFieldLabels(tt.section, tt.fields...)
		if strings.Join(th, "|") != tt.th || strings.Join(en, "|") != tt.en {
			t.Errorf("SubFieldLabels(%q, %v) = %v, %v want %q, %q", tt.section, tt.fields, th, en, tt.th, tt.en)
		}
	}
}
