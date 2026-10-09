package agentmanagement

import "regexp"

// เบอร์โทร 2 field (MGMT-08 — lead E1-5 2026-10-09): phone_country_code (รหัสโทรออกของประเทศ) + phone (เบอร์ในประเทศ ไม่มี 0 นำหน้า)
// ใช้กับ agent และ sub · Member ทำใน member_management

// countryCallingCodes — รหัสโทรออกของประเทศ / เขตที่มีจริง (ITU-T E.164) · ไม่รวมรหัสที่ไม่ใช่ประเทศ (800, 808, 870, 878, 881–888, 979)
var countryCallingCodes = map[string]bool{}

func init() {
	for _, c := range []string{
		"1", "7",
		"20", "27", "30", "31", "32", "33", "34", "36", "39", "40", "41", "43", "44", "45", "46", "47", "48", "49",
		"51", "52", "53", "54", "55", "56", "57", "58", "60", "61", "62", "63", "64", "65", "66",
		"81", "82", "84", "86", "90", "91", "92", "93", "94", "95", "98",
		"211", "212", "213", "216", "218",
		"220", "221", "222", "223", "224", "225", "226", "227", "228", "229",
		"230", "231", "232", "233", "234", "235", "236", "237", "238", "239",
		"240", "241", "242", "243", "244", "245", "246", "247", "248", "249",
		"250", "251", "252", "253", "254", "255", "256", "257", "258",
		"260", "261", "262", "263", "264", "265", "266", "267", "268", "269",
		"290", "291", "297", "298", "299",
		"350", "351", "352", "353", "354", "355", "356", "357", "358", "359",
		"370", "371", "372", "373", "374", "375", "376", "377", "378", "379",
		"380", "381", "382", "383", "385", "386", "387", "389",
		"420", "421", "423",
		"500", "501", "502", "503", "504", "505", "506", "507", "508", "509",
		"590", "591", "592", "593", "594", "595", "596", "597", "598", "599",
		"670", "672", "673", "674", "675", "676", "677", "678", "679",
		"680", "681", "682", "683", "685", "686", "687", "688", "689", "690", "691", "692",
		"850", "852", "853", "855", "856", "880", "886",
		"960", "961", "962", "963", "964", "965", "966", "967", "968",
		"970", "971", "972", "973", "974", "975", "976", "977",
		"992", "993", "994", "995", "996", "998",
	} {
		countryCallingCodes[c] = true
	}
}

// PhoneMaxDigits — รหัสประเทศ + เบอร์ในประเทศ รวมไม่เกิน 15 หลัก (E.164)
const PhoneMaxDigits = 15

var nationalPhoneRe = regexp.MustCompile(`^[1-9][0-9]*$`)

// PhoneViolation — ผลตรวจเบอร์ 2 field (MGMT-08)
type PhoneViolation int

const (
	PhoneOK            PhoneViolation = iota
	PhoneNotPaired                    // ส่งค่าเดียว (ต้องว่างทั้งคู่ หรือมีทั้งคู่)
	PhoneCodeInvalid                  // รหัสประเทศไม่มีจริง
	PhoneNumberInvalid                // ไม่ใช่ตัวเลขล้วน หรือมี 0 นำหน้า
	PhoneTooLong                      // รวมเกิน 15 หลัก
)

// IsCountryCallingCode — รหัสโทรออกที่มีจริง (ตัวเลข 1–3 หลัก ไม่มี +)
func IsCountryCallingCode(code string) bool { return countryCallingCodes[code] }

// CheckPhone — ว่างทั้งคู่ = ไม่กรอก (OK) · ตรวจตามลำดับ คู่ → รหัสประเทศ → เบอร์ → ความยาวรวม
func CheckPhone(code, phone string) PhoneViolation {
	if code == "" && phone == "" {
		return PhoneOK
	}
	if code == "" || phone == "" {
		return PhoneNotPaired
	}
	if !IsCountryCallingCode(code) {
		return PhoneCodeInvalid
	}
	if !nationalPhoneRe.MatchString(phone) {
		return PhoneNumberInvalid
	}
	if len(code)+len(phone) > PhoneMaxDigits {
		return PhoneTooLong
	}
	return PhoneOK
}
