package asset

import (
	"strings"

	"app/pkg/apperr"
	"app/pkg/utils"
)

// request ของ module asset — scope + agent_id อ่านโดย middleware ResolveAssetTarget (AST-08) ไม่อยู่ใน DTO เหล่านี้
// ทุกเส้นเป็น POST · ห้ามส่ง null (ใช้ utils.ParseBodyNoNull — ACC-32) · Validate() ตรวจแค่การมีค่าและรูปแบบ
// การตรวจกับ catalog / สิทธิ์อยู่ที่ service

func requiredText(field, value string) error {
	if strings.TrimSpace(value) == "" {
		return apperr.ErrValidation.WithMessage("ต้องส่ง "+field, field+" is required")
	}
	return nil
}

func requiredID(field string, value uint) error {
	if value == 0 {
		return apperr.ErrValidation.WithMessage("ต้องส่ง "+field+" มากกว่า 0", field+" is required and must be greater than 0")
	}
	return nil
}

// GameRequest — เส้นที่ระบุแค่เกม: `slots/list` · `used/list` · `theme/detail/get` · `theme/reset` · `bundle/detail/get`
type GameRequest struct {
	GameCode string `json:"game_code"`
}

func (r *GameRequest) Validate() error { return requiredText("game_code", r.GameCode) }

// FilesListRequest — POST asset/files/list (AST-27) · page / limit ใน body (กฎข้อ 19)
type FilesListRequest struct {
	GameCode  string `json:"game_code"`
	AssetType string `json:"asset_type"`
	Page      int    `json:"page"`
	Limit     int    `json:"limit"`
}

func (r *FilesListRequest) Validate() error {
	if err := requiredText("game_code", r.GameCode); err != nil {
		return err
	}
	return requiredText("asset_type", r.AssetType)
}

// FilesDeleteRequest — POST asset/files/delete (AST-28)
type FilesDeleteRequest struct {
	ConfigID uint `json:"config_id"`
}

func (r *FilesDeleteRequest) Validate() error { return requiredID("config_id", r.ConfigID) }

// UsedUpdateRequest — POST asset/used/update (AST-23)
type UsedUpdateRequest struct {
	GameCode string `json:"game_code"`
	ConfigID uint   `json:"config_id"`
}

func (r *UsedUpdateRequest) Validate() error {
	if err := requiredText("game_code", r.GameCode); err != nil {
		return err
	}
	return requiredID("config_id", r.ConfigID)
}

// UsedResetRequest — POST asset/used/reset (AST-54)
type UsedResetRequest struct {
	GameCode  string `json:"game_code"`
	AssetType string `json:"asset_type"`
}

func (r *UsedResetRequest) Validate() error {
	if err := requiredText("game_code", r.GameCode); err != nil {
		return err
	}
	return requiredText("asset_type", r.AssetType)
}

// ThemeUpdateRequest — POST asset/theme/detail/update · ต้องส่งครบ 5 ค่า = แทนทั้งชุด (AST-33)
// ค่าที่ขาดเป็น "" แล้ว core.NormalizeTheme ตอบว่า field ไหนขาด / ผิด
type ThemeUpdateRequest struct {
	GameCode           string `json:"game_code"`
	FontFamily         string `json:"font_family"`
	MainColor1         string `json:"main_color_1"`
	MainColor2         string `json:"main_color_2"`
	StatsBarColor      string `json:"stats_bar_color"`
	StatsBarFontFamily string `json:"stats_bar_font_family"`
}

func (r *ThemeUpdateRequest) Validate() error { return requiredText("game_code", r.GameCode) }

// BrandingUpdateRequest — POST asset/branding/detail/update · MVP รับ site_name อย่างเดียว (AST-42)
// site_theme ส่งมา = 422 (เลื่อน — lead GE13-1)
type BrandingUpdateRequest struct {
	SiteName  string           `json:"site_name"`
	SiteTheme utils.JSONString `json:"site_theme"`
}

func (r *BrandingUpdateRequest) Validate() error {
	if r.SiteTheme.Present {
		return apperr.ErrValidation.WithMessage("site_theme ยังไม่เปิดใช้งาน", "site_theme is not supported yet")
	}
	return nil // site_name ตรวจด้วย core.NormalizeSiteName ที่ service (ตัดช่องว่างก่อน)
}

// UploadRequest — field ของ multipart/form-data ของ `files/upload` (AST-21) · scope / agent_id อ่านโดย middleware · ไฟล์อ่านที่ controller
type UploadRequest struct {
	GameCode  string `json:"game_code" form:"game_code"`
	AssetType string `json:"asset_type" form:"asset_type"`
	UseNow    string `json:"use_now" form:"use_now"` // "" = true (ค่าเริ่มต้น) · "true" / "false"

	// ค่าหลังตรวจแล้ว — service ใช้ค่านี้
	UseNowValue bool `json:"-" form:"-"`
}

func (r *UploadRequest) Validate() error {
	if err := requiredText("game_code", r.GameCode); err != nil {
		return err
	}
	if err := requiredText("asset_type", r.AssetType); err != nil {
		return err
	}
	switch strings.TrimSpace(r.UseNow) {
	case "", "true":
		r.UseNowValue = true
	case "false":
		r.UseNowValue = false
	default:
		return apperr.ErrValidation.WithMessage(`use_now ต้องเป็น "true" หรือ "false"`, `use_now must be "true" or "false"`)
	}
	return nil
}
