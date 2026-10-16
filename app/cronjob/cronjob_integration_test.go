//go:build integration

// integration test ของ app/cronjob + ตาราง cronjobs + metrics (docs/modules/currency_rate.md CR-10 – CR-13 · CJ-01 – CJ-06)
package cronjob_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"app/app/cronjob"
	"app/app/models"
	"app/pkg/configs"
	"app/pkg/testutil"
	"app/platform/database"
	"app/platform/metrics"

	dto "github.com/prometheus/client_model/go"
)

const code = "currency_rate_sync"

func setup(t *testing.T) *testutil.FakeAskmelotto {
	t.Helper()
	testutil.Setup(t)
	configs.Cfg.IsCronjob = true
	return testutil.NewFakeAskmelotto(t)
}

func row(t *testing.T) models.Cronjob {
	t.Helper()
	var c models.Cronjob
	if err := database.DBConn.Where("code = ?", code).Take(&c).Error; err != nil {
		t.Fatalf("ไม่พบแถว cronjobs: %v", err)
	}
	return c
}

func countRows(t *testing.T) int64 {
	t.Helper()
	var n int64
	database.DBConn.Model(&models.Cronjob{}).Count(&n)
	return n
}

func rateRows(t *testing.T) int64 {
	t.Helper()
	var n int64
	database.DBConn.Model(&models.CurrencyRate{}).Count(&n)
	return n
}

// waitStatus รอให้รอบแรกที่ Start เริ่มเองทันทีตอน boot เสร็จ (บันทึก last_status)
func waitStatus(t *testing.T) models.Cronjob {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		var c models.Cronjob
		if err := database.DBConn.Where("code = ?", code).Take(&c).Error; err == nil && c.LastStatus != "" {
			return c
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatal("รอบแรกไม่ทำงานภายใน 10 วินาที")
	return models.Cronjob{}
}

// counter อ่านค่า cronjob_runs_total{job,status} / lastSuccess อ่าน cronjob_last_success_timestamp{job} — อ่านผ่าน client_model ตรงๆ ไม่เพิ่ม dependency
func counter(status string) float64 {
	m := &dto.Metric{}
	_ = metrics.CronjobRunsTotal.WithLabelValues(code, status).Write(m)
	return m.GetCounter().GetValue()
}

func lastSuccess() float64 {
	m := &dto.Metric{}
	_ = metrics.CronjobLastSuccessTimestamp.WithLabelValues(code).Write(m)
	return m.GetGauge().GetValue()
}

func TestStartDisabledOnThisPod(t *testing.T) { // CR-11: IS_CRONJOB=false ไม่เริ่ม cronjob · ไม่ยิง · ไม่สร้างแถว
	fake := setup(t)
	configs.Cfg.IsCronjob = false
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	cronjob.Start(ctx)
	time.Sleep(300 * time.Millisecond)
	if countRows(t) != 0 || fake.Hits() != 0 || rateRows(t) != 0 {
		t.Errorf("rows %d hits %d rates %d — ต้องไม่ทำอะไรเลย", countRows(t), fake.Hits(), rateRows(t))
	}
}

func TestStartRunsImmediatelyAndRecords(t *testing.T) { // CR-10 (รันทันทีตอน boot) · CJ-02 · CJ-03 · CJ-04 · CJ-06
	fake := setup(t)
	okBefore := counter("SUCCESS")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	cronjob.Start(ctx)

	c := waitStatus(t)
	if !c.Enabled || c.UpdatedBy != "system" || c.CreatedAt.IsZero() { // CJ-03: ค่าเริ่มต้นของ currency_rate_sync = เปิด
		t.Errorf("แถว = %+v", c)
	}
	if c.LastStatus != models.CronjobStatusSuccess || c.LastError != "" || c.LastStartedAt == nil || c.LastFinishedAt == nil || c.LastFinishedAt.Before(*c.LastStartedAt) {
		t.Errorf("last_* = %+v", c)
	}
	if rateRows(t) != 27 || fake.Hits() != 1 {
		t.Errorf("rates %d hits %d", rateRows(t), fake.Hits())
	}
	if got := counter("SUCCESS") - okBefore; got != 1 {
		t.Errorf("cronjob_runs_total{SUCCESS} เพิ่ม %v want 1", got)
	}
	if ts := lastSuccess(); time.Since(time.Unix(int64(ts), 0)) > time.Minute {
		t.Errorf("cronjob_last_success_timestamp = %v", ts)
	}
}

func TestBootDoesNotOverwriteEnabled(t *testing.T) { // CJ-02: boot ครั้งที่ 2 หลังปิดด้วย SQL ต้องไม่ทับ
	setup(t)
	ctx, cancel := context.WithCancel(context.Background())
	cronjob.Start(ctx)
	waitStatus(t)
	cancel()

	if err := database.DBConn.Exec(`UPDATE cronjobs SET enabled = false, updated_by = 'sql:zerph', updated_at = now() WHERE code = ?`, code).Error; err != nil {
		t.Fatal(err)
	}
	ctx2, cancel2 := context.WithCancel(context.Background())
	defer cancel2()
	cronjob.Start(ctx2) // boot อีกรอบ
	time.Sleep(300 * time.Millisecond)
	c := row(t)
	if c.Enabled || c.UpdatedBy != "sql:zerph" {
		t.Errorf("boot ทับค่าที่ตั้งด้วย SQL: %+v", c)
	}
}

func TestDisabledJobSkipsSilently(t *testing.T) { // CR-12 · CJ-04: ปิด = ข้ามเงียบ ไม่ยิง ไม่แก้แถว ไม่นับ metrics
	fake := setup(t)
	database.DBConn.Create(&models.Cronjob{Code: code, Enabled: false, UpdatedBy: "sql:zerph", LastStatus: models.CronjobStatusFailed, LastError: "old error"})
	before := row(t)
	skipped, ok, failed := counter("SKIPPED"), counter("SUCCESS"), counter("FAILED")

	if !cronjob.RunOnce(context.Background(), code) {
		t.Fatal("ไม่รู้จัก job")
	}
	after := row(t)
	if fake.Hits() != 0 || rateRows(t) != 0 {
		t.Errorf("ปิดอยู่ต้องไม่ยิง: hits %d rates %d", fake.Hits(), rateRows(t))
	}
	if after.LastStatus != before.LastStatus || after.LastError != before.LastError || after.LastStartedAt != nil || !after.UpdatedAt.Equal(before.UpdatedAt) {
		t.Errorf("ปิดอยู่ต้องไม่แก้แถว: %+v → %+v", before, after)
	}
	if counter("SKIPPED") != skipped || counter("SUCCESS") != ok || counter("FAILED") != failed {
		t.Error("ปิดอยู่ต้องไม่นับ metrics")
	}

	// เปิดแล้วรันได้
	database.DBConn.Exec(`UPDATE cronjobs SET enabled = true WHERE code = ?`, code)
	cronjob.RunOnce(context.Background(), code)
	if fake.Hits() != 1 || row(t).LastStatus != models.CronjobStatusSuccess || row(t).LastError != "" {
		t.Errorf("เปิดแล้วต้องรัน: hits %d row %+v", fake.Hits(), row(t))
	}
}

func TestMissingRowCountsAsDisabled(t *testing.T) { // ไม่มีแถว (ยังไม่เคย Ensure) = ไม่รัน (ปลอดภัยไว้ก่อน)
	fake := setup(t)
	cronjob.RunOnce(context.Background(), code)
	if fake.Hits() != 0 || countRows(t) != 0 {
		t.Errorf("hits %d rows %d", fake.Hits(), countRows(t))
	}
}

func TestFailedRunRecordsError(t *testing.T) { // CJ-04 · CJ-06: FAILED + last_error + metrics · last_success ไม่ขยับ
	fake := setup(t)
	database.DBConn.Create(&models.Cronjob{Code: code, Enabled: true})
	fake.Set(func(c *testutil.FakeAskmelottoConfig) { c.Code, c.Msg = 430503, "no rate" })
	failedBefore := counter("FAILED")
	tsBefore := lastSuccess()

	cronjob.RunOnce(context.Background(), code)
	c := row(t)
	if c.LastStatus != models.CronjobStatusFailed || !strings.Contains(c.LastError, "430503") || c.LastStartedAt == nil || c.LastFinishedAt == nil {
		t.Errorf("row = %+v", c)
	}
	if counter("FAILED")-failedBefore != 1 {
		t.Error("cronjob_runs_total{FAILED} ต้องเพิ่ม 1")
	}
	if lastSuccess() != tsBefore {
		t.Error("รอบที่ล้มต้องไม่ขยับ last_success_timestamp")
	}

	// รอบถัดไปสำเร็จ → last_error ถูกล้าง
	fake.Reset()
	cronjob.RunOnce(context.Background(), code)
	if c := row(t); c.LastStatus != models.CronjobStatusSuccess || c.LastError != "" {
		t.Errorf("หลังสำเร็จ: %+v", c)
	}
}

func TestIncompleteRunIsFailed(t *testing.T) { // CR-14: ไม่ครบ 27 = FAILED บอกสกุลที่ขาด (สกุลที่ได้บันทึกแล้ว)
	fake := setup(t)
	database.DBConn.Create(&models.Cronjob{Code: code, Enabled: true})
	rates := testutil.DefaultAskmelottoRates()
	delete(rates, "KRW")
	fake.Set(func(c *testutil.FakeAskmelottoConfig) { c.Rates = rates })

	cronjob.RunOnce(context.Background(), code)
	if c := row(t); c.LastStatus != models.CronjobStatusFailed || !strings.Contains(c.LastError, "KRW") {
		t.Errorf("row = %+v", c)
	}
	if rateRows(t) != 26 {
		t.Errorf("ต้องบันทึกสกุลที่ได้ 26: %d", rateRows(t))
	}
}

func TestCompleteMonthIsSuccessWithoutCall(t *testing.T) { // CR-10 / CR-14: ครบแล้ว = SUCCESS ไม่ยิง
	fake := setup(t)
	database.DBConn.Create(&models.Cronjob{Code: code, Enabled: true})
	cronjob.RunOnce(context.Background(), code)
	cronjob.RunOnce(context.Background(), code)
	if fake.Hits() != 1 || row(t).LastStatus != models.CronjobStatusSuccess {
		t.Errorf("hits %d row %+v", fake.Hits(), row(t))
	}
}

func TestLockHeldIsSkipped(t *testing.T) { // CR-13 · CJ-04: lock ไม่ได้ = SKIPPED
	fake := setup(t)
	database.DBConn.Create(&models.Cronjob{Code: code, Enabled: true})
	if err := database.DBRedis.Set(context.Background(), "currency_rate:sync:lock", "other-pod", time.Minute).Err(); err != nil {
		t.Fatal(err)
	}
	skippedBefore := counter("SKIPPED")
	cronjob.RunOnce(context.Background(), code)
	c := row(t)
	if c.LastStatus != models.CronjobStatusSkipped || fake.Hits() != 0 || counter("SKIPPED")-skippedBefore != 1 {
		t.Errorf("row %+v hits %d", c, fake.Hits())
	}
}

func TestLastErrorTruncated(t *testing.T) { // CJ-04: last_error ไม่เกิน 500 ตัวอักษร
	fake := setup(t)
	database.DBConn.Create(&models.Cronjob{Code: code, Enabled: true})
	fake.Set(func(c *testutil.FakeAskmelottoConfig) { c.Code, c.Msg = 430999, strings.Repeat("ก", 2000) })
	cronjob.RunOnce(context.Background(), code)
	c := row(t)
	if c.LastStatus != models.CronjobStatusFailed || len([]rune(c.LastError)) != 500 {
		t.Errorf("last_error ยาว %d ตัวอักษร (status %s)", len([]rune(c.LastError)), c.LastStatus)
	}
}

func TestRunOnceUnknownJob(t *testing.T) {
	setup(t)
	if cronjob.RunOnce(context.Background(), "no_such_job") {
		t.Error("job ที่ไม่มีในทะเบียนต้องคืน false")
	}
}

func TestStartStopsOnContextCancel(t *testing.T) { // กฎข้อ 27: goroutine หยุดตอน ctx ถูกยกเลิก (ไม่ค้าง ไม่ยิงต่อ)
	fake := setup(t)
	ctx, cancel := context.WithCancel(context.Background())
	cronjob.Start(ctx)
	waitStatus(t)
	cancel()
	hits := fake.Hits()
	time.Sleep(300 * time.Millisecond)
	if fake.Hits() != hits {
		t.Error("หลัง cancel ต้องไม่ยิงเพิ่ม")
	}
}
