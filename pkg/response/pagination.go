package response

import (
	"app/pkg/utils"

	"github.com/gofiber/fiber/v2"
)

// PageData คือรูปแบบ list มาตรฐานเดียวของทั้งระบบ (แบบ type A ของ askmelotto-api)
type PageData[T any] struct {
	CurrentPage int   `json:"current_page"`
	TotalPage   int   `json:"total_page"`
	TotalCount  int64 `json:"total_count"`
	Limit       int   `json:"limit"`
	HasNext     bool  `json:"has_next"`
	HasPrev     bool  `json:"has_prev"`
	Data        []T   `json:"data"`
}

func NewPageData[T any](items []T, p utils.Page, total int64) PageData[T] {
	if items == nil {
		items = []T{} // ตอบ [] เสมอ ไม่ตอบ null
	}
	totalPage := int((total + int64(p.Limit) - 1) / int64(p.Limit))
	return PageData[T]{
		CurrentPage: p.Page,
		TotalPage:   totalPage,
		TotalCount:  total,
		Limit:       p.Limit,
		HasNext:     p.Page < totalPage,
		HasPrev:     p.Page > 1,
		Data:        items,
	}
}

func Page[T any](c *fiber.Ctx, items []T, p utils.Page, total int64) error {
	return OK(c, NewPageData(items, p, total))
}
