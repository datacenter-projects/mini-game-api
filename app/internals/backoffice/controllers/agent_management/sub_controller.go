package agentmanagement

import (
	agentManagementDto "app/app/internals/backoffice/dto/agent_management"
	"app/app/internals/backoffice/middleware"
	agentManagementService "app/app/service/agent_management"
	"app/pkg/response"
	"app/pkg/utils"

	"github.com/gofiber/fiber/v2"
)

// เส้น sub (phase 5) — id ของ sub อยู่ใน body · ผู้เรียกมาจาก token

// ListSubaccountsController — POST /api/v1/bo/pr/manage/subaccounts (page / limit ใน body)
func ListSubaccountsController(c *fiber.Ctx) error {
	var req agentManagementDto.SubListRequest
	if err := utils.ParseBodyNoNull(c, &req); err != nil {
		return response.Error(c, err)
	}
	page := utils.NewPage(req.Page, req.Limit)
	rows, total, err := agentManagementService.ListSubaccountsService(c.UserContext(), middleware.GetActor(c), req, page)
	if err != nil {
		return response.Error(c, err)
	}
	return response.Page(c, rows, page, total)
}

// GetSubaccountController — POST /api/v1/bo/pr/manage/subaccounts/detail
func GetSubaccountController(c *fiber.Ctx) error {
	var req agentManagementDto.DetailRequest
	if err := utils.ParseBodyNoNull(c, &req); err != nil {
		return response.Error(c, err)
	}
	res, err := agentManagementService.GetSubaccountService(c.UserContext(), middleware.GetActor(c), req.ID)
	if err != nil {
		return response.Error(c, err)
	}
	return response.OK(c, res)
}

// CreateSubaccountController — POST /api/v1/bo/pr/manage/subaccounts/create
func CreateSubaccountController(c *fiber.Ctx) error {
	var req agentManagementDto.SubCreateRequest
	if err := utils.ParseBodyNoNull(c, &req); err != nil {
		return response.Error(c, err)
	}
	res, err := agentManagementService.CreateSubaccountService(c.UserContext(), middleware.GetActor(c), req, middleware.RequestMeta(c))
	if err != nil {
		return response.Error(c, err)
	}
	return response.OK(c, res)
}

// UpdateSubaccountController — POST /api/v1/bo/pr/manage/subaccounts/update-info
func UpdateSubaccountController(c *fiber.Ctx) error {
	var req agentManagementDto.SubUpdateRequest
	if err := utils.ParseBodyNoNull(c, &req); err != nil {
		return response.Error(c, err)
	}
	return result(c, agentManagementService.UpdateSubaccountService(c.UserContext(), middleware.GetActor(c), req, middleware.RequestMeta(c)))
}

// UpdateSubaccountStatusController — POST /api/v1/bo/pr/manage/subaccounts/update-status
func UpdateSubaccountStatusController(c *fiber.Ctx) error {
	var req agentManagementDto.SubStatusRequest
	if err := utils.ParseBodyNoNull(c, &req); err != nil {
		return response.Error(c, err)
	}
	return result(c, agentManagementService.UpdateSubaccountStatusService(c.UserContext(), middleware.GetActor(c), req, middleware.RequestMeta(c)))
}
