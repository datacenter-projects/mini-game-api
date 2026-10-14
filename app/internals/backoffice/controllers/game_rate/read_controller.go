// Package gamerate คือ controller ของเส้น /rate/* (อัตราแพ้ชนะ · เมนู ⑥) — spec: docs/modules/game_rate.md หัวข้อ 5
// สิทธิ์ rate ตรวจที่ route (RequirePermission) · บันทึก / ย้อนต้อง passcode (RequirePasscode หลังสิทธิ์ — RATE-02)
package gamerate

import (
	gameRateDto "app/app/internals/backoffice/dto/game_rate"
	gameRateService "app/app/service/game_rate"
	"app/pkg/response"
	"app/pkg/utils"

	"github.com/gofiber/fiber/v2"
)

// ListRatesController — RE1 POST /api/v1/bo/pr/rate/list
func ListRatesController(c *fiber.Ctx) error {
	res, err := gameRateService.ListRatesService(c.UserContext())
	if err != nil {
		return response.Error(c, err)
	}
	return response.OK(c, res)
}

// GetRateController — RE2 POST /api/v1/bo/pr/rate/detail/get
func GetRateController(c *fiber.Ctx) error {
	var req gameRateDto.GameRequest
	if err := utils.ParseBodyNoNull(c, &req); err != nil {
		return response.Error(c, err)
	}
	res, err := gameRateService.GetRateService(c.UserContext(), req.GameCode)
	if err != nil {
		return response.Error(c, err)
	}
	return response.OK(c, res)
}

// CheckRateController — RE3 POST /api/v1/bo/pr/rate/detail/check (ไม่บันทึก)
func CheckRateController(c *fiber.Ctx) error {
	var req gameRateDto.CheckRequest
	if err := utils.ParseBodyNoNull(c, &req); err != nil {
		return response.Error(c, err)
	}
	return response.OK(c, gameRateService.CheckRateService(req))
}

// ListVersionsController — RE5 POST /api/v1/bo/pr/rate/versions/list
func ListVersionsController(c *fiber.Ctx) error {
	var req gameRateDto.VersionsListRequest
	if err := utils.ParseBodyNoNull(c, &req); err != nil {
		return response.Error(c, err)
	}
	page := utils.NewPage(req.Page, req.Limit)
	rows, total, err := gameRateService.ListVersionsService(c.UserContext(), req.GameCode, page)
	if err != nil {
		return response.Error(c, err)
	}
	return response.Page(c, rows, page, total)
}

// GetVersionController — RE6 POST /api/v1/bo/pr/rate/versions/detail/get
func GetVersionController(c *fiber.Ctx) error {
	var req gameRateDto.VersionRequest
	if err := utils.ParseBodyNoNull(c, &req); err != nil {
		return response.Error(c, err)
	}
	res, err := gameRateService.GetVersionService(c.UserContext(), req.GameCode, req.Version)
	if err != nil {
		return response.Error(c, err)
	}
	return response.OK(c, res)
}
