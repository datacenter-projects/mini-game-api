package utils

import (
	"errors"

	"app/pkg/apperr"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
)

var validate = validator.New(validator.WithRequiredStructEnabled())

// ParseBody = BodyParser + validate tag ในขั้นเดียว — controller ใช้แบบนี้:
//
//	var req dto.CreateAgentRequest
//	if err := utils.ParseBody(c, &req); err != nil {
//		return response.Error(c, err)
//	}
func ParseBody(c *fiber.Ctx, out any) error {
	if err := c.BodyParser(out); err != nil {
		return apperr.ErrBadRequest.Wrap(err)
	}
	return Validate(out)
}

// ParseQuery เหมือน ParseBody แต่อ่านจาก query string (tag `query:"..."`)
func ParseQuery(c *fiber.Ctx, out any) error {
	if err := c.QueryParser(out); err != nil {
		return apperr.ErrBadRequest.Wrap(err)
	}
	return Validate(out)
}

func Validate(s any) error {
	if err := validate.Struct(s); err != nil {
		var ve validator.ValidationErrors
		if errors.As(err, &ve) {
			return apperr.ErrValidation.Wrap(err)
		}
		return apperr.ErrBadRequest.Wrap(err)
	}
	return nil
}
