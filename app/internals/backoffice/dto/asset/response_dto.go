package asset

// response ของ module asset — docs/modules/asset.md หัวข้อ 5 · ไม่มี null (ACC-32): ไม่มีค่า = "" · 0 · [] · {}

// GameItem — `games/list` (GN1)
type GameItem struct {
	GameCode  string `json:"game_code"`
	GameCover string `json:"game_cover"` // "" = ยังไม่มีปก
}

// SizeItem — ขนาดภาพที่ช่องรับ
type SizeItem struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

// SlotItem — `slots/list` (GN2) · ไม่มี description (lead GE2-1)
type SlotItem struct {
	AssetType    string     `json:"asset_type"`
	AllowedSizes []SizeItem `json:"allowed_sizes"` // [] = ไม่ตรวจขนาด
}

// UploadResponse — `files/upload` (GN3)
type UploadResponse struct {
	ConfigID  uint   `json:"config_id"`
	Scope     string `json:"scope"`
	AgentID   uint   `json:"agent_id"` // 0 = GLOBAL
	GameCode  string `json:"game_code"`
	AssetType string `json:"asset_type"`
	URL       string `json:"url"`
	IsUsed    bool   `json:"is_used"`
	CreatedBy string `json:"created_by"`
	CreatedAt string `json:"created_at"`
}

// FileItem — แถวของ `files/list` (GN4)
type FileItem struct {
	ConfigID  uint   `json:"config_id"`
	URL       string `json:"url"`
	Source    string `json:"source"` // self / global
	IsUsed    bool   `json:"is_used"`
	CreatedBy string `json:"created_by"`
	CreatedAt string `json:"created_at"`
}

// UsedItem — 1 ช่องของ `used/list` (GN6) · ช่องที่ไม่มีไฟล์: config_id = 0 · url = "" · source = ""
type UsedItem struct {
	AssetType     string `json:"asset_type"`
	ConfigID      uint   `json:"config_id"`
	URL           string `json:"url"`
	Source        string `json:"source"`
	InheritedFrom string `json:"inherited_from"` // username ชั้นบน เมื่อ source = upline
	IsRetired     bool   `json:"is_retired"`
}

// UsedListResponse — `used/list` (GN6)
type UsedListResponse struct {
	Source string     `json:"source"` // self / upline / global / mixed · "" = ไม่มีรูปเลย
	Items  []UsedItem `json:"items"`
}

// ThemeBase — 5 ค่าของธีม + shades (AST-33, AST-39)
type ThemeBase struct {
	Source             string            `json:"source"`
	FontFamily         string            `json:"font_family"`
	MainColor1         string            `json:"main_color_1"`
	MainColor2         string            `json:"main_color_2"`
	StatsBarColor      string            `json:"stats_bar_color"`
	StatsBarColorHex   string            `json:"stats_bar_color_hex"`
	StatsBarFontFamily string            `json:"stats_bar_font_family"`
	MainColor1Shades   map[string]string `json:"main_color_1_shades"`
	MainColor2Shades   map[string]string `json:"main_color_2_shades"`
}

// ThemeDetailResponse — `theme/detail/get` (GN9)
type ThemeDetailResponse struct {
	ThemeBase
	InheritedFrom string `json:"inherited_from"`
}

// BrandingResponse — `branding/detail/get` (GN12)
type BrandingResponse struct {
	Source        string `json:"source"` // self / upline / "" (ไม่มีใครตั้ง — ทุก field "")
	InheritedFrom string `json:"inherited_from"`
	SiteName      string `json:"site_name"`
	Logo          string `json:"logo"`
}

// LogoResponse — `branding/logo/upload` (GN14)
type LogoResponse struct {
	URL string `json:"url"`
}

// BundleSite — branding ใน bundle (ไม่มี inherited_from — AST-48)
type BundleSite struct {
	SiteName string `json:"site_name"`
	Logo     string `json:"logo"`
}

// BundleResponse — `bundle/detail/get` (GN22) · ผู้เล่นใช้รูปแบบเดียวกัน (GN23) แต่ theme ไม่มี inherited_from
type BundleResponse struct {
	Source string            `json:"source"`
	Assets map[string]string `json:"assets"` // asset_type → URL เฉพาะช่องที่มีไฟล์
	Theme  ThemeBase         `json:"theme"`
	Site   BundleSite        `json:"site"`
}
