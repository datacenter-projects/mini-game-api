package apperr

import "net/http"

// agent_auth (module id 01) — spec: docs/modules/agent_auth.md, docs/modules/agent_auth_phase2.md
var (
	ErrInvalidCredentials = NewWithStatus(http.StatusUnauthorized, 401201, "username หรือรหัสผ่านไม่ถูกต้อง", "Invalid username or password")
	ErrInvalidToken       = NewWithStatus(http.StatusUnauthorized, 401202, "กรุณาเข้าสู่ระบบ", "Missing or invalid token")
	ErrSessionEnded       = NewWithStatus(http.StatusUnauthorized, 401203, "เซสชันสิ้นสุดแล้ว กรุณาเข้าสู่ระบบใหม่", "Session has ended, please log in again")
	// ErrPasscodeIncorrect ใช้คู่กับ WithMessage เพื่อบอกจำนวนครั้งที่เหลือ (AUTH-35)
	ErrPasscodeIncorrect    = New(401204, "passcode ไม่ถูกต้อง", "Incorrect passcode")
	ErrPasscodeTooManyFails = NewWithStatus(http.StatusUnauthorized, 401205, "ใส่ passcode ผิดครบ 5 ครั้ง ระบบออกจากระบบและระงับการเข้าใช้ 1 ชั่วโมง", "Too many incorrect passcodes, you have been logged out and blocked for 1 hour")
	ErrOldPasswordIncorrect = New(401206, "รหัสผ่านเดิมไม่ถูกต้อง", "Current password is incorrect")

	ErrAccountLocked           = NewWithStatus(http.StatusForbidden, 401301, "บัญชีนี้ถูกล็อก กรุณาติดต่อผู้ดูแลระบบ", "This account is locked")
	ErrUplineLocked            = NewWithStatus(http.StatusForbidden, 401302, "สายบัญชีต้นสังกัดถูกล็อก กรุณาติดต่อผู้ดูแลระบบ", "An upline account is locked")
	ErrLoginTemporarilyBlocked = NewWithStatus(http.StatusForbidden, 401303, "เข้าสู่ระบบผิดหลายครั้ง กรุณารอสักครู่แล้วลองใหม่", "Too many failed login attempts, please try again later")
	ErrPasscodeNotSet          = NewWithStatus(http.StatusForbidden, 401304, "กรุณาตั้ง passcode ก่อนใช้งาน", "Please set your passcode first")
	// 401305 จองถาวร (ตัดก่อนปล่อยใช้)
	ErrMustChangePassword    = NewWithStatus(http.StatusForbidden, 401306, "กรุณาเปลี่ยนรหัสผ่านก่อนใช้งาน", "Please change your password first")
	ErrMustChangePasscode    = NewWithStatus(http.StatusForbidden, 401307, "กรุณาเปลี่ยน passcode ก่อนใช้งาน", "Please change your passcode first")
	ErrAdminOnly             = NewWithStatus(http.StatusForbidden, 401308, "ไม่มีสิทธิ์ใช้งานส่วนของ admin", "Admin access required")
	ErrPasscodeBlocked       = NewWithStatus(http.StatusForbidden, 401309, "ระงับการเข้าสู่ระบบชั่วคราวเพราะใส่ passcode ผิดหลายครั้ง", "Login is temporarily blocked due to too many incorrect passcodes")
	ErrTempCredentialExpired = NewWithStatus(http.StatusForbidden, 401310, "รหัสชั่วคราวหมดอายุแล้ว กรุณาติดต่อ admin", "Temporary credential has expired, please contact admin")
	ErrAccountSuspended      = NewWithStatus(http.StatusForbidden, 401311, "บัญชีถูกระงับ ใช้งานได้เฉพาะหน้าประวัติของฉันและรายงาน", "This account is suspended, only Profile and Report are available")

	ErrPasscodeAlreadySet    = New(401401, "ตั้ง passcode ไว้แล้ว", "Passcode is already set")
	ErrPasswordReused        = New(401402, "รหัสผ่านใหม่ต้องไม่ซ้ำกับรหัสผ่านที่เคยใช้", "New password must not match a recently used password")
	ErrPasscodeReused        = New(401403, "passcode ใหม่ต้องไม่ซ้ำกับ passcode เดิม", "New passcode must differ from the current passcode")
	ErrResetTargetNotFound   = New(401404, "ไม่พบบัญชีที่ต้องการรีเซ็ต", "Account to reset not found")
	ErrResetTargetNoPasscode = New(401405, "บัญชีนี้ยังไม่ได้ตั้ง passcode", "This account has not set a passcode")
	ErrResetNotAllowed       = New(401406, "ไม่สามารถรีเซ็ตบัญชีนี้ได้", "This account cannot be reset")
	ErrResetTargetLocked     = New(401407, "บัญชีเป้าหมายถูกล็อก", "Target account is locked")
)
