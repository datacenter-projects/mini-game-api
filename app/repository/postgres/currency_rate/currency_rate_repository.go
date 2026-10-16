// Package currencyrate คือ query ของตาราง currency_rates — ทุกฟังก์ชันรับ db เป็น param แรก ไม่เปิด transaction (กฎข้อ 5)
// rate_month ส่ง / รับเป็นเที่ยงคืน UTC ของวันที่ 1 (core.MonthDate) เพื่อไม่ให้ driver เลื่อนวัน
package currencyrate

import (
	"errors"
	"time"

	"app/app/models"
	"app/pkg/apperr"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// GetCurrencyRateRepository — เรท 1 แถวด้วย primary key (CR-26) · ไม่พบ = apperr.ErrNotFound
func GetCurrencyRateRepository(db *gorm.DB, month time.Time, currency string) (models.CurrencyRate, error) {
	var r models.CurrencyRate
	err := db.Where("rate_month = ? AND currency = ?", month, currency).Take(&r).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return r, apperr.ErrNotFound
	}
	return r, err
}

// ListCurrenciesOfMonthRepository — สกุลที่มีเรทของเดือนนี้แล้ว (เรียงตามชื่อ) ใช้ตัดสินว่า "ครบ 27 สกุลแล้ว" (CR-14)
func ListCurrenciesOfMonthRepository(db *gorm.DB, month time.Time) ([]string, error) {
	var out []string
	err := db.Model(&models.CurrencyRate{}).Where("rate_month = ?", month).Order("currency").Pluck("currency", &out).Error
	return out, err
}

// InsertCurrencyRatesIfAbsentRepository — INSERT … ON CONFLICT DO NOTHING (CR-05): แถวที่มีอยู่แล้วไม่ถูกแตะ (รวม synced_at)
// คืนจำนวนแถวที่เพิ่มจริง
func InsertCurrencyRatesIfAbsentRepository(db *gorm.DB, rows []models.CurrencyRate) (int64, error) {
	if len(rows) == 0 {
		return 0, nil
	}
	res := db.Clauses(clause.OnConflict{DoNothing: true}).Create(&rows)
	return res.RowsAffected, res.Error
}

// ReplaceCurrencyRatesRepository — เขียนทับทั้งชุด (INSERT … ON CONFLICT DO UPDATE) ใช้กับคำสั่งมือ -force เท่านั้น (CR-17)
func ReplaceCurrencyRatesRepository(db *gorm.DB, rows []models.CurrencyRate) error {
	if len(rows) == 0 {
		return nil
	}
	return db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "rate_month"}, {Name: "currency"}},
		DoUpdates: clause.AssignmentColumns([]string{"usdt_rate", "thb_rate", "synced_at"}),
	}).Create(&rows).Error
}
