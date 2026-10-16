// Package asset คือ request ฝั่งผู้เล่นของ module asset — docs/modules/asset.md GN23
// response ใช้รูปแบบเดียวกับ preview ของ BO (assetService.BundleOfAgentService — AST-48)
package asset

import (
	"strings"

	"app/pkg/apperr"
)

// BundleRequest — POST /api/v1/player/pr/asset/bundle · มีแค่ game_code
// เป้าหมายมาจาก session ผู้เล่นเท่านั้น (ผู้สร้างตรง — AST-52) · field อื่นใน body (เช่น agent_id) ถูกเมิน
type BundleRequest struct {
	GameCode string `json:"game_code"`
}

func (r *BundleRequest) Validate() error {
	if strings.TrimSpace(r.GameCode) == "" {
		return apperr.ErrValidation.WithMessage("ต้องส่ง game_code", "game_code is required")
	}
	return nil
}
