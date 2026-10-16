package models

import "time"

// ตารางของ game_rate (อัตราแพ้ชนะ · เมนู ⑥) — docs/modules/game_rate.md หัวข้อ 6
// ค่า % / ตัวคูณ float64 ปัด 4 ตำแหน่ง (กฎข้อ 9) · แถวเวอร์ชันห้าม UPDATE / DELETE (RATE-05 · RATE-08)

// GameRateVersion — 1 แถวต่อการบันทึกที่ค่าเปลี่ยน · เป็น log คนแก้ด้วย
type GameRateVersion struct {
	ID           uint        `gorm:"column:id;primaryKey"`
	GameCode     string      `gorm:"column:game_code"`
	Version      int         `gorm:"column:version"`
	WinPercent   *float64    `gorm:"column:win_percent"` // รูปแบบ prizes เท่านั้น · steps = nil
	ActorType    AccountType `gorm:"column:actor_type"`
	ActorID      uint        `gorm:"column:actor_id"`
	CreatedBy    string      `gorm:"column:created_by"` // username ณ เวลานั้น · sub = owner@name
	IP           string      `gorm:"column:ip"`
	RequestID    string      `gorm:"column:request_id"`
	RollbackFrom int         `gorm:"column:rollback_from"` // 0 = บันทึกปกติ (RATE-05A)
	CreatedAt    time.Time   `gorm:"column:created_at"`
}

func (GameRateVersion) TableName() string { return "game_rate_versions" }

// GameRateStep — 1 ขั้นของรูปแบบ steps (Coin Flip / RPS)
type GameRateStep struct {
	VersionID  uint    `gorm:"column:version_id;primaryKey"`
	Step       int     `gorm:"column:step;primaryKey"`
	Multiplier float64 `gorm:"column:multiplier"`
	WinPercent float64 `gorm:"column:win_percent"`
	Status     bool    `gorm:"column:status"` // ค่าที่หน้าบ้านเก็บไว้ (ไม่กระทบสูตร)
}

func (GameRateStep) TableName() string { return "game_rate_steps" }

// GameRatePrize — 1 แถวรางวัลของรูปแบบ prizes (Scratch Card)
type GameRatePrize struct {
	VersionID    uint    `gorm:"column:version_id;primaryKey"`
	RowNo        int     `gorm:"column:row_no;primaryKey"`
	Multiplier   float64 `gorm:"column:multiplier"`
	SharePercent float64 `gorm:"column:share_percent"`
	Status       bool    `gorm:"column:status"` // ค่าที่หน้าบ้านเก็บไว้ (ไม่กระทบสูตร)
}

func (GameRatePrize) TableName() string { return "game_rate_prizes" }
