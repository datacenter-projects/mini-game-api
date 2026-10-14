package utils

import (
	"io"

	"app/pkg/apperr"

	"github.com/gofiber/fiber/v2"
)

// ReadFormFile อ่านไฟล์จาก field ของ multipart/form-data เข้าหน่วยความจำ — อ่านไม่เกิน limit ไบต์
// (ไม่ว่า client ส่งมาเท่าไร จึงไม่กินหน่วยความจำเกินเพดาน)
//
// ผู้เรียกส่ง limit = เพดานขนาดไฟล์ + 1 แล้วเทียบ len(data) กับเพดานเอง — ทำให้ตัดสินเรื่องขนาดตามลำดับตรวจของ module ได้
// ไม่แนบไฟล์ = 422 บอก field · field text อื่นของฟอร์มอ่านด้วย ParseBody ตามปกติ (BodyParser รองรับ multipart: tag `form`)
//
//	data, err := utils.ReadFormFile(c, "file", assetCore.MaxFileSize+1)
func ReadFormFile(c *fiber.Ctx, field string, limit int64) ([]byte, error) {
	fh, err := c.FormFile(field)
	if err != nil {
		return nil, apperr.ErrValidation.WithMessage("ต้องแนบไฟล์ใน field "+field, "a file is required in field "+field)
	}
	f, err := fh.Open()
	if err != nil {
		return nil, apperr.ErrBadRequest.Wrap(err)
	}
	defer f.Close() //nolint:errcheck
	data, err := io.ReadAll(io.LimitReader(f, limit))
	if err != nil {
		return nil, apperr.ErrBadRequest.Wrap(err)
	}
	return data, nil
}
