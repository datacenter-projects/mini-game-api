package utils

import (
	"app/pkg/apperr"

	"github.com/gofiber/fiber/v2"
)

// Validatable คือ DTO ที่ตรวจความถูกต้องของตัวเองได้
//
// เขียนเงื่อนไขเป็น if ธรรมดาใน method Validate() ของ DTO แล้วคืน
// apperr.ErrValidation.WithMessage(th, en) ที่บอกว่าผิดที่ field ไหน — ไม่ใช้ struct tag
type Validatable interface {
	Validate() error
}

// ParseBody = BodyParser + Validate() ในขั้นเดียว — controller ใช้แบบนี้:
//
//	var req dto.CreateAgentRequest
//	if err := utils.ParseBody(c, &req); err != nil {
//		return response.Error(c, err)
//	}
func ParseBody(c *fiber.Ctx, out Validatable) error {
	if err := c.BodyParser(out); err != nil {
		return apperr.ErrBadRequest.Wrap(err)
	}
	return out.Validate()
}

// ParseQuery เหมือน ParseBody แต่อ่านจาก query string (tag `query:"..."`)
func ParseQuery(c *fiber.Ctx, out Validatable) error {
	if err := c.QueryParser(out); err != nil {
		return apperr.ErrBadRequest.Wrap(err)
	}
	return out.Validate()
}
