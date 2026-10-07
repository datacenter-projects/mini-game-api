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
	agentAuthController "app/app/internals/backoffice/controllers/agent_auth"
	agentManagementController "app/app/internals/backoffice/controllers/agent_management"
	mw "app/app/internals/backoffice/middleware"
	"app/app/models"

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

	// ทุก route ใต้ /pr ต้องมี mw.PassedGates(...) (ด่านหลัง login — AUTH-29) บรรทัดเดียวกับ route
	pr := bo.Group("/pr", mw.Authenticated())

	// agent_auth phase 2 — docs/modules/agent_auth_phase2.md หัวข้อ 5
	pr.Post("/auth/passcode/setup", mw.PassedGates(agentAuthCore.GateSetupPasscode), agentAuthController.SetupPasscodeController)
	pr.Post("/auth/passcode/change", mw.PassedGates(agentAuthCore.GateChangePasscode), agentAuthController.ChangePasscodeController)
	pr.Post("/auth/password/change", mw.PassedGates(agentAuthCore.GateChangePassword), mw.RequirePasscodeUnlessMustChangePassword(), agentAuthController.ChangePasswordController)
	pr.Post("/admin/passcode/reset", mw.PassedGates(), mw.RequireRole(models.AgentRoleAdmin), mw.RequirePasscode(), agentAuthController.ResetPasscodeController)
	pr.Post("/admin/password/reset", mw.PassedGates(), mw.RequireRole(models.AgentRoleAdmin), mw.RequirePasscode(), agentAuthController.ResetPasswordController)

	// agent_management — docs/modules/agent_management.md หัวข้อ 5 · payment = edit เมื่อส่ง balance ตรวจใน service (MGMT-51)
	memberEdit := mw.RequirePermission(agentManagementCore.MenuMember, agentManagementCore.LevelEdit)
	ptEdit := mw.RequirePermission(agentManagementCore.MenuPT, agentManagementCore.LevelEdit)
	pr.Post("/manage/agents", mw.PassedGates(), memberEdit, ptEdit, agentManagementController.CreateAgentController)
	pr.Post("/manage/members", mw.PassedGates(), memberEdit, ptEdit, agentManagementController.CreateMemberController)
	memberView := mw.RequirePermission(agentManagementCore.MenuMember, agentManagementCore.LevelView)
	ptView := mw.RequirePermission(agentManagementCore.MenuPT, agentManagementCore.LevelView)
	pr.Get("/manage/downlines", mw.PassedGates(), memberView, agentManagementController.ListDownlinesController)
	pr.Get("/manage/agents/copy-sources", mw.PassedGates(), ptView, agentManagementController.ListCopySourcesController) // ก่อน /agents/:id
	pr.Get("/manage/agents/:id", mw.PassedGates(), memberView, agentManagementController.GetAgentDetailController)
	pr.Get("/manage/members/:id", mw.PassedGates(), memberView, agentManagementController.GetMemberDetailController)
}
