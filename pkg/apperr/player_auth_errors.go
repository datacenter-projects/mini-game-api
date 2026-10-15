package apperr

import "net/http"

// player_auth (module id 07) — spec: docs/modules/player_auth.md หัวข้อ 8 · HTTP status ตรงแบบ agent_auth (PA-27)
var (
	ErrPlayerInvalidCredentials   = NewWithStatus(http.StatusUnauthorized, 407201, "username หรือรหัสผ่านไม่ถูกต้อง", "Invalid username or password")
	ErrPlayerInvalidToken         = NewWithStatus(http.StatusUnauthorized, 407202, "กรุณาเข้าสู่ระบบ", "Missing or invalid token")
	ErrPlayerSessionEnded         = NewWithStatus(http.StatusUnauthorized, 407203, "เซสชันสิ้นสุดแล้ว กรุณาเข้าสู่ระบบใหม่", "Session has ended, please log in again")
	ErrPlayerKicked               = NewWithStatus(http.StatusUnauthorized, 407204, "บัญชีนี้ถูกเข้าสู่ระบบจากที่อื่น", "This account has been logged in from another place")
	ErrPlayerOldPasswordIncorrect = New(407205, "รหัสผ่านเดิมไม่ถูกต้อง", "Current password is incorrect")
	ErrOperatorInvalidKey         = NewWithStatus(http.StatusUnauthorized, 407206, "agent_username หรือ key ไม่ถูกต้อง", "Invalid agent_username or key")

	ErrPlayerLocked       = NewWithStatus(http.StatusForbidden, 407301, "บัญชีนี้ถูกล็อก กรุณาติดต่อผู้ดูแลระบบ", "This account is locked")
	ErrPlayerUplineLocked = NewWithStatus(http.StatusForbidden, 407302, "สายบัญชีต้นสังกัดถูกล็อก กรุณาติดต่อผู้ดูแลระบบ", "An upline account is locked")
	ErrPlayerLoginBlocked = NewWithStatus(http.StatusForbidden, 407303, "ใส่รหัสผ่านผิดหลายครั้ง กรุณารอสักครู่แล้วลองใหม่", "Too many incorrect passwords, please try again later")
	// 407304 เว้นไว้ — เดิมพันไม่ได้เพราะ SUSPENDED เป็นของ module Bet
	ErrOperatorIPNotAllowed   = NewWithStatus(http.StatusForbidden, 407305, "IP นี้ไม่ได้รับอนุญาตให้เรียก API", "This IP address is not allowed")
	ErrOperatorNotAllowed     = NewWithStatus(http.StatusForbidden, 407306, "บัญชีนี้ใช้ทางเข้านี้ไม่ได้", "This account cannot use this endpoint")
	ErrPlayerNoPassword       = New(407307, "บัญชีนี้ไม่มีรหัสผ่าน", "This account has no password")
	ErrPlayerCreatorSuspended = NewWithStatus(http.StatusForbidden, 407308, "บัญชีผู้สร้างถูกระงับ สร้างผู้เล่นใหม่ไม่ได้", "The creator account is suspended, new players cannot be created")

	ErrPlayerExists = New(407401, "มีผู้เล่นนี้แล้ว", "This player already exists")
)
