package apperr

import "net/http"

// agent_auth (module id 01) — spec: docs/modules/agent_auth.md
var (
	ErrInvalidCredentials = NewWithStatus(http.StatusUnauthorized, 401201, "username หรือรหัสผ่านไม่ถูกต้อง", "Invalid username or password")
	ErrInvalidToken       = NewWithStatus(http.StatusUnauthorized, 401202, "กรุณาเข้าสู่ระบบ", "Missing or invalid token")
	ErrSessionEnded       = NewWithStatus(http.StatusUnauthorized, 401203, "เซสชันสิ้นสุดแล้ว กรุณาเข้าสู่ระบบใหม่", "Session has ended, please log in again")

	ErrAccountLocked           = NewWithStatus(http.StatusForbidden, 401301, "บัญชีนี้ถูกล็อก กรุณาติดต่อผู้ดูแลระบบ", "This account is locked")
	ErrUplineLocked            = NewWithStatus(http.StatusForbidden, 401302, "สายบัญชีต้นสังกัดถูกล็อก กรุณาติดต่อผู้ดูแลระบบ", "An upline account is locked")
	ErrLoginTemporarilyBlocked = NewWithStatus(http.StatusForbidden, 401303, "เข้าสู่ระบบผิดหลายครั้ง กรุณารอสักครู่แล้วลองใหม่", "Too many failed login attempts, please try again later")
)
