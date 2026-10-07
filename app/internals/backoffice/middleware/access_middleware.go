package middleware

import (
	agentAuthCore "app/app/core/agent_auth"
	agentManagementCore "app/app/core/agent_management"
	agentAuthDto "app/app/internals/backoffice/dto/agent_auth"
	"app/app/models"
	agentAuthService "app/app/service/agent_auth"
	agentManagementService "app/app/service/agent_management"
	"app/pkg/apperr"
	"app/pkg/response"
	"app/pkg/utils"

	"github.com/gofiber/fiber/v2"
)

// middleware ในไฟล์นี้ต้องอยู่หลัง Authenticated เสมอ — spec: docs/modules/agent_auth_phase2.md หัวข้อ 5

// PassedGates — ด่านหลัง login (AUTH-29) ต้องผ่านครบ ยกเว้นด่านที่ route นี้เป็นทางผ่าน
//
//	pr.Get("/x", mw.PassedGates(), ...)                                     // route ทั่วไป
//	pr.Post("/auth/passcode/setup", mw.PassedGates(agentAuthCore.GateSetupPasscode), ...)
func PassedGates(allow ...agentAuthCore.Gate) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if err := agentAuthService.CheckGateService(GetActor(c), allow...); err != nil {
			return response.Error(c, err)
		}
		return c.Next()
	}
}

// RequireRole — เฉพาะบัญชี agent ที่มี role นี้ (AUTH-44)
func RequireRole(role models.AgentRole) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if err := agentAuthService.CheckRoleService(GetActor(c), role); err != nil {
			return response.Error(c, err)
		}
		return c.Next()
	}
}

// RequirePermission — สิทธิ์ต่อเมนูของบัญชีหลัก / sub (MGMT-50, MGMT-51) · ไม่พอ = 402303
//
//	pr.Get("/manage/downlines", mw.PassedGates(), mw.RequirePermission(agentManagementCore.MenuMember, agentManagementCore.LevelView), ...)
func RequirePermission(menu agentManagementCore.Menu, need agentManagementCore.Level) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if err := agentManagementService.CheckPermissionService(c.UserContext(), GetActor(c), menu, need); err != nil {
			return response.Error(c, err)
		}
		return c.Next()
	}
}

// RequireMainAccount — เฉพาะบัญชีหลัก · sub เรียก = 402311 (agent_management MGMT-40)
func RequireMainAccount() fiber.Handler {
	return func(c *fiber.Ctx) error {
		if err := agentManagementService.CheckMainAccountService(GetActor(c)); err != nil {
			return response.Error(c, err)
		}
		return c.Next()
	}
}

// RequirePasscode — ต้องส่ง passcode ของตัวเองมาใน body (AUTH-33, AUTH-35)
// อ่านจาก body เท่านั้น — ห้ามใช้กับ GET
func RequirePasscode() fiber.Handler {
	return func(c *fiber.Ctx) error {
		if err := verifyPasscodeFromBody(c); err != nil {
			return response.Error(c, err)
		}
		return c.Next()
	}
}

// RequirePasscodeUnlessMustChangePassword — เหมือน RequirePasscode แต่ข้ามตอนถูกบังคับเปลี่ยนรหัสผ่าน
// ใช้กับ route เปลี่ยนรหัสผ่านเท่านั้น (AUTH-41)
func RequirePasscodeUnlessMustChangePassword() fiber.Handler {
	return func(c *fiber.Ctx) error {
		if !GetActor(c).MustChangePassword {
			if err := verifyPasscodeFromBody(c); err != nil {
				return response.Error(c, err)
			}
		}
		return c.Next()
	}
}

func verifyPasscodeFromBody(c *fiber.Ctx) error {
	var body struct {
		Passcode utils.JSONString `json:"passcode"`
	}
	if len(c.Body()) > 0 { // ไม่มี body = ไม่ได้ส่ง passcode → 422 จาก ValidatePasscodeField
		if err := c.BodyParser(&body); err != nil {
			return apperr.ErrBadRequest.Wrap(err)
		}
	}
	if err := agentAuthDto.ValidatePasscodeField("passcode", body.Passcode); err != nil {
		return err
	}
	return agentAuthService.VerifyPasscodeService(c.UserContext(), GetActor(c), body.Passcode.Value, RequestMeta(c))
}
