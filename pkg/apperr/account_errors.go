package apperr

// account (module id 03) — spec: docs/modules/account.md
// 403302 จองถาวร (ตัดก่อนปล่อยใช้ — AUTH-54 กันที่ middleware แทน)
var (
	ErrNoAPICredential = New(403301, "บัญชีนี้ไม่มีข้อมูลรับรอง API", "This account has no API credentials")
)
