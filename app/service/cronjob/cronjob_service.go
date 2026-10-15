// Package cronjob คือ business logic ของตาราง cronjobs: ห่อทุก job ตามเวลา (เปิด / ปิด · ผลรอบล่าสุด) — spec: docs/modules/currency_rate.md หัวข้อ 6.2
// app/cronjob (ตัวตั้งเวลา) เรียกได้แค่ service นี้และ service ของ module (CLAUDE.md — กฎ cronjob)
package cronjob

import (
	"context"
	"errors"
	"time"

	"app/app/models"
	cronjobPostgres "app/app/repository/postgres/cronjob"
	"app/pkg/apperr"
	"app/platform/database"
	"app/platform/logger"
)

// Status — ผลของรอบ (ค่าเดียวกับ cronjobs.last_status)
type Status string

const (
	StatusSuccess Status = Status(models.CronjobStatusSuccess)
	StatusFailed  Status = Status(models.CronjobStatusFailed)
	StatusSkipped Status = Status(models.CronjobStatusSkipped)
)

// Outcome — ผลที่ job คืนให้ตัวห่อ · Err ใช้เมื่อ Status = FAILED (บันทึกลง last_error)
type Outcome struct {
	Status Status
	Err    error
}

// maxErrorLen — last_error ยาวไม่เกินนี้ (ตรงกับ VARCHAR(500) — CJ-04)
const maxErrorLen = 500

// EnsureService — boot: สร้างแถวของ job ถ้ายังไม่มี ด้วยค่าเริ่มต้นในโค้ด · มีแล้วไม่ทับ (CJ-02 / CJ-03)
func EnsureService(ctx context.Context, code string, defaultEnabled bool) error {
	return cronjobPostgres.EnsureCronjobRepository(database.DBConn.WithContext(ctx), code, defaultEnabled)
}

// RunResult — ผลของ RunService
type RunResult struct {
	Ran     bool // false = job ปิดอยู่ (ข้ามเงียบ ไม่แก้แถว — CJ-04 / CR-12)
	Outcome Outcome
}

// RunService ห่อ 1 รอบของ job: อ่าน enabled ก่อน — ปิด = ข้ามเงียบ ไม่แก้แถว · เปิด = รัน run แล้วบันทึก last_* (CJ-04)
// ไม่มีแถว (ยังไม่เคย Ensure) ถือว่าปิด · error ของการอ่าน / บันทึกสถานะไม่ทำให้ job ล้ม (log warn) แต่ถ้าอ่าน enabled ไม่ได้จะไม่รัน (ปลอดภัยไว้ก่อน)
func RunService(ctx context.Context, code string, run func(ctx context.Context) Outcome) (RunResult, error) {
	db := database.DBConn.WithContext(ctx)
	job, err := cronjobPostgres.GetCronjobRepository(db, code)
	if errors.Is(err, apperr.ErrNotFound) {
		return RunResult{}, nil
	}
	if err != nil {
		return RunResult{}, err
	}
	if !job.Enabled {
		return RunResult{}, nil
	}

	started := time.Now()
	out := run(ctx)
	finished := time.Now()

	msg := ""
	if out.Status == StatusFailed && out.Err != nil {
		msg = truncate(out.Err.Error(), maxErrorLen)
	}
	if err := cronjobPostgres.RecordCronjobRunRepository(db, code, started, finished, models.CronjobStatus(out.Status), msg); err != nil {
		logger.Ctx(ctx).Warnw("record cronjob run failed", "job", code, "error", err)
	}
	return RunResult{Ran: true, Outcome: out}, nil
}

// truncate ตัดข้อความไม่เกิน n ตัวอักษร (rune) ไม่ตัดกลางตัวอักษร
func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n])
}
