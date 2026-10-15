package membermanagement

import (
	memberManagementDto "app/app/internals/backoffice/dto/member_management"
	"app/app/internals/backoffice/middleware"
	memberManagementService "app/app/service/member_management"
	"app/pkg/response"
	"app/pkg/utils"

	"github.com/gofiber/fiber/v2"
)

// GetMemberDetailController — POST /api/v1/bo/pr/manage/members/detail/get (M2)
func GetMemberDetailController(c *fiber.Ctx) error {
	var req memberManagementDto.DetailRequest
	if err := utils.ParseBodyNoNull(c, &req); err != nil {
		return response.Error(c, err)
	}
	res, err := memberManagementService.GetMemberDetailService(c.UserContext(), middleware.GetActor(c), req.ID)
	if err != nil {
		return response.Error(c, err)
	}
	return response.OK(c, res)
}

// ListMembersController — POST /api/v1/bo/pr/manage/members/list (M5)
func ListMembersController(c *fiber.Ctx) error {
	var req memberManagementDto.MemberListRequest
	if err := utils.ParseBodyNoNull(c, &req); err != nil {
		return response.Error(c, err)
	}
	res, err := memberManagementService.ListMembersService(c.UserContext(), middleware.GetActor(c), req)
	if err != nil {
		return response.Error(c, err)
	}
	return response.OK(c, res)
}
