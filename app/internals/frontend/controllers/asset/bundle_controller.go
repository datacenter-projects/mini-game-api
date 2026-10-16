// Package asset คือ controller ฝั่งผู้เล่นของ module asset — parse → เรียก service → response เท่านั้น
package asset

import (
	assetDto "app/app/internals/frontend/dto/asset"
	"app/app/internals/frontend/middleware"
	assetService "app/app/service/asset"
	"app/pkg/response"
	"app/pkg/utils"

	"github.com/gofiber/fiber/v2"
)

// BundleController — GN23 POST /api/v1/player/pr/asset/bundle · รูปทุกช่อง + ธีม + ชื่อเว็บ / โลโก้ของเว็บผู้เล่น
// เว็บ = ผู้สร้างตรงของ Member จาก session (AST-52) · ไม่รับ agent_id · ตัวสร้างเดียวกับ BO preview (AST-48)
func BundleController(c *fiber.Ctx) error {
	var req assetDto.BundleRequest
	if err := utils.ParseBodyNoNull(c, &req); err != nil {
		return response.Error(c, err)
	}
	res, err := assetService.BundleOfAgentService(c.UserContext(), middleware.GetPlayer(c).AgentID, req.GameCode)
	if err != nil {
		return response.Error(c, err)
	}
	return response.OK(c, res)
}
