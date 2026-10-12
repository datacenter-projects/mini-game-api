package asset

import (
	assetCore "app/app/core/asset"
	assetDto "app/app/internals/backoffice/dto/asset"
	"app/app/internals/backoffice/middleware"
	assetService "app/app/service/asset"
	"app/pkg/response"
	"app/pkg/utils"

	"github.com/gofiber/fiber/v2"
)

// UploadFileController — POST /api/v1/bo/pr/asset/files/upload (GN3) · multipart/form-data
// scope / agent_id อ่านโดย middleware ResolveAssetTarget · ไฟล์อ่านเกินเพดาน 1 ไบต์ แล้วให้ service ตัดสินขนาดตามลำดับ AST-15
func UploadFileController(c *fiber.Ctx) error {
	var req assetDto.UploadRequest
	if err := utils.ParseBody(c, &req); err != nil {
		return response.Error(c, err)
	}
	file, err := utils.ReadFormFile(c, "file", assetCore.MaxFileSize+1)
	if err != nil {
		return response.Error(c, err)
	}
	res, err := assetService.UploadFileService(c.UserContext(), middleware.GetActor(c), middleware.GetAssetTarget(c), req, file)
	if err != nil {
		return response.Error(c, err)
	}
	return response.OK(c, res)
}

// ListFilesController — POST /api/v1/bo/pr/asset/files/list (GN4) · page / limit ใน body (กฎข้อ 19)
func ListFilesController(c *fiber.Ctx) error {
	var req assetDto.FilesListRequest
	if err := utils.ParseBodyNoNull(c, &req); err != nil {
		return response.Error(c, err)
	}
	page := utils.NewPage(req.Page, req.Limit)
	rows, total, err := assetService.ListFilesService(c.UserContext(), middleware.GetActor(c), middleware.GetAssetTarget(c), req, page)
	if err != nil {
		return response.Error(c, err)
	}
	return response.Page(c, rows, page, total)
}

// DeleteFileController — POST /api/v1/bo/pr/asset/files/delete (GN5)
func DeleteFileController(c *fiber.Ctx) error {
	var req assetDto.FilesDeleteRequest
	if err := utils.ParseBodyNoNull(c, &req); err != nil {
		return response.Error(c, err)
	}
	if err := assetService.DeleteFileService(c.UserContext(), middleware.GetActor(c), middleware.GetAssetTarget(c), req); err != nil {
		return response.Error(c, err)
	}
	return response.OK(c, nil)
}
