package middleware

import (
	"encoding/json"
	"strings"

	assetCore "app/app/core/asset"
	assetDto "app/app/internals/backoffice/dto/asset"
	assetService "app/app/service/asset"
	"app/pkg/apperr"
	"app/pkg/response"
	"app/pkg/utils"

	"github.com/gofiber/fiber/v2"
)

// เป้าหมายของ module asset — docs/modules/asset.md AST-02 – AST-09
// ลำดับใน routes.go: Authenticated → PassedGates → RequirePermission(MenuAsset) → ResolveAssetTarget → RequireAssetTarget
// BO ระบุบัญชีอื่นไม่ได้ (lead 2026-10-16): เป้าหมาย AGENT = บัญชีที่ login (sub = เจ้าของ) · ไม่รับ agent_id

const assetTargetLocalsKey = "asset_target"

// ResolveAssetTarget — อ่าน scope จาก body (JSON) หรือ form field (multipart) แล้วเก็บไว้ให้ RequireAssetTarget (AST-08)
//
//	withScope = true : ส่ง scope = "GLOBAL" / "AGENT" บังคับ (asset / theme)
//	withScope = false: เส้น branding / bundle ไม่มี scope (ของตัวเองเสมอ)
//
// ส่ง agent_id มา = 422 · ไม่ส่ง scope / ค่าอื่น / อ่านไม่ออก = 422 — ห้ามตีความค่าว่างเป็น GLOBAL เงียบๆ
// ใช้ได้กับเส้นที่ส่ง body เท่านั้น (ทุกเส้นของ module เป็น POST)
func ResolveAssetTarget(withScope bool) fiber.Handler {
	return func(c *fiber.Ctx) error {
		scope, agentIDSent, err := readTargetFields(c)
		if err != nil {
			return response.Error(c, err)
		}
		target, err := assetDto.ResolveTarget(withScope, scope, agentIDSent)
		if err != nil {
			return response.Error(c, err)
		}
		c.Locals(assetTargetLocalsKey, target)
		return c.Next()
	}
}

// RequireAssetTarget — เติมบัญชีจาก token + สาย แล้วตรวจสิทธิ์ต่อ scope (AST-02 – AST-05) · write = true ต้องมีสิทธิ์แก้
// ต้องอยู่หลัง ResolveAssetTarget · ไม่มี scope ที่อ่านแล้ว = ปฏิเสธ (fail-closed — AST-09)
func RequireAssetTarget(write bool) fiber.Handler {
	return func(c *fiber.Ctx) error {
		target, err := assetService.CheckAssetTargetService(c.UserContext(), GetActor(c), GetAssetTarget(c), write)
		if err != nil {
			return response.Error(c, err)
		}
		c.Locals(assetTargetLocalsKey, target) // Resolved · บัญชีจาก token · Chain ที่โหลดแล้ว — service ไม่ต้องอ่านสายซ้ำ
		return c.Next()
	}
}

// GetAssetTarget — เป้าหมายของ request · Resolved = false = route ไม่ได้ผ่าน RequireAssetTarget (service ปฏิเสธ — AST-09)
func GetAssetTarget(c *fiber.Ctx) assetCore.Target {
	t, _ := c.Locals(assetTargetLocalsKey).(assetCore.Target)
	return t
}

// readTargetFields — scope ดิบ + มี field agent_id มาหรือไม่ จาก JSON body หรือ multipart form (ไม่ตีความ — ResolveTarget ตัดสิน)
func readTargetFields(c *fiber.Ctx) (utils.JSONString, bool, error) {
	if strings.HasPrefix(strings.ToLower(c.Get(fiber.HeaderContentType)), fiber.MIMEMultipartForm) {
		// อ่านจาก form field ของ multipart เท่านั้น — c.FormValue อ่านจาก query string ด้วย ซึ่งสเปกไม่รับ (lead GQ4)
		form, err := c.MultipartForm()
		if err != nil {
			return utils.JSONString{}, false, apperr.ErrBadRequest.Wrap(err)
		}
		var scope utils.JSONString
		if v := form.Value["scope"]; len(v) > 0 && v[0] != "" {
			scope = utils.JSONString{Value: v[0], Present: true, IsString: true}
		}
		_, agentIDSent := form.Value["agent_id"]
		return scope, agentIDSent, nil
	}

	body := c.Body()
	if len(body) == 0 {
		return utils.JSONString{}, false, nil
	}
	if err := utils.CheckNoNull(body); err != nil { // ไม่มี null ใน API (ACC-32) — ข้อความเดียวกับ ParseBodyNoNull
		return utils.JSONString{}, false, err
	}
	var in struct {
		Scope   utils.JSONString `json:"scope"`
		AgentID json.RawMessage  `json:"agent_id"`
	}
	if err := json.Unmarshal(body, &in); err != nil {
		return utils.JSONString{}, false, apperr.ErrBadRequest.Wrap(err)
	}
	return in.Scope, len(in.AgentID) > 0, nil
}
