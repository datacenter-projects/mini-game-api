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
	assetController "app/app/internals/backoffice/controllers/asset"
	gameRateController "app/app/internals/backoffice/controllers/game_rate"
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

	// member_management — docs/modules/member_management.md หัวข้อ 3 (lead MN1 · M1–M5) · เมนู member (MGMT-51 · MN4)
	// payment = edit เมื่อส่ง balance ตรวจใน service (MGMT-15A · lead U3)
	pr.Post("/manage/members/create", mw.PassedGates(), memberEdit, memberManagementController.CreateMemberController)
	pr.Post("/manage/members/detail/get", mw.PassedGates(), memberView, memberManagementController.GetMemberDetailController)
	pr.Post("/manage/members/detail/update", mw.PassedGates(), memberEdit, memberManagementController.UpdateMemberDetailController)
	pr.Post("/manage/members/status/update", mw.PassedGates(), memberEdit, memberManagementController.UpdateMemberStatusController)
	pr.Post("/manage/members/list", mw.PassedGates(), memberView, memberManagementController.ListMembersController)

	// sub: เฉพาะบัญชีหลัก (sub เรียก = 402311 — MGMT-40) · ไม่ใช้สิทธิ์เมนู
	pr.Post("/manage/subaccounts/list", mw.PassedGates(), mw.RequireMainAccount(), agentManagementController.ListSubaccountsController)
	pr.Post("/manage/subaccounts/detail/get", mw.PassedGates(), mw.RequireMainAccount(), agentManagementController.GetSubaccountController)
	pr.Post("/manage/subaccounts/create", mw.PassedGates(), mw.RequireMainAccount(), agentManagementController.CreateSubaccountController)
	pr.Post("/manage/subaccounts/detail/update", mw.PassedGates(), mw.RequireMainAccount(), agentManagementController.UpdateSubaccountController)
	pr.Post("/manage/subaccounts/status/update", mw.PassedGates(), mw.RequireMainAccount(), agentManagementController.UpdateSubaccountStatusController)

	// asset — docs/modules/asset.md หัวข้อ 5 (lead GN1–GN22) · ทุกเส้นใช้เมนู asset (GQ3) · view = อ่าน · edit = เขียน
	// เป้าหมาย: asset / theme ส่ง scope + agent_id (ResolveAssetTarget(true)) · branding / bundle ส่ง agent_id อย่างเดียว (false) — AST-08
	// RequireAssetTarget ตรวจสิทธิ์ต่อเป้าหมาย (AST-03 / 04 / 05) · multipart: files/upload · branding/logo/upload (≤ 5 MB — AST-16)
	assetView := mw.RequirePermission(agentManagementCore.MenuAsset, agentManagementCore.LevelView)
	assetEdit := mw.RequirePermission(agentManagementCore.MenuAsset, agentManagementCore.LevelEdit)
	scoped, byAgent := mw.ResolveAssetTarget(true), mw.ResolveAssetTarget(false)
	targetView, targetEdit := mw.RequireAssetTarget(false), mw.RequireAssetTarget(true)
	pr.Post("/asset/games/list", mw.PassedGates(), assetView, assetController.ListGamesController)
	pr.Post("/asset/slots/list", mw.PassedGates(), assetView, assetController.ListSlotsController)
	pr.Post("/asset/files/upload", mw.PassedGates(), assetEdit, scoped, targetEdit, assetController.UploadFileController)
	pr.Post("/asset/files/list", mw.PassedGates(), assetView, scoped, targetView, assetController.ListFilesController)
	pr.Post("/asset/files/delete", mw.PassedGates(), assetEdit, scoped, targetEdit, assetController.DeleteFileController)
	pr.Post("/asset/used/list", mw.PassedGates(), assetView, scoped, targetView, assetController.ListUsedController)
	pr.Post("/asset/used/update", mw.PassedGates(), assetEdit, scoped, targetEdit, assetController.UpdateUsedController)
	pr.Post("/asset/used/reset", mw.PassedGates(), assetEdit, scoped, targetEdit, assetController.ResetUsedController)
	pr.Post("/asset/theme/detail/get", mw.PassedGates(), assetView, scoped, targetView, assetController.GetThemeController)
	pr.Post("/asset/theme/detail/update", mw.PassedGates(), assetEdit, scoped, targetEdit, assetController.UpdateThemeController)
	pr.Post("/asset/theme/reset", mw.PassedGates(), assetEdit, scoped, targetEdit, assetController.ResetThemeController)
	pr.Post("/asset/branding/detail/get", mw.PassedGates(), assetView, byAgent, targetView, assetController.GetBrandingController)
	pr.Post("/asset/branding/detail/update", mw.PassedGates(), assetEdit, byAgent, targetEdit, assetController.UpdateBrandingController)
	pr.Post("/asset/branding/logo/upload", mw.PassedGates(), assetEdit, byAgent, targetEdit, assetController.UploadLogoController)
	pr.Post("/asset/branding/reset", mw.PassedGates(), assetEdit, byAgent, targetEdit, assetController.ResetBrandingController)
	pr.Post("/asset/bundle/detail/get", mw.PassedGates(), assetView, byAgent, targetView, assetController.GetBundleController)
	// game_rate (อัตราแพ้ชนะ · เมนู ⑥) — docs/modules/game_rate.md หัวข้อ 5 (lead RN1) · สิทธิ์ rate (มีแค่ Superadmin + sub ที่ได้เมนูนี้ — RATE-02)
	// บันทึก / ย้อน: สิทธิ์ก่อน passcode
	rateView := mw.RequirePermission(agentManagementCore.MenuRate, agentManagementCore.LevelView)
	rateEdit := mw.RequirePermission(agentManagementCore.MenuRate, agentManagementCore.LevelEdit)
	pr.Post("/rate/list", mw.PassedGates(), rateView, gameRateController.ListRatesController)
	pr.Post("/rate/detail/get", mw.PassedGates(), rateView, gameRateController.GetRateController)
	pr.Post("/rate/detail/check", mw.PassedGates(), rateView, gameRateController.CheckRateController)
	pr.Post("/rate/detail/update", mw.PassedGates(), rateEdit, mw.RequirePasscode(), gameRateController.UpdateRateController)
	pr.Post("/rate/versions/list", mw.PassedGates(), rateView, gameRateController.ListVersionsController)
	pr.Post("/rate/versions/detail/get", mw.PassedGates(), rateView, gameRateController.GetVersionController)
	pr.Post("/rate/versions/rollback", mw.PassedGates(), rateEdit, mw.RequirePasscode(), gameRateController.RollbackRateController)
	pr.Post("/rate/status/update", mw.PassedGates(), rateEdit, mw.RequirePasscode(), gameRateController.SetGameStatusController)
}
