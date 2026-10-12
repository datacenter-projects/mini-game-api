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

// เป้าหมายของ module asset — docs/modules/asset.md AST-03 – AST-09
// ลำดับใน routes.go: Authenticated → PassedGates → RequirePermission(MenuAsset) → ResolveAssetTarget → RequireAssetTarget

const assetTargetLocalsKey = "asset_target"

// ResolveAssetTarget — อ่าน scope + agent_id จาก body (JSON) หรือ form field (multipart) แล้วเก็บเป็นเป้าหมาย (AST-08)
//
//	withScope = true : ส่ง scope = "GLOBAL" / "AGENT" บังคับ (asset / theme)
//	withScope = false: เส้น branding / bundle ไม่มี scope (มีแต่ AGENT)
//
// ไม่ส่ง / ค่าอื่น / อ่านไม่ออก = 422 — ห้ามตีความค่าว่างหรือ 0 เป็น GLOBAL เงียบๆ
// ใช้ได้กับเส้นที่ส่ง body เท่านั้น (ทุกเส้นของ module เป็น POST)
func ResolveAssetTarget(withScope bool) fiber.Handler {
	return func(c *fiber.Ctx) error {
		scope, agentID, err := readTargetFields(c)
		if err != nil {
			return response.Error(c, err)
		}
		target, err := assetDto.ResolveTarget(withScope, scope, agentID)
		if err != nil {
			return response.Error(c, err)
		}
		c.Locals(assetTargetLocalsKey, target)
		return c.Next()
	}
}

// RequireAssetTarget — สิทธิ์ต่อเป้าหมาย (AST-03 / AST-04 / AST-05) · write = true ต้องมีสิทธิ์แก้ · ต้องอยู่หลัง ResolveAssetTarget
// ไม่เจอเป้าหมายที่ resolve แล้ว = ปฏิเสธ (fail-closed — AST-09)
func RequireAssetTarget(write bool) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if err := assetService.CheckAssetTargetService(c.UserContext(), GetActor(c), GetAssetTarget(c), write); err != nil {
			return response.Error(c, err)
		}
		return c.Next()
	}
}

// GetAssetTarget — เป้าหมายของ request · ค่า zero (Resolved = false) = route ไม่ได้ผ่าน ResolveAssetTarget
func GetAssetTarget(c *fiber.Ctx) assetCore.Target {
	t, _ := c.Locals(assetTargetLocalsKey).(assetCore.Target)
	return t
}

// readTargetFields — scope กับ agent_id ดิบจาก JSON body หรือ multipart form (ไม่ตีความ — ResolveTarget ตัดสิน)
func readTargetFields(c *fiber.Ctx) (utils.JSONString, string, error) {
	if strings.HasPrefix(strings.ToLower(c.Get(fiber.HeaderContentType)), fiber.MIMEMultipartForm) {
		// อ่านจาก form field ของ multipart เท่านั้น — c.FormValue อ่านจาก query string ด้วย ซึ่งสเปกไม่รับ (lead GQ4)
		form, err := c.MultipartForm()
		if err != nil {
			return utils.JSONString{}, "", apperr.ErrBadRequest.Wrap(err)
		}
		first := func(key string) string {
			if v := form.Value[key]; len(v) > 0 {
				return v[0]
			}
			return ""
		}
		var scope utils.JSONString
		if v := first("scope"); v != "" {
			scope = utils.JSONString{Value: v, Present: true, IsString: true}
		}
		return scope, first("agent_id"), nil
	}

	body := c.Body()
	if len(body) == 0 {
		return utils.JSONString{}, "", nil
	}
	if path, ok := utils.FindJSONNull(body); ok { // ไม่มี null ใน API (ACC-32)
		if path == "" {
			path = "body"
		}
		return utils.JSONString{}, "", apperr.ErrValidation.WithMessage(path+" ห้ามเป็น null", path+" must not be null")
	}
	var in struct {
		Scope   utils.JSONString `json:"scope"`
		AgentID json.RawMessage  `json:"agent_id"`
	}
	if err := json.Unmarshal(body, &in); err != nil {
		return utils.JSONString{}, "", apperr.ErrBadRequest.Wrap(err)
	}
	return in.Scope, string(in.AgentID), nil
}
