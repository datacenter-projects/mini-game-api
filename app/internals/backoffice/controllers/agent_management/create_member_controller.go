package agentmanagement

import (
	agentManagementDto "app/app/internals/backoffice/dto/agent_management"
	"app/app/internals/backoffice/middleware"
	agentManagementService "app/app/service/agent_management"
	"app/pkg/response"
	"app/pkg/utils"

	"github.com/gofiber/fiber/v2"
)

// CreateMemberController — POST /api/v1/bo/pr/manage/members
func CreateMemberController(c *fiber.Ctx) error {
	var req agentManagementDto.CreateMemberRequest
	if err := utils.ParseBodyNoNull(c, &req); err != nil {
		return response.Error(c, err)
	}
	res, err := agentManagementService.CreateMemberService(c.UserContext(), middleware.GetActor(c), req, middleware.RequestMeta(c))
	if err != nil {
		return response.Error(c, err)
	}
	return response.OK(c, res)
}
