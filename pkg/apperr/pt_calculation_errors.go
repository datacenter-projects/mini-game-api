package apperr

import "net/http"

// pt_calculation (module id 08) — spec: docs/modules/pt_calculation.md
// 408301 จองถาวร (ตัดก่อนปล่อยใช้ — lead Q2: ชั้นไม่รับ PT ถือ 0 ไม่ error)
var (
	ErrPTCMemberNotFound = New(408401, "ไม่พบ Member", "Member not found")
	ErrPTCInvalidChain   = NewWithStatus(http.StatusInternalServerError, 508601,
		"ข้อมูล PT ในสายไม่ถูกต้อง คิด PT ไม่ได้", "PT settings of this line are invalid, cannot calculate PT")
)
