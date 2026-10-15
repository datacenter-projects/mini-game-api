package agentmanagement

import (
	agentManagementDto "app/app/internals/backoffice/dto/agent_management"
	"app/app/internals/backoffice/middleware"
	agentManagementService "app/app/service/agent_management"
	"app/pkg/response"
	"app/pkg/utils"

	"github.com/gofiber/fiber/v2"
)

// เส้นแก้บัญชี — id ของบัญชีที่จะแก้อยู่ใน body · ผู้แก้มาจาก token (spec หัวข้อ 5 · lead E4 / E5)

// UpdateAgentDetailController — POST /api/v1/bo/pr/manage/agents/detail/update
func UpdateAgentDetailController(c *fiber.Ctx) error {
	var req agentManagementDto.AgentDetailUpdateRequest
	if err := utils.ParseBodyNoNull(c, &req); err != nil {
		return response.Error(c, err)
	}
	return result(c, agentManagementService.UpdateAgentDetailService(c.UserContext(), middleware.GetActor(c), req, middleware.RequestMeta(c)))
}

// UpdateAgentStatusController — POST /api/v1/bo/pr/manage/agents/status/update
func UpdateAgentStatusController(c *fiber.Ctx) error {
	var req agentManagementDto.UpdateStatusRequest
	if err := utils.ParseBodyNoNull(c, &req); err != nil {
		return response.Error(c, err)
	}
	return result(c, agentManagementService.UpdateAgentStatusService(c.UserContext(), middleware.GetActor(c), req, middleware.RequestMeta(c)))
}

// result — เส้นแก้ไม่มี data
func result(c *fiber.Ctx, err error) error {
	if err != nil {
		return response.Error(c, err)
	}
	return response.OK(c, nil)
}
