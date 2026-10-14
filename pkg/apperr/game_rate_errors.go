package apperr

// game_rate (module id 06 · เมนู ⑥) — spec: docs/modules/game_rate.md หัวข้อ 8
var (
	// ErrRateHouseEdge — ใช้คู่กับ WithMessage บอกขั้น / แถว และ win_percent สูงสุดที่ตั้งได้ (RATE-07)
	ErrRateHouseEdge = New(406101, "ตั้งค่านี้แล้วร้านได้ต่ำกว่าขั้นต่ำ 1%", "House edge would be below the 1% minimum")
	ErrRateShareSum  = New(406102, "สัดส่วนรางวัลรวมต้องเท่ากับ 100%", "Prize shares must add up to 100%")
	// ErrRateStale — base_version ไม่ตรงเวอร์ชันปัจจุบัน · ใช้คู่กับ WithMessage บอกเวอร์ชันล่าสุดและคนแก้ (RATE-05B · lead Q5)
	ErrRateStale           = New(406301, "มีคนแก้ไปแล้ว กรุณาโหลดใหม่", "Rates were changed by someone else, please reload")
	ErrRateGameNotFound    = New(406401, "ไม่พบเกมนี้", "Game not found")
	ErrRateVersionNotFound = New(406402, "ไม่พบเวอร์ชันนี้", "Version not found")
)
