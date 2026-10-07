package agentmanagement

import (
	agentManagementDto "app/app/internals/backoffice/dto/agent_management"
	"app/app/internals/backoffice/middleware"
	agentManagementService "app/app/service/agent_management"
	"app/pkg/response"
	"app/pkg/utils"

	"github.com/gofiber/fiber/v2"
)

// เส้นแก้บัญชี (phase 4) — id ของบัญชีที่จะแก้อยู่ใน body · ผู้แก้มาจาก token

// UpdateAgentInfoController — POST /api/v1/bo/pr/manage/agents/update-info
func UpdateAgentInfoController(c *fiber.Ctx) error {
	var req agentManagementDto.UpdateInfoRequest
	if err := utils.ParseBodyNoNull(c, &req); err != nil {
		return response.Error(c, err)
	}
	return result(c, agentManagementService.UpdateAgentInfoService(c.UserContext(), middleware.GetActor(c), req, middleware.RequestMeta(c)))
}

// UpdateMemberInfoController — POST /api/v1/bo/pr/manage/members/update-info
func UpdateMemberInfoController(c *fiber.Ctx) error {
	var req agentManagementDto.UpdateInfoRequest
	if err := utils.ParseBodyNoNull(c, &req); err != nil {
		return response.Error(c, err)
	}
	return result(c, agentManagementService.UpdateMemberInfoService(c.UserContext(), middleware.GetActor(c), req, middleware.RequestMeta(c)))
}

// UpdateAgentStatusController — POST /api/v1/bo/pr/manage/agents/update-status
func UpdateAgentStatusController(c *fiber.Ctx) error {
	var req agentManagementDto.UpdateStatusRequest
	if err := utils.ParseBodyNoNull(c, &req); err != nil {
		return response.Error(c, err)
	}
	return result(c, agentManagementService.UpdateAgentStatusService(c.UserContext(), middleware.GetActor(c), req, middleware.RequestMeta(c)))
}

// UpdateMemberStatusController — POST /api/v1/bo/pr/manage/members/update-status
func UpdateMemberStatusController(c *fiber.Ctx) error {
	var req agentManagementDto.UpdateStatusRequest
	if err := utils.ParseBodyNoNull(c, &req); err != nil {
		return response.Error(c, err)
	}
	return result(c, agentManagementService.UpdateMemberStatusService(c.UserContext(), middleware.GetActor(c), req, middleware.RequestMeta(c)))
}

// UpdateChildPTController — POST /api/v1/bo/pr/manage/agents/update-pt
func UpdateChildPTController(c *fiber.Ctx) error {
	var req agentManagementDto.UpdateChildPTRequest
	if err := utils.ParseBodyNoNull(c, &req); err != nil {
		return response.Error(c, err)
	}
	return result(c, agentManagementService.UpdateChildPTService(c.UserContext(), middleware.GetActor(c), req, middleware.RequestMeta(c)))
}

// UpdateMemberCommissionController — POST /api/v1/bo/pr/manage/members/update-commission
func UpdateMemberCommissionController(c *fiber.Ctx) error {
	var req agentManagementDto.UpdateMemberPTRequest
	if err := utils.ParseBodyNoNull(c, &req); err != nil {
		return response.Error(c, err)
	}
	return result(c, agentManagementService.UpdateMemberCommissionService(c.UserContext(), middleware.GetActor(c), req, middleware.RequestMeta(c)))
}

// UpdateOwnHoldController — POST /api/v1/bo/pr/manage/agents/update-hold
func UpdateOwnHoldController(c *fiber.Ctx) error {
	var req agentManagementDto.UpdateOwnPTRequest
	if err := utils.ParseBodyNoNull(c, &req); err != nil {
		return response.Error(c, err)
	}
	return result(c, agentManagementService.UpdateOwnHoldService(c.UserContext(), middleware.GetActor(c), req, middleware.RequestMeta(c)))
}

// result — เส้นแก้ไม่มี data
func result(c *fiber.Ctx, err error) error {
	if err != nil {
		return response.Error(c, err)
	}
	return response.OK(c, nil)
}
