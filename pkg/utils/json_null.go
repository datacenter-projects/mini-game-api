package utils

import (
	"sort"
	"strconv"

	"app/pkg/apperr"

	"github.com/goccy/go-json"
	"github.com/gofiber/fiber/v2"
)

// FindJSONNull — หา null ตัวแรกใน JSON (เรียง key A→Z ให้ผลคงที่) คืน path เช่น "pt.game.force" หรือ "currencies[1]"
// JSON ผิดรูปแบบ = ไม่พบ (ให้ BodyParser ตอบ 400 ตามปกติ)
func FindJSONNull(body []byte) (string, bool) {
	var v any
	if err := json.Unmarshal(body, &v); err != nil {
		return "", false
	}
	return findNull(v, "")
}

func findNull(v any, path string) (string, bool) {
	switch x := v.(type) {
	case nil:
		return path, true
	case map[string]any:
		keys := make([]string, 0, len(x))
		for k := range x {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			p := k
			if path != "" {
				p = path + "." + k
			}
			if found, ok := findNull(x[k], p); ok {
				return found, true
			}
		}
	case []any:
		for i, e := range x {
			if found, ok := findNull(e, path+"["+strconv.Itoa(i)+"]"); ok {
				return found, true
			}
		}
	}
	return "", false
}

// CheckNoNull — 422 เมื่อมี null ที่ใดใน body (account ACC-32: ไม่มี null ใน API) · body ว่าง / ไม่มี null = nil
// ใช้ร่วมกันทุกที่ที่อ่าน body เอง (ParseBodyNoNull, middleware ที่อ่าน field ก่อน controller) ให้ข้อความเหมือนกัน
func CheckNoNull(body []byte) error {
	if len(body) == 0 {
		return nil
	}
	path, ok := FindJSONNull(body)
	if !ok {
		return nil
	}
	if path == "" {
		path = "body"
	}
	th, en := FieldLabel(path)
	return apperr.ErrValidation.WithMessage(th+" ห้ามเป็น null", en+" must not be null")
}

// ParseBodyNoNull — เหมือน ParseBody แต่ตอบ 422 เมื่อมี null ที่ใดใน body (account ACC-32: ไม่มี null ใน API)
func ParseBodyNoNull(c *fiber.Ctx, out Validatable) error {
	if err := CheckNoNull(c.Body()); err != nil {
		return err
	}
	return ParseBody(c, out)
}
