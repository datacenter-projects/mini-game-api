package cronjob

import (
	"context"
	"time"

	cronjobService "app/app/service/cronjob"
	currencyRateService "app/app/service/currency_rate"
)

// currencyRateJob — ดึงเรทรายเดือนจาก askmelotto (docs/modules/currency_rate.md CR-10 – CR-16)
// ทุก 15 นาที + รันทันทีตอน boot · เปิดเป็นค่าเริ่มต้น (CJ-03) · logic ทั้งหมดอยู่ใน service
var currencyRateJob = job{
	code:           "currency_rate_sync",
	defaultEnabled: true,
	interval:       15 * time.Minute,
	run: func(ctx context.Context) cronjobService.Outcome {
		res, err := currencyRateService.SyncCurrencyRateService(ctx, time.Now(), currencyRateService.SyncOptions{})
		switch {
		case err != nil: // ยิงไม่สำเร็จ / response ผิด / ไม่ครบ 27 สกุล → FAILED (log ระดับ info/warn ทำใน service แล้ว — CR-16)
			return cronjobService.Outcome{Status: cronjobService.StatusFailed, Err: err}
		case res.Skipped: // ไม่ได้ lock (CR-13)
			return cronjobService.Outcome{Status: cronjobService.StatusSkipped}
		default: // เดือนนี้ครบแล้ว (ไม่ยิง) หรือครบ 27 สกุลหลังรอบนี้
			return cronjobService.Outcome{Status: cronjobService.StatusSuccess}
		}
	},
}
