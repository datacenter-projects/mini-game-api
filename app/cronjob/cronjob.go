// Package cronjob คือที่เดียวของงานตามเวลา (docs/modules/currency_rate.md J1 / CJ-01)
//
//	cronjob.go        Start(ctx) — จุดเดียวที่ application.go เรียก · ตัวตั้งเวลา + metrics + log
//	{job}_job.go      1 ไฟล์ต่อ job: ชื่อ (code) · ค่าเริ่มต้นเปิด/ปิด · ความถี่ · เรียก service ของ module
//
// กฎ: ไม่มี business logic · **เรียกได้แค่ service** (ห้ามเรียก repository / ต่อ DB เอง — scripts/check_structure.sh ตรวจ)
// รันใน process เดียวกับ API เฉพาะ pod ที่ IS_CRONJOB=true (เปิด deployment เดียว) · ยังมี Redis lock ของแต่ละ job กันซ้ำ
// เปิด / ปิดแต่ละ job ที่ตาราง cronjobs (ทุกรอบอ่าน enabled ก่อน — ปิด = ข้ามเงียบ)
package cronjob

import (
	"context"
	"time"

	cronjobService "app/app/service/cronjob"
	"app/pkg/configs"
	"app/platform/logger"
	"app/platform/metrics"
)

// runTimeout — เวลาสูงสุดของ 1 รอบ (ไม่เกินอายุ lock ของ job — CR-13)
const runTimeout = 5 * time.Minute

// job คือนิยามของงานตามเวลา 1 ตัว
type job struct {
	code           string                                           // PK ของตาราง cronjobs
	defaultEnabled bool                                             // ค่าเริ่มต้นตอนสร้างแถวครั้งแรก (CJ-03) — job เสี่ยงในอนาคตตั้ง false
	interval       time.Duration                                    // ความถี่อยู่ในโค้ด ไม่เก็บในตาราง (T1)
	run            func(ctx context.Context) cronjobService.Outcome // เรียก service ของ module
}

// jobs คือทะเบียนงานทั้งหมด — เพิ่ม job ใหม่ที่นี่
var jobs = []job{currencyRateJob}

// Start เริ่มทุก job ถ้า pod นี้เป็น cronjob pod (IS_CRONJOB=true) — เรียกครั้งเดียวหลังต่อ DB / Redis แล้ว
// สร้างแถวใน cronjobs ถ้ายังไม่มี (ไม่ทับค่าที่ตั้งไว้) แล้วรัน 1 รอบทันทีและทุก interval จนกว่า ctx จะถูกยกเลิก
// goroutine ต่อ job 1 ตัว (จำนวนเท่าทะเบียน — มีขอบเขต) แต่ละรอบทำต่อกันไม่ซ้อน (กฎข้อ 27)
func Start(ctx context.Context) {
	if !configs.Cfg.IsCronjob {
		logger.Ctx(ctx).Infow("cronjob disabled on this pod (IS_CRONJOB=false)")
		return
	}
	for _, j := range jobs {
		if err := cronjobService.EnsureService(ctx, j.code, j.defaultEnabled); err != nil {
			logger.Ctx(ctx).Errorw("cronjob: ensure row failed — job will not run until restart", "job", j.code, "error", err)
			continue
		}
		go loop(ctx, j)
		logger.Ctx(ctx).Infow("cronjob started", "job", j.code, "interval", j.interval.String())
	}
}

func loop(ctx context.Context, j job) {
	ticker := time.NewTicker(j.interval)
	defer ticker.Stop()
	for {
		runOnce(ctx, j)
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

// RunOnce รัน 1 รอบของ job ตาม code (ผ่านตัวห่อเดียวกับตัวตั้งเวลา) — ใช้ใน test · ไม่รู้จัก code = false
func RunOnce(ctx context.Context, code string) bool {
	for _, j := range jobs {
		if j.code == code {
			runOnce(ctx, j)
			return true
		}
	}
	return false
}

func runOnce(ctx context.Context, j job) {
	rctx, cancel := context.WithTimeout(ctx, runTimeout)
	defer cancel()
	res, err := cronjobService.RunService(rctx, j.code, j.run)
	if err != nil {
		logger.Ctx(ctx).Warnw("cronjob: run failed before start", "job", j.code, "error", err)
		return
	}
	if !res.Ran { // ปิดอยู่ — ข้ามเงียบ ไม่นับ metrics ไม่แก้แถว (CR-12)
		return
	}
	metrics.CronjobRunsTotal.WithLabelValues(j.code, string(res.Outcome.Status)).Inc()
	switch res.Outcome.Status {
	case cronjobService.StatusSuccess:
		metrics.CronjobLastSuccessTimestamp.WithLabelValues(j.code).Set(float64(time.Now().Unix()))
	case cronjobService.StatusFailed:
		logger.Ctx(ctx).Warnw("cronjob run failed", "job", j.code, "error", res.Outcome.Err)
	}
}
