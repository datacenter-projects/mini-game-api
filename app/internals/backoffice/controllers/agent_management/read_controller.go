package agentmanagement

import (
	agentManagementDto "app/app/internals/backoffice/dto/agent_management"
	"app/app/internals/backoffice/middleware"
	agentManagementService "app/app/service/agent_management"
	"app/pkg/response"
	"app/pkg/utils"

	"github.com/gofiber/fiber/v2"
)

// เส้นอ่านที่ต้องระบุบัญชีใช้ POST + id ใน body (ผู้ดูมาจาก token) — copy-sources ไม่ระบุบัญชีจึงเป็น GET

// ListDownlinesController — POST /api/v1/bo/pr/manage/downlines/list (page / limit ใน body)
func ListDownlinesController(c *fiber.Ctx) error {
	var req agentManagementDto.DownlinesRequest
	if err := utils.ParseBodyNoNull(c, &req); err != nil {
		return response.Error(c, err)
	}
	page := utils.NewPage(req.Page, req.Limit)
	rows, total, err := agentManagementService.ListDownlinesService(c.UserContext(), middleware.GetActor(c), req, page)
	if err != nil {
		return response.Error(c, err)
	}
	return response.Page(c, rows, page, total)
}

// GetAgentDetailController — POST /api/v1/bo/pr/manage/agents/detail
func GetAgentDetailController(c *fiber.Ctx) error {
	var req agentManagementDto.DetailRequest
	if err := utils.ParseBodyNoNull(c, &req); err != nil {
		return response.Error(c, err)
	}
	res, err := agentManagementService.GetAgentDetailService(c.UserContext(), middleware.GetActor(c), req.ID)
	if err != nil {
		return response.Error(c, err)
	}
	return response.OK(c, res)
}

// GetMemberDetailController — POST /api/v1/bo/pr/manage/members/detail
func GetMemberDetailController(c *fiber.Ctx) error {
	var req agentManagementDto.DetailRequest
	if err := utils.ParseBodyNoNull(c, &req); err != nil {
		return response.Error(c, err)
	}
	res, err := agentManagementService.GetMemberDetailService(c.UserContext(), middleware.GetActor(c), req.ID)
	if err != nil {
		return response.Error(c, err)
	}
	return response.OK(c, res)
}

// ListCopySourcesController — GET /api/v1/bo/pr/manage/agents/copy-sources
func ListCopySourcesController(c *fiber.Ctx) error {
	res, err := agentManagementService.ListCopySourcesService(c.UserContext(), middleware.GetActor(c))
	if err != nil {
		return response.Error(c, err)
	}
	return response.OK(c, res)
}
