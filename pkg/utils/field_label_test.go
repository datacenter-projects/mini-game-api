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
		{"pt.minigame.pt_from_parent", "PT ที่ปล่อยให้ชั้นล่าง", "PT given to the downline"},
		{"pt.minigame.force", "บังคับถือสู้", "Forced hold"},
		{"pt.minigame.remain_quota", "เอาส่วนที่เหลือ", "Take the remainder"},
		{"pt.minigame.commission_percent", "คอมมิชชั่น", "Commission"},
		{"pt.minigame.status", "รับ PT", "Receives PT"},
		{"pt.minigame.pt", "PT ที่ถือสู้กับสมาชิก", "PT held against members"},
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
		// game_rate (lead Q9): แถวของตารางมีเลข (เริ่ม 1)
		{"steps[0].win_percent", "ขั้นที่ 1 · โอกาสชนะ", "Step 1 · Win chance"},
		{"steps[9].multiplier", "ขั้นที่ 10 · ตัวคูณ", "Step 10 · Multiplier"},
		{"steps[2].draw_percent", "ขั้นที่ 3 · โอกาสเสมอ", "Step 3 · Draw chance"},
		{"prizes[1].share_percent", "รางวัลที่ 2 · สัดส่วนรางวัล", "Prize 2 · Prize share"},
		{"prizes[0].multiplier", "รางวัลที่ 1 · ตัวคูณ", "Prize 1 · Multiplier"},
		{"win_percent", "อัตราชนะ", "Win rate"},
		{"steps", "ตารางขั้น", "Steps"},
		{"prizes", "ตารางรางวัล", "Prizes"},
		{"game_code", "เกม", "Game"},
		{"base_version", "เวอร์ชันที่โหลดมา", "Loaded version"},
		{"status_game", "สถานะเกม", "Game status"},
		{"steps[0].unknown", "steps[0].unknown", "steps[0].unknown"},
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
		{"pt", []string{"pt_from_parent", "commission_percent"}, "PT ที่ปล่อยให้ชั้นล่าง|คอมมิชชั่น", "PT given to the downline|Commission"},
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
