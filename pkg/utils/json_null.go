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

// ParseBodyNoNull — เหมือน ParseBody แต่ตอบ 422 เมื่อมี null ที่ใดใน body (account ACC-32: ไม่มี null ใน API)
func ParseBodyNoNull(c *fiber.Ctx, out Validatable) error {
	if body := c.Body(); len(body) > 0 {
		if path, ok := FindJSONNull(body); ok {
			if path == "" {
				path = "body"
			}
			return apperr.ErrValidation.WithMessage(path+" ห้ามเป็น null", path+" must not be null")
		}
	}
	return ParseBody(c, out)
}
