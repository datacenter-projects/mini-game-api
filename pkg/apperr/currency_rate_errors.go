package apperr

// currency_rate (module id 09) — spec: docs/modules/currency_rate.md หัวข้อ 8 · ไม่มี endpoint จึงมี error เดียว
var (
	// ErrCurrencyRateUnavailable — ไม่มีเรทของเดือน / สกุลนี้ (CR-24) · ไม่ใช้เรทเดือนก่อน
	// module เดิมพันแปลงเป็น code ของตัวเองและปฏิเสธการแทง (fail closed — lead C1) ไม่ส่งเลขนี้ถึง client ตรงๆ
	ErrCurrencyRateUnavailable = New(409401, "ไม่มีอัตราแลกเปลี่ยนของเดือนนี้", "Exchange rate is not available")
)
