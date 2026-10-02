// Package response คือ envelope มาตรฐานของทุก endpoint: {code, msg, data}
//
// controller ใช้แค่ 3 ฟังก์ชัน:
//
//	return response.OK(c, data)
//	return response.Page(c, items, page)   // list ที่มี pagination
//	return response.Error(c, err)          // err จาก service (apperr หรือ error อะไรก็ได้)
package response

import (
	"errors"
	"strings"

	"app/pkg/apperr"
	"app/platform/logger"

	"github.com/gofiber/fiber/v2"
)

type Envelope struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data any    `json:"data,omitempty"` // ไม่มีข้อมูล (nil) = ไม่ส่ง key data
}

// ข้อความตอบกลับเลือกภาษาจาก header X-Lang ก่อน แล้วค่อย Accept-Language — default ไทย
func lang(c *fiber.Ctx) string {
	for _, h := range []string{c.Get("X-Lang"), c.Get(fiber.HeaderAcceptLanguage)} {
		if strings.HasPrefix(strings.ToLower(strings.TrimSpace(h)), "en") {
			return "en"
		}
		if h != "" {
			return "th"
		}
	}
	return "th"
}

func message(c *fiber.Ctx, e *apperr.Error) string {
	if lang(c) == "en" {
		return e.MsgEN
	}
	return e.MsgTH
}

func OK(c *fiber.Ctx, data any) error {
	msg := "สำเร็จ"
	if lang(c) == "en" {
		msg = "Success"
	}
	return c.JSON(Envelope{Code: apperr.CodeSuccess, Msg: msg, Data: data})
}

// Error แปลง err เป็น envelope
//   - apperr.Error → ตอบ code/msg ของมัน (สาเหตุที่ Wrap ไว้จะถูก log ไม่ส่งถึง client)
//   - error อื่นทั้งหมด → log เต็ม แล้วตอบ ErrInternal (ไม่หลุดรายละเอียดภายในไปถึง client)
func Error(c *fiber.Ctx, err error) error {
	e := apperr.From(err)
	if cause := errors.Unwrap(e); cause != nil {
		log := logger.Ctx(c.UserContext()).With("code", e.Code, "method", c.Method(), "path", c.Path(), "error", cause)
		if e.HTTPStatus >= fiber.StatusInternalServerError {
			log.Error("request failed")
		} else {
			log.Warn("request rejected")
		}
	}
	return c.Status(e.HTTPStatus).JSON(Envelope{Code: e.Code, Msg: message(c, e)})
}

// FiberErrorHandler รับ error ที่ไม่ได้ผ่าน response.Error (เช่น 404/405 จาก router, body ใหญ่เกิน)
func FiberErrorHandler(c *fiber.Ctx, err error) error {
	var fe *fiber.Error
	if errors.As(err, &fe) {
		switch fe.Code {
		case fiber.StatusNotFound:
			return Error(c, apperr.ErrRouteNotFound)
		case fiber.StatusRequestEntityTooLarge, fiber.StatusBadRequest, fiber.StatusMethodNotAllowed:
			return Error(c, apperr.ErrBadRequest)
		}
	}
	return Error(c, err)
}
