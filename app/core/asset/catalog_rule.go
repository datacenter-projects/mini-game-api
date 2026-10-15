// Package asset คือกฎธุรกิจของหน้าตาเกม / หน้าเว็บ — spec: docs/modules/asset.md
// pure function ล้วน (ไม่แตะ DB / storage / fiber) · ทุกกฎมี table test
package asset

import (
	agentManagementCore "app/app/core/agent_management"
)

// ChannelGameCover — ชื่อช่องสงวนของรูปปกเกม (AST-51)
const ChannelGameCover = "game_cover"

// GameCodes — รหัสเกมที่ตั้งหน้าตาได้ = ชุดเดียวกับกลุ่ม PT (AST-10 · MGMT-16) เรียงตามลำดับในกลุ่ม
func GameCodes() []string {
	var out []string
	for _, g := range agentManagementCore.Groups() {
		for _, game := range agentManagementCore.GamesOf(g) {
			out = append(out, game.GameCode)
		}
	}
	return out
}

// IsGameCode — รหัสเกมต้องตรงทุกตัวอักษร (AST-14)
func IsGameCode(code string) bool {
	for _, c := range GameCodes() {
		if c == code {
			return true
		}
	}
	return false
}
