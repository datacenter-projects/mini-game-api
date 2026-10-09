package membermanagement

import (
	memberManagementDto "app/app/internals/backoffice/dto/member_management"
	"app/app/internals/backoffice/middleware"
	memberManagementService "app/app/service/member_management"
	"app/pkg/response"
	"app/pkg/utils"

	"github.com/gofiber/fiber/v2"
)

// UpdateMemberInfoController — POST /api/v1/bo/pr/manage/members/update-info
func UpdateMemberInfoController(c *fiber.Ctx) error {
	var req memberManagementDto.UpdateInfoRequest
	if err := utils.ParseBodyNoNull(c, &req); err != nil {
		return response.Error(c, err)
	}
	return result(c, memberManagementService.UpdateMemberInfoService(c.UserContext(), middleware.GetActor(c), req, middleware.RequestMeta(c)))
}

// UpdateMemberStatusController — POST /api/v1/bo/pr/manage/members/update-status
func UpdateMemberStatusController(c *fiber.Ctx) error {
	var req memberManagementDto.UpdateStatusRequest
	if err := utils.ParseBodyNoNull(c, &req); err != nil {
		return response.Error(c, err)
	}
	return result(c, memberManagementService.UpdateMemberStatusService(c.UserContext(), middleware.GetActor(c), req, middleware.RequestMeta(c)))
}

// UpdateMemberPTController — POST /api/v1/bo/pr/manage/members/update-pt
func UpdateMemberPTController(c *fiber.Ctx) error {
	var req memberManagementDto.UpdateMemberPTRequest
	if err := utils.ParseBodyNoNull(c, &req); err != nil {
		return response.Error(c, err)
	}
	return result(c, memberManagementService.UpdateMemberPTService(c.UserContext(), middleware.GetActor(c), req, middleware.RequestMeta(c)))
}

// result — เส้นแก้ไม่มี data
func result(c *fiber.Ctx, err error) error {
	if err != nil {
		return response.Error(c, err)
	}
	return response.OK(c, nil)
}
