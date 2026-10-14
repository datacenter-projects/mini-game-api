package apperr

import "net/http"

// deposit_withdraw (module id 04) — spec: docs/modules/deposit_withdraw.md
var (
	ErrDWTargetInactive     = New(404301, "ฝากให้บัญชีที่ถูกระงับหรือล็อกไม่ได้", "Cannot deposit to a suspended or locked account")
	ErrDWSeamless           = New(404302, "บัญชีฝั่ง Seamless ไม่มียอดเงิน ฝากหรือถอนไม่ได้", "Seamless accounts have no balance to deposit or withdraw")
	ErrDWCurrency           = New(404303, "สกุลเงินนี้ไม่มีในบัญชีของคุณหรือบัญชีปลายทาง", "This currency is not available on your account or the target account")
	ErrDWParentInsufficient = New(404304, "ยอดเงินของคุณไม่พอ", "Your balance is insufficient")
	ErrDWAdminNotAllowed    = New(404305, "ADMIN ทำรายการฝากหรือถอนไม่ได้", "ADMIN cannot deposit or withdraw")
	ErrDWTargetInsufficient = New(404306, "ยอดเงินของบัญชีปลายทางไม่พอ", "The target account balance is insufficient")
	ErrDWInProgress         = New(404307, "มีรายการของบัญชีนี้กำลังทำอยู่ กรุณาลองใหม่", "Another transaction for this account is in progress, please try again")
	ErrDWTargetNotFound     = New(404401, "ไม่พบบัญชีลูกตรงของคุณ", "Direct downline account not found")
	ErrDWLockUnavailable    = NewWithStatus(http.StatusServiceUnavailable, 504801, "ระบบไม่พร้อมทำรายการ กรุณาลองใหม่", "The system is not ready, please try again")
)
