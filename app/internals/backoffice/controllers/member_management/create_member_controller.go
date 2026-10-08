package membermanagement

import (
	memberManagementDto "app/app/internals/backoffice/dto/member_management"
	"app/app/internals/backoffice/middleware"
	memberManagementService "app/app/service/member_management"
	"app/pkg/response"
	"app/pkg/utils"

	"github.com/gofiber/fiber/v2"
)

// CreateMemberController — POST /api/v1/bo/pr/manage/members/create
func CreateMemberController(c *fiber.Ctx) error {
	var req memberManagementDto.CreateMemberRequest
	if err := utils.ParseBodyNoNull(c, &req); err != nil {
		return response.Error(c, err)
	}
	res, err := memberManagementService.CreateMemberService(c.UserContext(), middleware.GetActor(c), req, middleware.RequestMeta(c))
	if err != nil {
		return response.Error(c, err)
	}
	return response.OK(c, res)
}
