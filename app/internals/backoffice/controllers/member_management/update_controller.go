package membermanagement

import (
	memberManagementDto "app/app/internals/backoffice/dto/member_management"
	"app/app/internals/backoffice/middleware"
	memberManagementService "app/app/service/member_management"
	"app/pkg/response"
	"app/pkg/utils"

	"github.com/gofiber/fiber/v2"
)

// UpdateMemberDetailController — POST /api/v1/bo/pr/manage/members/detail/update (M3)
func UpdateMemberDetailController(c *fiber.Ctx) error {
	var req memberManagementDto.MemberDetailUpdateRequest
	if err := utils.ParseBodyNoNull(c, &req); err != nil {
		return response.Error(c, err)
	}
	return result(c, memberManagementService.UpdateMemberDetailService(c.UserContext(), middleware.GetActor(c), req, middleware.RequestMeta(c)))
}

// UpdateMemberStatusController — POST /api/v1/bo/pr/manage/members/status/update (M4)
func UpdateMemberStatusController(c *fiber.Ctx) error {
	var req memberManagementDto.UpdateStatusRequest
	if err := utils.ParseBodyNoNull(c, &req); err != nil {
		return response.Error(c, err)
	}
	return result(c, memberManagementService.UpdateMemberStatusService(c.UserContext(), middleware.GetActor(c), req, middleware.RequestMeta(c)))
}

// result — เส้นแก้ไม่มี data
func result(c *fiber.Ctx, err error) error {
	if err != nil {
		return response.Error(c, err)
	}
	return response.OK(c, nil)
}
