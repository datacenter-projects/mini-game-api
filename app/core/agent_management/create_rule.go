// Package agentmanagement คือกฎของการจัดการสมาชิกแบบ pure function — spec: docs/modules/agent_management.md
package agentmanagement

import (
	"regexp"
	"strings"

	"app/app/models"
)

// UserType — ประเภทบัญชีใน API (MGMT-01) · ตรงกับ account ACC-12
type UserType string

const (
	UserTypeSuperadmin              UserType = "SUPERADMIN"
	UserTypeAdmin                   UserType = "ADMIN"
	UserTypeCompanyTransfer         UserType = "COMPANY_TRANSFER"
	UserTypeCompanySeamlessReseller UserType = "COMPANY_SEAMLESS_RESELLER"
	UserTypeCompanySeamlessMaster   UserType = "COMPANY_SEAMLESS_MASTER"
	UserTypeCompanySeamless1to1     UserType = "COMPANY_SEAMLESS_1TO1"
	UserTypeShareB2B                UserType = "SHARE_B2B"
	UserTypeShareB2C                UserType = "SHARE_B2C"
	UserTypeShareReseller           UserType = "SHARE_RESELLER"
	UserTypeShareMaster             UserType = "SHARE_MASTER"
	UserTypeAgent                   UserType = "AGENT"
	UserTypeMember                  UserType = "MEMBER"
)

var userTypeOf = map[models.AgentType]UserType{
	models.AgentTypeTransfer:         UserTypeCompanyTransfer,
	models.AgentTypeSeamlessReseller: UserTypeCompanySeamlessReseller,
	models.AgentTypeSeamlessMaster:   UserTypeCompanySeamlessMaster,
	models.AgentTypeSeamless1to1:     UserTypeCompanySeamless1to1,
	models.AgentTypeShareB2B:         UserTypeShareB2B,
	models.AgentTypeShareB2C:         UserTypeShareB2C,
	models.AgentTypeShareReseller:    UserTypeShareReseller,
	models.AgentTypeShareMaster:      UserTypeShareMaster,
}

// UserTypeOf — ประเภทบัญชีจาก role + agent_type · Company / Share ที่ยังไม่มี agent_type = ""
func UserTypeOf(role models.AgentRole, agentType *models.AgentType) UserType {
	switch role {
	case models.AgentRoleSuperAdmin:
		return UserTypeSuperadmin
	case models.AgentRoleAdmin:
		return UserTypeAdmin
	case models.AgentRoleAgent:
		return UserTypeAgent
	}
	if agentType == nil {
		return ""
	}
	return userTypeOf[*agentType]
}

// NewAccount — ผลของการตัดสินว่าผู้สร้างสร้างประเภทนี้ได้ไหม
type NewAccount struct {
	Role      models.AgentRole  // ไม่ใช้เมื่อเป็น Member
	AgentType *models.AgentType // nil = Agent
	UserType  UserType          // ประเภทที่เก็บจริง (Share ใต้ Seamless Reseller / Master เปลี่ยนเป็น SHARE_RESELLER / MASTER)
}

func ptr(t models.AgentType) *models.AgentType { return &t }

// ResolveNewAgent — กฎ MGMT-02 สำหรับเส้นสร้างฝั่ง agent · ok = false = สร้างประเภทนี้ไม่ได้ (402301)
func ResolveNewAgent(creator UserType, requested UserType) (NewAccount, bool) {
	switch creator {
	case UserTypeSuperadmin:
		switch requested {
		case UserTypeCompanyTransfer:
			return NewAccount{models.AgentRoleCompany, ptr(models.AgentTypeTransfer), requested}, true
		case UserTypeCompanySeamlessReseller:
			return NewAccount{models.AgentRoleCompany, ptr(models.AgentTypeSeamlessReseller), requested}, true
		case UserTypeCompanySeamlessMaster:
			return NewAccount{models.AgentRoleCompany, ptr(models.AgentTypeSeamlessMaster), requested}, true
		case UserTypeCompanySeamless1to1:
			return NewAccount{models.AgentRoleCompany, ptr(models.AgentTypeSeamless1to1), requested}, true
		}
	case UserTypeCompanyTransfer:
		switch requested {
		case UserTypeShareB2B:
			return NewAccount{models.AgentRoleShareholder, ptr(models.AgentTypeShareB2B), requested}, true
		case UserTypeShareB2C:
			return NewAccount{models.AgentRoleShareholder, ptr(models.AgentTypeShareB2C), requested}, true
		}
	case UserTypeCompanySeamlessReseller:
		if requested == UserTypeShareB2C {
			return NewAccount{models.AgentRoleShareholder, ptr(models.AgentTypeShareReseller), UserTypeShareReseller}, true
		}
	case UserTypeCompanySeamlessMaster:
		if requested == UserTypeShareB2C {
			return NewAccount{models.AgentRoleShareholder, ptr(models.AgentTypeShareMaster), UserTypeShareMaster}, true
		}
	case UserTypeShareB2B, UserTypeShareB2C, UserTypeShareReseller, UserTypeShareMaster, UserTypeAgent:
		if requested == UserTypeAgent {
			return NewAccount{Role: models.AgentRoleAgent, UserType: UserTypeAgent}, true
		}
	}
	return NewAccount{}, false
}

// CanCreateMember — กฎ MGMT-02 สำหรับเส้นสร้าง Member
func CanCreateMember(creator UserType) bool {
	switch creator {
	case UserTypeCompanySeamless1to1, UserTypeShareB2C, UserTypeShareReseller, UserTypeShareMaster, UserTypeAgent:
		return true
	}
	return false
}

// IsSeamless — บัญชีฝั่ง Seamless ไม่มียอดเงิน (MGMT-15) · ตัดสินจาก Company หัวสาย
func IsSeamless(companyType UserType) bool {
	switch companyType {
	case UserTypeCompanySeamlessReseller, UserTypeCompanySeamlessMaster, UserTypeCompanySeamless1to1:
		return true
	}
	return false
}

var (
	usernameRe = regexp.MustCompile(`^[a-z0-9]{3,32}$`)
	nameRe     = regexp.MustCompile(`^[\p{Thai}A-Za-z0-9]{3,32}$`) // นับเป็นตัวอักษร (rune) ไม่ใช่ byte
	phoneRe    = regexp.MustCompile(`^[0-9]{8,15}$`)
	subNameRe  = regexp.MustCompile(`^[a-z0-9]{3,20}$`)
)

// NormalizeUsername — trim + ตัวพิมพ์เล็ก (MGMT-05, MGMT-41 · AUTH-01)
func NormalizeUsername(s string) string { return strings.ToLower(strings.TrimSpace(s)) }

// IsValidUsername — 3–32 ตัว a-z 0-9 หลัง normalize (MGMT-05)
func IsValidUsername(s string) bool { return usernameRe.MatchString(s) }

// IsValidName — ชื่อ / ชื่อเล่น 3–32 ตัวอักษร ไทย อังกฤษ ตัวเลข ไม่มีช่องว่าง (MGMT-07, MGMT-41 — แก้ 2026-10-08)
func IsValidName(s string) bool { return nameRe.MatchString(s) }

// IsValidPhone — ว่าง = ไม่กรอก · ไม่ว่าง = ตัวเลข 8–15 ตัว (MGMT-08)
func IsValidPhone(s string) bool { return s == "" || phoneRe.MatchString(s) }

// IsValidSubName — ส่วนหลัง @ ของ sub 3–20 ตัว a-z 0-9 หลัง normalize (MGMT-41 · AUTH-18)
func IsValidSubName(s string) bool { return subNameRe.MatchString(s) }

// CreatableTypes — user_type ที่ผู้สร้างส่งมาในเส้นสร้างฝั่ง agent ได้ (MGMT-02) · ใช้บอกใน msg ของ 402301
// ไม่รวม Member (เส้นของตัวเอง) · Share ใต้ Seamless Reseller / Master ส่ง SHARE_B2C
func CreatableTypes(creator UserType) []UserType {
	switch creator {
	case UserTypeSuperadmin:
		return []UserType{UserTypeCompanyTransfer, UserTypeCompanySeamlessReseller, UserTypeCompanySeamlessMaster, UserTypeCompanySeamless1to1}
	case UserTypeCompanyTransfer:
		return []UserType{UserTypeShareB2B, UserTypeShareB2C}
	case UserTypeCompanySeamlessReseller, UserTypeCompanySeamlessMaster:
		return []UserType{UserTypeShareB2C}
	case UserTypeShareB2B, UserTypeShareB2C, UserTypeShareReseller, UserTypeShareMaster, UserTypeAgent:
		return []UserType{UserTypeAgent}
	}
	return nil
}

// creatorOrder — ลำดับผู้สร้างที่ใช้ตอนบอกใน msg (บนลงล่างตามสายงาน)
var creatorOrder = []UserType{UserTypeSuperadmin, UserTypeCompanyTransfer, UserTypeCompanySeamlessReseller, UserTypeCompanySeamlessMaster,
	UserTypeShareB2B, UserTypeShareB2C, UserTypeShareReseller, UserTypeShareMaster, UserTypeAgent}

// CreatorsOf — ผู้สร้างที่ส่ง user_type นี้ในเส้นสร้างฝั่ง agent ได้ (MGMT-02) · ใช้บอกเหตุผลใน msg ของ 402301
// nil = ไม่มีใครส่งค่านี้ได้ (ไม่มีประเภทนี้ / MEMBER / SUPERADMIN / ADMIN / SHARE_RESELLER / SHARE_MASTER)
func CreatorsOf(t UserType) []UserType {
	var out []UserType
	for _, c := range creatorOrder {
		for _, nt := range CreatableTypes(c) {
			if nt == t {
				out = append(out, c)
			}
		}
	}
	return out
}
