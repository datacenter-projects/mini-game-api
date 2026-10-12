// Package game คือรายชื่อเกมของระบบ (game_rate RATE-01A · agent_management MGMT-16 / MGMT-20) — pure ไม่มี DB / config
// เพิ่มเกมใหม่ = แก้ที่นี่พร้อมเอนจิ้นของเกมนั้น · ไม่มีเส้นสร้างเกม
package game

// Group — กลุ่มเกม (ใช้เป็นกลุ่ม PT ของ agent_management — MGMT-16)
type Group string

const GroupMinigame Group = "minigame"

// รหัสเกม
const (
	CoinToss          = "coin_toss"
	RockPaperScissors = "rock_paper_scissors"
	ScratchCard       = "scratch_card"
)

// Game — เกมหนึ่งเกม
type Game struct {
	Category string
	GameCode string
}

// groupGames — กลุ่ม → เกม เรียงตามลำดับที่แสดง (scratch_card → coin_toss → rock_paper_scissors — 2026-10-12)
var groupGames = map[Group][]Game{
	GroupMinigame: {
		{"minigame", ScratchCard},
		{"minigame", CoinToss},
		{"minigame", RockPaperScissors},
	},
}

// Groups — กลุ่มทั้งหมดเรียงตามชื่อ
func Groups() []Group { return []Group{GroupMinigame} }

// GamesOf — เกมในกลุ่ม · กลุ่มที่ไม่มี = nil
func GamesOf(g Group) []Game { return groupGames[g] }

// All — ทุกเกม เรียงตามกลุ่มแล้วตามลำดับในกลุ่ม (ลำดับของ game_rate RE1)
func All() []Game {
	var out []Game
	for _, g := range Groups() {
		out = append(out, groupGames[g]...)
	}
	return out
}

// GroupOf — กลุ่มของรหัสเกม · ไม่พบ = ok false
func GroupOf(gameCode string) (Group, bool) {
	for g, games := range groupGames {
		for _, gg := range games {
			if gg.GameCode == gameCode {
				return g, true
			}
		}
	}
	return "", false
}

// IsGame — รหัสเกมนี้มีในระบบไหม
func IsGame(gameCode string) bool {
	_, ok := GroupOf(gameCode)
	return ok
}
