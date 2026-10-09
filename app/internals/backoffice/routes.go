// Package backoffice คือ HTTP layer ของหลังบ้าน agent/subaccount
//
//	controllers/{module}/  handler — parse request → เรียก service → ตอบ response
//	dto/{module}/          request/response struct
//	middleware/            auth + สิทธิ์ที่ใช้เฉพาะหลังบ้าน
//	routes.go              ประกาศทุก route ของหลังบ้าน (ไฟล์นี้)
//
// อ่านไฟล์นี้ไฟล์เดียวต้องรู้ได้ว่า endpoint ไหนต้องผ่าน middleware/สิทธิ์อะไรบ้าง
// จึงให้ใส่ middleware ต่อ route ไว้บรรทัดเดียวกับ route เสมอ
package backoffice

import (
	agentAuthCore "app/app/core/agent_auth"
	agentManagementCore "app/app/core/agent_management"
	accountController "app/app/internals/backoffice/controllers/account"
	adminManagementController "app/app/internals/backoffice/controllers/admin_management"
	agentAuthController "app/app/internals/backoffice/controllers/agent_auth"
	agentManagementController "app/app/internals/backoffice/controllers/agent_management"
	memberManagementController "app/app/internals/backoffice/controllers/member_management"
	mw "app/app/internals/backoffice/middleware"
	adminMw "app/app/internals/backoffice/middleware/admin_management"

	"github.com/gofiber/fiber/v2"
)

func RegisterRoutes(api fiber.Router) {
	bo := api.Group("/bo")

	// ---- public: ไม่ต้อง login ----
	pb := bo.Group("/pb")
	pb.Post("/auth/login", agentAuthController.LoginController)

	// ---- private: ต้อง login ----
	// logout ต้องลงทะเบียนก่อน group /pr ที่มี mw.Authenticated() เพื่อไม่ให้ถูกบังคับผ่าน middleware
	// (session ที่หลุดไปแล้วต้อง logout สำเร็จได้ — AUTH-09)
	bo.Post("/pr/auth/logout", agentAuthController.LogoutController)

	// ทุก route ใต้ /pr ต้องมี mw.PassedGates(...) (ด่านหลัง login — AUTH-29 · ปฏิเสธบัญชีถูกระงับ — AUTH-54) บรรทัดเดียวกับ route
	// route ดูข้อมูลของ Profile / Report ที่บัญชีถูกระงับเข้าได้ใช้ mw.PassedGatesAllowSuspended(...) แทน
	pr := bo.Group("/pr", mw.Authenticated())

	// agent_auth phase 2 — docs/modules/agent_auth_phase2.md หัวข้อ 5
	pr.Post("/auth/passcode/setup", mw.PassedGates(agentAuthCore.GateSetupPasscode), agentAuthController.SetupPasscodeController)
	pr.Post("/auth/passcode/change", mw.PassedGates(agentAuthCore.GateChangePasscode), agentAuthController.ChangePasscodeController)
	pr.Post("/auth/password/change", mw.PassedGates(agentAuthCore.GateChangePassword), mw.RequirePasscodeUnlessMustChangePassword(), agentAuthController.ChangePasswordController)

	// admin_management — docs/modules/admin_management.md · ADMIN เท่านั้น (adminMw.RequireAdmin — AUTH-44)
	pr.Post("/admin/passcode/reset", mw.PassedGates(), adminMw.RequireAdmin(), mw.RequirePasscode(), adminManagementController.ResetPasscodeController)
	pr.Post("/admin/password/reset", mw.PassedGates(), adminMw.RequireAdmin(), mw.RequirePasscode(), adminManagementController.ResetPasswordController)
	pr.Post("/admin/accounts/search", mw.PassedGates(), adminMw.RequireAdmin(), adminManagementController.SearchAccountsController) // MGMT-27B

	// account — docs/modules/account.md หัวข้อ 5 · Profile เปิดได้เสมอไม่ต้องมีสิทธิ์ (lead A2 / A5) · 1.3 sub ต้องมีสิทธิ์ api_credential (ACC-02 · lead A6)
	// Profile และดู 1.3 เปิดได้ตอนบัญชีถูกระงับ (ACC-11, ACC-31, AUTH-54)
	pr.Get("/account/profile", mw.PassedGatesAllowSuspended(), accountController.GetProfileController)
	pr.Get("/account/api-credential", mw.PassedGatesAllowSuspended(), mw.RequireSubPermission(agentManagementCore.MenuAPICredential, agentManagementCore.LevelView), accountController.GetAPICredentialController) // ACC-31: ถูกระงับดูได้ บันทึกไม่ได้
	pr.Post("/account/update-credential", mw.PassedGates(), mw.RequireSubPermission(agentManagementCore.MenuAPICredential, agentManagementCore.LevelEdit), mw.RequirePasscode(), accountController.SaveAPICredentialController)

	// agent_management — docs/modules/agent_management.md หัวข้อ 5 (lead E1–E6 · N1) · ทุกเส้นใช้เมนู member (MGMT-51 · P3–P5)
	// payment = edit เมื่อส่ง balance ตรวจใน service (MGMT-15A · 7.1)
	memberView := mw.RequirePermission(agentManagementCore.MenuMember, agentManagementCore.LevelView)
	memberEdit := mw.RequirePermission(agentManagementCore.MenuMember, agentManagementCore.LevelEdit)
	pr.Post("/manage/agents/create", mw.PassedGates(), memberEdit, agentManagementController.CreateAgentController)
	pr.Post("/manage/downlines/list", mw.PassedGates(), memberView, agentManagementController.ListDownlinesController)
	pr.Post("/manage/agents/detail/get", mw.PassedGates(), memberView, agentManagementController.GetAgentDetailController)
	pr.Post("/manage/agents/detail/update", mw.PassedGates(), memberEdit, agentManagementController.UpdateAgentDetailController)
	pr.Post("/manage/agents/status/update", mw.PassedGates(), memberEdit, agentManagementController.UpdateAgentStatusController)
	pr.Post("/manage/agents/list", mw.PassedGates(), memberView, agentManagementController.ListAgentsController)

	// member_management — ไม่มีเมนู pt แล้ว: เส้นที่เคยใช้ pt edit เปลี่ยนเป็น member edit (lead B3 · path ยังไม่เปลี่ยน)
	pr.Post("/manage/members/create", mw.PassedGates(), memberEdit, memberManagementController.CreateMemberController)
	pr.Post("/manage/members/detail", mw.PassedGates(), memberView, memberManagementController.GetMemberDetailController)
	pr.Post("/manage/members/update-info", mw.PassedGates(), memberEdit, memberManagementController.UpdateMemberInfoController)
	pr.Post("/manage/members/update-status", mw.PassedGates(), memberEdit, memberManagementController.UpdateMemberStatusController)
	pr.Post("/manage/members/update-pt", mw.PassedGates(), memberEdit, memberManagementController.UpdateMemberPTController)

	// sub: เฉพาะบัญชีหลัก (sub เรียก = 402311 — MGMT-40) · ไม่ใช้สิทธิ์เมนู
	pr.Post("/manage/subaccounts/list", mw.PassedGates(), mw.RequireMainAccount(), agentManagementController.ListSubaccountsController)
	pr.Post("/manage/subaccounts/detail/get", mw.PassedGates(), mw.RequireMainAccount(), agentManagementController.GetSubaccountController)
	pr.Post("/manage/subaccounts/create", mw.PassedGates(), mw.RequireMainAccount(), agentManagementController.CreateSubaccountController)
	pr.Post("/manage/subaccounts/detail/update", mw.PassedGates(), mw.RequireMainAccount(), agentManagementController.UpdateSubaccountController)
	pr.Post("/manage/subaccounts/status/update", mw.PassedGates(), mw.RequireMainAccount(), agentManagementController.UpdateSubaccountStatusController)
}
