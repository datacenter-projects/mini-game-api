package asset

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

// storage key / URL (AST-49) — key ใหม่ทุกครั้งที่อัป ไม่เขียนทับ object เดิม

// AssetObjectKey — key ของไฟล์ asset
//
//	GLOBAL: games/{game_code}/global/{asset_type}/{ts}-{rand}.{ext}
//	AGENT : games/{game_code}/agents/{agent_id}/{asset_type}/{ts}-{rand}.{ext}
func AssetObjectKey(scope Scope, agentID uint, gameCode, assetType string, unixNano int64, rand, ext string) string {
	if scope == ScopeGlobal {
		return fmt.Sprintf("games/%s/global/%s/%d-%s.%s", gameCode, assetType, unixNano, rand, ext)
	}
	return fmt.Sprintf("games/%s/agents/%d/%s/%d-%s.%s", gameCode, agentID, assetType, unixNano, rand, ext)
}

// LogoObjectKey — key ของโลโก้ sites/{agent_id}/logo/{ts}-{rand}.{ext}
func LogoObjectKey(agentID uint, unixNano int64, rand, ext string) string {
	return fmt.Sprintf("sites/%d/logo/%d-%s.%s", agentID, unixNano, rand, ext)
}

// ObjectURL — URL = S3_CDN_BASE_URL (ตัด / ท้าย) + "/" + key (ตัด / หน้า) · key ว่าง = "" (ACC-32)
// ไม่ตั้ง CDN base = คืน key ตรงๆ
func ObjectURL(cdnBase, key string) string {
	if key == "" {
		return ""
	}
	key = strings.TrimLeft(key, "/")
	base := strings.TrimRight(cdnBase, "/")
	if base == "" {
		return key
	}
	return base + "/" + key
}

// SiteNameMaxLen — ความยาวสูงสุดของชื่อเว็บ (นับเป็นตัวอักษร — AST-42)
const SiteNameMaxLen = 50

// NameViolation — เหตุที่ชื่อเว็บไม่ผ่าน
type NameViolation int

const (
	NameOK NameViolation = iota
	NameEmpty
	NameTooLong
	NameControlChar
)

// NormalizeSiteName — ตัดช่องว่างหน้า-หลัง แล้วต้องยาว 1–50 ตัวอักษร และห้ามอักขระควบคุม (AST-42 · lead GE13)
func NormalizeSiteName(s string) (string, NameViolation) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", NameEmpty
	}
	if utf8.RuneCountInString(s) > SiteNameMaxLen {
		return "", NameTooLong
	}
	for _, r := range s {
		if unicode.IsControl(r) {
			return "", NameControlChar
		}
	}
	return s, NameOK
}
