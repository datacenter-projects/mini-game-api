package agentmanagement

import (
	agentAuthCore "app/app/core/agent_auth"
	agentManagementCore "app/app/core/agent_management"
	agentAuthDto "app/app/internals/backoffice/dto/agent_auth"
	"app/pkg/utils"
)

// request / response ของ sub (phase 5) — spec: docs/modules/agent_management.md MGMT-40 – MGMT-46
// เฉพาะบัญชีหลัก (middleware RequireMainAccount) · id ของ sub ส่งใน body · ผู้เรียกมาจาก token

// SubStatusInactive — สถานะ INACTIVE ใน API = SUSPENDED ในตาราง (MGMT-43)
const SubStatusInactive = "INACTIVE"

// SubView — 1 แถวของรายชื่อ sub และรายละเอียด sub (MGMT-46)
type SubView struct {
	ID               uint              `json:"id"`
	Username         string            `json:"username"`
	Name             string            `json:"name"`
	PhoneCountryCode string            `json:"phone_country_code"` // MGMT-08
	Phone            string            `json:"phone"`
	Status           string            `json:"status"` // ACTIVE / INACTIVE · หรือสถานะของเจ้าของ / หัวสายเมื่อถูกระงับ / ล็อก
	Permissions      map[string]string `json:"permissions"`
	CreatedAt        string            `json:"created_at"`
	LastLoginAt      string            `json:"last_login_at"`
	LastLoginIP      string            `json:"last_login_ip"`
}

// SubListRequest — POST /manage/subaccounts/list · ทุกค่าไม่บังคับ — ไม่กรองให้ส่ง {}
type SubListRequest struct {
	OwnerID uint   `json:"owner_id"` // 0 = ตัวเอง · อื่นต้องอยู่ในสายล่าง
	Keyword string `json:"keyword"`  // ค้น username บางส่วน 4–32 ตัว (MGMT-46 — แบบเดียวกับ downlines/list)
	Page    int    `json:"page"`
	Limit   int    `json:"limit"`
}

func (r *SubListRequest) Validate() error { return NormalizeKeyword(&r.Keyword) }

// SubCreateRequest — POST /manage/subaccounts/create (MGMT-41)
type SubCreateRequest struct {
	NameSuffix       string            `json:"name_suffix"` // ส่วนหลัง @ · Validate แปลงเป็นตัวเล็ก
	Password         string            `json:"password"`
	Name             string            `json:"name"`
	PhoneCountryCode string            `json:"phone_country_code"` // MGMT-08 · ไม่กรอก = "" คู่กับ phone
	Phone            string            `json:"phone"`
	Permissions      map[string]string `json:"permissions"` // ไม่ส่งเมนูไหน = off (MGMT-50)
}

func (r *SubCreateRequest) Validate() error {
	r.NameSuffix = agentManagementCore.NormalizeUsername(r.NameSuffix)
	if !agentManagementCore.IsValidSubName(r.NameSuffix) {
		return Invalid("name_suffix", "ต้องยาว 3–20 ตัว ใช้ได้เฉพาะ a-z และ 0-9", "must be 3–20 characters of a-z and 0-9")
	}
	if r.Password == "" {
		return Invalid("password", "ต้องกรอก", "is required")
	}
	if err := agentAuthDto.PasswordPolicyError("password", agentAuthCore.CheckPasswordPolicy(r.Password)); err != nil {
		return err
	}
	return validateNamePhone(r.Name, &r.PhoneCountryCode, &r.Phone)
}

// SubUpdateRequest — POST /manage/subaccounts/detail/update (MGMT-42) · แทนทั้งชุด: ต้องส่งครบทุก field
type SubUpdateRequest struct {
	ID               uint               `json:"id"`
	Name             string             `json:"name"`
	PhoneCountryCode utils.JSONString   `json:"phone_country_code"`
	Phone            utils.JSONString   `json:"phone"`
	Permissions      *map[string]string `json:"permissions"`
}

func (r *SubUpdateRequest) Validate() error {
	// ไล่ตามลำดับ field ใน body: id → name → phone_country_code → phone → permissions (7.1)
	if err := CheckID(r.ID); err != nil {
		return err
	}
	if !r.PhoneCountryCode.Present || !r.PhoneCountryCode.IsString {
		return Invalid("phone_country_code", `ต้องส่งเป็นข้อความ (ไม่ตั้งให้ส่ง "")`, `must be a string (send "" to leave it empty)`)
	}
	if !r.Phone.Present || !r.Phone.IsString {
		return Invalid("phone", `ต้องส่งเป็นข้อความ (ไม่ตั้งให้ส่ง "")`, `must be a string (send "" to leave it empty)`)
	}
	if err := validateNamePhone(r.Name, &r.PhoneCountryCode.Value, &r.Phone.Value); err != nil {
		return err
	}
	if r.Permissions == nil {
		return Invalid("permissions", "ต้องส่ง (แทนทั้งชุด — ไม่ให้สิทธิ์ใดให้ส่ง {})", "is required (replaces the whole set — send {} for none)")
	}
	return nil
}

// SubStatusRequest — POST /manage/subaccounts/status/update (MGMT-43)
type SubStatusRequest struct {
	ID     uint   `json:"id"`
	Status string `json:"status"`
}

func (r *SubStatusRequest) Validate() error {
	if err := CheckID(r.ID); err != nil {
		return err
	}
	if r.Status != "ACTIVE" && r.Status != SubStatusInactive {
		return Invalid("status", "ต้องเป็น ACTIVE / INACTIVE", "must be ACTIVE or INACTIVE")
	}
	return nil
}

type SubCreateResponse struct {
	ID       uint   `json:"id"`
	Username string `json:"username"`
}

func validateNamePhone(name string, code, phone *string) error {
	if !agentManagementCore.IsValidName(name) {
		return Invalid("name", "ต้องยาว 3–32 ตัวอักษร ใช้ได้เฉพาะภาษาไทย อังกฤษ และตัวเลข ไม่มีช่องว่าง", "must be 3–32 characters of Thai, English letters or digits, without spaces")
	}
	return ValidatePhonePair("", code, phone) // MGMT-08 · sub ซ้ำได้ (MGMT-41)
}
