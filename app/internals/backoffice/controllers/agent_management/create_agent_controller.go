package agentmanagement

import (
	agentManagementDto "app/app/internals/backoffice/dto/agent_management"
	"app/app/internals/backoffice/middleware"
	agentManagementService "app/app/service/agent_management"
	"app/pkg/response"
	"app/pkg/utils"

	"github.com/gofiber/fiber/v2"
)

// CreateAgentController — POST /api/v1/bo/pr/manage/agents
func CreateAgentController(c *fiber.Ctx) error {
	var req agentManagementDto.CreateAgentRequest
	if err := utils.ParseBodyNoNull(c, &req); err != nil {
		return response.Error(c, err)
	}
	res, err := agentManagementService.CreateAgentService(c.UserContext(), middleware.GetActor(c), req, middleware.RequestMeta(c))
	if err != nil {
		return response.Error(c, err)
	}
	return response.OK(c, res)
}
