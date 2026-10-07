package agentmanagement

import "app/app/models"

// สิทธิ์ต่อเมนู (MGMT-50 – MGMT-53)

// Menu — เมนูที่ใช้กำหนดสิทธิ์
type Menu string

const (
	MenuDashboard    Menu = "dashboard"
	MenuAccount      Menu = "account"
	MenuMember       Menu = "member"
	MenuPT           Menu = "pt"
	MenuReport       Menu = "report"
	MenuBetCancel    Menu = "bet_cancel"
	MenuPayment      Menu = "payment"
	MenuAsset        Menu = "asset"
	MenuAnnouncement Menu = "announcement"
	MenuRate         Menu = "rate"
)

// Level — ระดับสิทธิ์ · edit รวม view
type Level string

const (
	LevelOff  Level = "off"
	LevelView Level = "view"
	LevelEdit Level = "edit"
)

func (l Level) rank() int {
	switch l {
	case LevelView:
		return 1
	case LevelEdit:
		return 2
	}
	return 0
}

// Allows — ระดับนี้ทำงานที่ต้องการระดับ need ได้ไหม
func (l Level) Allows(need Level) bool { return l.rank() >= need.rank() }

// IsValidLevel — รับเฉพาะ off / view / edit ตัวพิมพ์เล็ก (MGMT-50)
func IsValidLevel(l Level) bool { return l == LevelOff || l == LevelView || l == LevelEdit }

// viewOnlyMenus — เมนูที่มีแค่ off / view (MGMT-51)
var viewOnlyMenus = map[Menu]bool{MenuDashboard: true, MenuReport: true}

// MaxLevel — ระดับสูงสุดของเมนู
func MaxLevel(m Menu) Level {
	if viewOnlyMenus[m] {
		return LevelView
	}
	return LevelEdit
}

var (
	superadminMenus = []Menu{MenuDashboard, MenuAccount, MenuMember, MenuPT, MenuReport, MenuBetCancel, MenuPayment, MenuAsset, MenuRate}
	agentMenus      = []Menu{MenuDashboard, MenuAccount, MenuMember, MenuPT, MenuReport, MenuBetCancel, MenuPayment, MenuAsset, MenuAnnouncement}
)

// MenusForRole — เมนูที่บัญชีหลักแต่ละ role มี (MGMT-52) · ADMIN ไม่ใช้ระบบนี้ (AUTH-44) = nil
func MenusForRole(role models.AgentRole) []Menu {
	switch role {
	case models.AgentRoleSuperAdmin:
		return superadminMenus
	case models.AgentRoleCompany, models.AgentRoleShareholder, models.AgentRoleAgent:
		return agentMenus
	}
	return nil
}

// FullPermissions — บัญชีหลักได้ระดับสูงสุดทุกเมนูของตัวเอง (MGMT-50, MGMT-53)
func FullPermissions(role models.AgentRole) map[Menu]Level {
	out := map[Menu]Level{}
	for _, m := range MenusForRole(role) {
		out[m] = MaxLevel(m)
	}
	return out
}

// PermissionViolation — ผลการตรวจสิทธิ์ที่เจ้าของให้ sub
type PermissionViolation struct {
	Menu   string // เมนูที่ผิด
	Reason string // "menu" = ไม่มีเมนูนี้ · "level" = ระดับไม่ถูกต้อง
}

// NormalizeSubPermissions — ตรวจสิทธิ์ที่ส่งมาตอนสร้าง / แก้ sub แล้วเติมเมนูที่ไม่ส่งเป็น off (MGMT-50, MGMT-52)
// ok = false → v บอก field ที่ผิด (422)
func NormalizeSubPermissions(ownerRole models.AgentRole, in map[string]string) (out map[Menu]Level, v PermissionViolation, ok bool) {
	menus := MenusForRole(ownerRole)
	allowed := make(map[Menu]bool, len(menus))
	for _, m := range menus {
		allowed[m] = true
	}
	out = make(map[Menu]Level, len(menus))
	for _, m := range menus {
		out[m] = LevelOff
	}
	for k, val := range in {
		m, l := Menu(k), Level(val)
		if !allowed[m] {
			return nil, PermissionViolation{k, "menu"}, false
		}
		if !IsValidLevel(l) || !MaxLevel(m).Allows(l) {
			return nil, PermissionViolation{k, "level"}, false
		}
		out[m] = l
	}
	return out, PermissionViolation{}, true
}

// SubPermissionsView — สิทธิ์ของ sub สำหรับแสดง: ครบทุกเมนูของเจ้าของ · ไม่มี key / ค่าผิด = off (MGMT-50)
func SubPermissionsView(ownerRole models.AgentRole, stored map[string]string) map[Menu]Level {
	out := map[Menu]Level{}
	for _, m := range MenusForRole(ownerRole) {
		l := Level(stored[string(m)])
		if !IsValidLevel(l) || !MaxLevel(m).Allows(l) {
			l = LevelOff
		}
		out[m] = l
	}
	return out
}
