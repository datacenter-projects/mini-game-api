package agentmanagement

import (
	agentManagementDto "app/app/internals/backoffice/dto/agent_management"
	"app/app/internals/backoffice/middleware"
	agentManagementService "app/app/service/agent_management"
	"app/pkg/apperr"
	"app/pkg/response"
	"app/pkg/utils"

	"github.com/gofiber/fiber/v2"
)

// ListDownlinesController — GET /api/v1/bo/pr/manage/downlines
func ListDownlinesController(c *fiber.Ctx) error {
	var q agentManagementDto.DownlinesQuery
	if err := utils.ParseQuery(c, &q); err != nil {
		return response.Error(c, err)
	}
	page := utils.ParsePage(c)
	rows, total, err := agentManagementService.ListDownlinesService(c.UserContext(), middleware.GetActor(c), q, page)
	if err != nil {
		return response.Error(c, err)
	}
	return response.Page(c, rows, page, total)
}

// GetAgentDetailController — GET /api/v1/bo/pr/manage/agents/:id
func GetAgentDetailController(c *fiber.Ctx) error {
	id, err := paramID(c)
	if err != nil {
		return response.Error(c, err)
	}
	res, err := agentManagementService.GetAgentDetailService(c.UserContext(), middleware.GetActor(c), id)
	if err != nil {
		return response.Error(c, err)
	}
	return response.OK(c, res)
}

// GetMemberDetailController — GET /api/v1/bo/pr/manage/members/:id
func GetMemberDetailController(c *fiber.Ctx) error {
	id, err := paramID(c)
	if err != nil {
		return response.Error(c, err)
	}
	res, err := agentManagementService.GetMemberDetailService(c.UserContext(), middleware.GetActor(c), id)
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

// paramID — :id ต้องเป็นจำนวนเต็มบวก
func paramID(c *fiber.Ctx) (uint, error) {
	id, err := c.ParamsInt("id")
	if err != nil || id < 1 {
		return 0, apperr.ErrValidation.WithMessage("id ต้องเป็นจำนวนเต็มบวก", "id must be a positive integer")
	}
	return uint(id), nil
}
