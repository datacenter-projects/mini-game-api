package models

import "time"

// ตารางของ module asset — docs/modules/asset.md หัวข้อ 6

// AssetScope — เจ้าของค่า: ค่ากลาง หรือค่าที่ตั้งไว้ที่บัญชีใดบัญชีหนึ่ง (AST-01)
type AssetScope string

const (
	AssetScopeGlobal AssetScope = "GLOBAL"
	AssetScopeAgent  AssetScope = "AGENT"
)

// AssetType — catalog ของช่อง (AST-11) · เพิ่ม / แก้ / เลิกใช้ด้วย migration เท่านั้น
type AssetType struct {
	ID           uint      `gorm:"column:id;primaryKey"`
	GameCode     string    `gorm:"column:game_code"`
	AssetType    string    `gorm:"column:asset_type"`
	AllowedSizes string    `gorm:"column:allowed_sizes"` // JSONB [{"width","height"}] · [] = ไม่ตรวจขนาด (AST-12)
	IsRetired    bool      `gorm:"column:is_retired"`
	CreatedAt    time.Time `gorm:"column:created_at"`
	UpdatedAt    time.Time `gorm:"column:updated_at"`
}

func (AssetType) TableName() string { return "asset_types" }

// AssetConfig — ไฟล์ 1 เวอร์ชันที่เคยอัป = คลังรูป (AST-20)
type AssetConfig struct {
	ID          uint       `gorm:"column:id;primaryKey"`
	Scope       AssetScope `gorm:"column:scope"`
	AgentID     *uint      `gorm:"column:agent_id"` // nil = GLOBAL
	AssetTypeID uint       `gorm:"column:asset_type_id"`
	StorageKey  string     `gorm:"column:storage_key"`
	CreatedBy   string     `gorm:"column:created_by"`
	IsDeleted   bool       `gorm:"column:is_deleted"`
	DeletedAt   *time.Time `gorm:"column:deleted_at"`
	DeletedBy   *string    `gorm:"column:deleted_by"`
	CreatedAt   time.Time  `gorm:"column:created_at"`
}

func (AssetConfig) TableName() string { return "asset_configs" }

// AssetUsed — config ที่เลือกใช้ของช่อง · 1 แถวต่อ (เป้าหมาย, ช่อง) (AST-22)
type AssetUsed struct {
	ID            uint       `gorm:"column:id;primaryKey"`
	Scope         AssetScope `gorm:"column:scope"`
	AgentID       *uint      `gorm:"column:agent_id"` // nil = GLOBAL
	AssetTypeID   uint       `gorm:"column:asset_type_id"`
	AssetConfigID uint       `gorm:"column:asset_config_id"`
	CreatedBy     string     `gorm:"column:created_by"`
	UpdatedBy     string     `gorm:"column:updated_by"`
	CreatedAt     time.Time  `gorm:"column:created_at"`
	UpdatedAt     time.Time  `gorm:"column:updated_at"`
}

func (AssetUsed) TableName() string { return "asset_used" }

// GameTheme — ธีมเกม 5 ค่า · GLOBAL = ค่ากลาง · AGENT = override ของบัญชี (AST-37)
type GameTheme struct {
	ID                 uint       `gorm:"column:id;primaryKey"`
	Scope              AssetScope `gorm:"column:scope"`
	AgentID            *uint      `gorm:"column:agent_id"` // nil = GLOBAL
	GameCode           string     `gorm:"column:game_code"`
	FontFamily         string     `gorm:"column:font_family"`
	MainColor1         string     `gorm:"column:main_color_1"`
	MainColor2         string     `gorm:"column:main_color_2"`
	StatsBarColor      string     `gorm:"column:stats_bar_color"`
	StatsBarFontFamily string     `gorm:"column:stats_bar_font_family"`
	CreatedBy          string     `gorm:"column:created_by"`
	UpdatedBy          string     `gorm:"column:updated_by"`
	CreatedAt          time.Time  `gorm:"column:created_at"`
	UpdatedAt          time.Time  `gorm:"column:updated_at"`
}

func (GameTheme) TableName() string { return "game_themes" }

// SiteBranding — ชื่อเว็บ + โลโก้ของบัญชี 1 แถวต่อบัญชี · ไม่มี GLOBAL (AST-41)
type SiteBranding struct {
	AgentID   uint      `gorm:"column:agent_id;primaryKey"`
	SiteName  string    `gorm:"column:site_name"`
	LogoKey   string    `gorm:"column:logo_key"`
	CreatedBy string    `gorm:"column:created_by"`
	UpdatedBy string    `gorm:"column:updated_by"`
	CreatedAt time.Time `gorm:"column:created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at"`
}

func (SiteBranding) TableName() string { return "site_brandings" }
