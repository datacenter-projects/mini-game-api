package models

import "time"

// CurrencyRate — เรทรายเดือน 1 สกุล (docs/modules/currency_rate.md หัวข้อ 6.1)
// RateMonth = วันที่ 1 ของเดือน เก็บเป็นวันที่ล้วน (DATE) — ในโค้ดใช้ time.Time ที่เที่ยงคืน UTC ของวันนั้นเพื่อไม่ให้ขอบเขตเวลาเลื่อนวัน (core.MonthDate)
type CurrencyRate struct {
	RateMonth time.Time `gorm:"column:rate_month;primaryKey"`
	Currency  string    `gorm:"column:currency;primaryKey"`
	USDTRate  float64   `gorm:"column:usdt_rate"` // 1 USDT = ? หน่วยของสกุลนี้ (ค่าดิบ ทศนิยม ≤ 8)
	THBRate   float64   `gorm:"column:thb_rate"`  // 1 หน่วยของสกุลนี้ = ? บาท (ปัด 8 ตำแหน่ง)
	SyncedAt  time.Time `gorm:"column:synced_at"`
}

func (CurrencyRate) TableName() string { return "currency_rates" }

// CronjobStatus — ผลของรอบล่าสุดใน cronjobs.last_status (” = ยังไม่เคยรัน)
type CronjobStatus string

const (
	CronjobStatusSuccess CronjobStatus = "SUCCESS"
	CronjobStatusFailed  CronjobStatus = "FAILED"
	CronjobStatusSkipped CronjobStatus = "SKIPPED"
)

// Cronjob — สถานะของ job ตามเวลา 1 ตัว (docs/modules/currency_rate.md หัวข้อ 6.2)
type Cronjob struct {
	Code           string        `gorm:"column:code;primaryKey"`
	Enabled        bool          `gorm:"column:enabled"`
	LastStartedAt  *time.Time    `gorm:"column:last_started_at"`
	LastFinishedAt *time.Time    `gorm:"column:last_finished_at"`
	LastStatus     CronjobStatus `gorm:"column:last_status"`
	LastError      string        `gorm:"column:last_error"`
	UpdatedBy      string        `gorm:"column:updated_by"`
	UpdatedAt      time.Time     `gorm:"column:updated_at"`
	CreatedAt      time.Time     `gorm:"column:created_at"`
}

func (Cronjob) TableName() string { return "cronjobs" }
