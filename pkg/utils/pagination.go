package utils

import (
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

const (
	DefaultPageLimit = 20
	MaxPageLimit     = 100
)

type Page struct {
	Page  int
	Limit int
}

// ParsePage อ่าน ?page=&limit= — ค่าผิด/เกินขอบเขตถูกปรับให้อยู่ในช่วงที่รับได้ (ไม่ error)
func ParsePage(c *fiber.Ctx) Page {
	p := Page{Page: c.QueryInt("page", 1), Limit: c.QueryInt("limit", DefaultPageLimit)}
	if p.Page < 1 {
		p.Page = 1
	}
	if p.Limit < 1 {
		p.Limit = DefaultPageLimit
	}
	if p.Limit > MaxPageLimit {
		p.Limit = MaxPageLimit
	}
	return p
}

func (p Page) Offset() int { return (p.Page - 1) * p.Limit }

// Scope ใช้ใน repository: db.Scopes(p.Scope()).Find(&rows)
func (p Page) Scope() func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB { return db.Offset(p.Offset()).Limit(p.Limit) }
}
