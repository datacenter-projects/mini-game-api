package agentmanagement

import (
	agentManagementDto "app/app/internals/backoffice/dto/agent_management"
	"app/app/internals/backoffice/middleware"
	agentManagementService "app/app/service/agent_management"
	"app/pkg/response"
	"app/pkg/utils"

	"github.com/gofiber/fiber/v2"
)

// เส้นอ่านใช้ POST · ระบุบัญชีด้วย id ใน body (ผู้ดูมาจาก token) — spec หัวข้อ 5 (lead E2 / E3 / E6)

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

// GetAgentDetailController — POST /api/v1/bo/pr/manage/agents/detail/get
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

// ListAgentsController — POST /api/v1/bo/pr/manage/agents/list
func ListAgentsController(c *fiber.Ctx) error {
	var req agentManagementDto.AgentListRequest
	if err := utils.ParseBodyNoNull(c, &req); err != nil {
		return response.Error(c, err)
	}
	res, err := agentManagementService.ListAgentsService(c.UserContext(), middleware.GetActor(c), req)
	if err != nil {
		return response.Error(c, err)
	}
	return response.OK(c, res)
}
