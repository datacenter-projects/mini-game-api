package utils

import "strings"

// FieldLabel — path ของ field ใน body (เช่น "pt.minigame.pt_from_parent") → ชื่อที่ผู้ใช้อ่านได้ สำหรับขึ้นต้น msg ของ error
// เช่น "PT ที่ให้ (Minigame)" · path ไม่มีจุด หรือไม่อยู่ในตาราง = คืน path เดิม · path คู่ "a / b" แปลงทีละตัว
// ชื่อกลุ่ม / เกม แปลงจากรหัส (coin_toss → Coin Toss) · สกุลเงิน / เมนู ใช้รหัสตามเดิม
func FieldLabel(path string) (th, en string) {
	parts := strings.Split(path, " / ")
	ths, ens := make([]string, len(parts)), make([]string, len(parts))
	for i, p := range parts {
		ths[i], ens[i] = fieldLabel(p)
	}
	return strings.Join(ths, " / "), strings.Join(ens, " / ")
}

// SubFieldLabels — ชื่อ field ใน section (ไม่มีชื่อกลุ่มต่อท้าย) สำหรับใช้ในเนื้อข้อความ
// เช่น ("pt", "pt_from_parent", "force") → ["PT ที่ให้", "Force"] · ไม่รู้จัก = ชื่อ field เดิม
func SubFieldLabels(section string, fields ...string) (th, en []string) {
	th, en = make([]string, len(fields)), make([]string, len(fields))
	for i, f := range fields {
		th[i], en[i] = f, f
		if l, ok := sectionLabels[section][f]; ok {
			th[i], en[i] = l.th, l.en
		}
	}
	return th, en
}

type label struct{ th, en string }

// sectionLabels — section ใน body → field → ชื่อที่แสดง (แก้ชื่อที่นี่ที่เดียว ทั้ง msg และ test ของ FieldLabel)
var sectionLabels = map[string]map[string]label{
	"pt": {
		"pt_from_parent":     {"PT ที่ให้", "PT given"},
		"force":              {"Force", "Force"},
		"remain_quota":       {"Remain", "Remain"},
		"commission_percent": {"Commission", "Commission"},
		"status":             {"รับ PT", "Receives PT"},
		"pt":                 {"PT ที่ถือสู้", "PT held"},
	},
	"info": {
		"name":               {"ชื่อ", "Name"},
		"phone_country_code": {"รหัสประเทศ", "Country code"},
		"phone":              {"เบอร์โทร", "Phone"},
	},
	"payment": {
		"payment_type":    {"ประเภทการชำระ", "Payment type"},
		"payment_account": {"บัญชีรับเงิน", "Payment account"},
	},
	"payment_account": {
		"bank_code":      {"ธนาคาร", "Bank"},
		"account_number": {"เลขบัญชี", "Account number"},
		"name_th":        {"ชื่อ (ไทย)", "First name (Thai)"},
		"midname_th":     {"ชื่อกลาง (ไทย)", "Middle name (Thai)"},
		"surname_th":     {"นามสกุล (ไทย)", "Last name (Thai)"},
		"name_en":        {"ชื่อ (อังกฤษ)", "First name (English)"},
		"midname_en":     {"ชื่อกลาง (อังกฤษ)", "Middle name (English)"},
		"surname_en":     {"นามสกุล (อังกฤษ)", "Last name (English)"},
	},
	"limit": {
		"win_limit_per_day": {"ยอดชนะสูงสุดต่อวัน", "Daily win limit"},
	},
}

func fieldLabel(path string) (string, string) {
	seg := strings.Split(path, ".")
	n := len(seg)
	switch {
	case n == 2 && seg[0] == "pt":
		g := titleCode(seg[1])
		return "PT (" + g + ")", "PT (" + g + ")"
	case n == 3 && seg[0] == "pt":
		if l, ok := sectionLabels["pt"][seg[2]]; ok {
			g := titleCode(seg[1])
			return l.th + " (" + g + ")", l.en + " (" + g + ")"
		}
	case n == 2 && seg[0] == "status_game":
		g := titleCode(seg[1])
		return "สถานะเกม (" + g + ")", "Game status (" + g + ")"
	case n == 2 && seg[0] == "balance":
		return "ยอดเงินตั้งต้น (" + seg[1] + ")", "Initial balance (" + seg[1] + ")"
	case n == 2 && seg[0] == "permissions":
		return "สิทธิ์เมนู " + seg[1], "Permission: " + seg[1]
	case n == 2, n == 3 && seg[0] == "payment" && seg[1] == "payment_account": // info.x · payment.x · payment_account.x · limit.x
		if l, ok := sectionLabels[seg[n-2]][seg[n-1]]; ok {
			return l.th, l.en
		}
	}
	return path, path
}

// titleCode — รหัสกลุ่ม / เกม → ชื่อแสดง เช่น "rock_paper_scissors" → "Rock Paper Scissors"
func titleCode(code string) string {
	words := strings.Split(code, "_")
	for i, w := range words {
		if w != "" {
			words[i] = strings.ToUpper(w[:1]) + w[1:]
		}
	}
	return strings.Join(words, " ")
}
