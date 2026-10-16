//go:build integration

// integration test ของ currency_rate (sync + แปลงเป็นบาท) ตามตาราง test case ใน docs/modules/currency_rate.md หัวข้อ 9
// ใช้ askmelotto จำลอง (testutil.FakeAskmelotto) + Postgres / Redis จริงของ test env
package currencyrate_test

import (
	"context"
	"errors"
	"math"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	currencyRateCore "app/app/core/currency_rate"
	"app/app/models"
	currencyRateService "app/app/service/currency_rate"
	"app/pkg/apperr"
	"app/pkg/testutil"
	"app/platform/database"
	"app/platform/logger"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

var bkk = time.FixedZone("Asia/Bangkok", 7*3600)

func setup(t *testing.T) *testutil.FakeAskmelotto {
	t.Helper()
	testutil.Setup(t)
	return testutil.NewFakeAskmelotto(t)
}

func monthDate(y int, m time.Month) time.Time { return time.Date(y, m, 1, 0, 0, 0, 0, time.UTC) }

func ratesOf(t *testing.T, month time.Time) map[string]models.CurrencyRate {
	t.Helper()
	var rows []models.CurrencyRate
	if err := database.DBConn.Where("rate_month = ?", month).Find(&rows).Error; err != nil {
		t.Fatal(err)
	}
	out := map[string]models.CurrencyRate{}
	for _, r := range rows {
		out[r.Currency] = r
	}
	return out
}

func countAll(t *testing.T) int64 {
	t.Helper()
	var n int64
	database.DBConn.Model(&models.CurrencyRate{}).Count(&n)
	return n
}

func sync(t *testing.T, now time.Time, opts currencyRateService.SyncOptions) (currencyRateService.SyncResult, error) {
	t.Helper()
	return currencyRateService.SyncCurrencyRateService(context.Background(), now, opts)
}

func TestSyncRequestsFirstDayAndStoresAll(t *testing.T) { // CR-02 · CR-03 · CR-06 · CR-14
	fake := setup(t)
	now := time.Date(2026, 10, 16, 12, 0, 0, 0, bkk)
	res, err := sync(t, now, currencyRateService.SyncOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if got := fake.Dates(); len(got) != 1 || got[0] != "2026-10-01" { // วันที่ 1 เสมอ ไม่ใช่วันปัจจุบัน
		t.Fatalf("date ที่ขอ = %v want [2026-10-01]", got)
	}
	if res.Saved != 27 || len(res.Missing) != 0 || res.AlreadyComplete || res.Skipped {
		t.Fatalf("res = %+v", res)
	}
	rows := ratesOf(t, monthDate(2026, 10))
	if len(rows) != 27 {
		t.Fatalf("แถว = %d want 27", len(rows))
	}
	if r := rows["USD"]; r.USDTRate != 0.99 || r.THBRate != 33.95959596 || r.SyncedAt.IsZero() {
		t.Errorf("USD = %+v", r)
	}
	if r := rows["THB"]; r.USDTRate != 33.62 || r.THBRate != 1 {
		t.Errorf("THB = %+v", r)
	}
	if r := rows["USDT"]; r.USDTRate != 1 || r.THBRate != 33.62 {
		t.Errorf("USDT = %+v", r)
	}
	if r := rows["IDR"]; r.USDTRate != 17849.27 || math.Abs(r.THBRate-0.00188355) > 5e-9 { // 33.62 ÷ 17849.27 = 0.0018835546… → 0.00188355
		t.Errorf("IDR = %+v", r)
	}
}

func TestSyncMonthIsBangkokMonth(t *testing.T) { // CR-01 · CR-02 — 31 ต.ค. 17:30 UTC = 1 พ.ย. 00:30 ไทย
	fake := setup(t)
	if _, err := sync(t, time.Date(2026, 10, 31, 17, 30, 0, 0, time.UTC), currencyRateService.SyncOptions{}); err != nil {
		t.Fatal(err)
	}
	if got := fake.Dates(); len(got) != 1 || got[0] != "2026-11-01" {
		t.Fatalf("date ที่ขอ = %v want [2026-11-01]", got)
	}
	if len(ratesOf(t, monthDate(2026, 11))) != 27 || len(ratesOf(t, monthDate(2026, 10))) != 0 {
		t.Error("ต้องบันทึกเดือน พ.ย. เท่านั้น")
	}
}

func TestSyncCompleteMonthDoesNotCall(t *testing.T) { // CR-10 · CR-14
	fake := setup(t)
	now := time.Date(2026, 10, 16, 12, 0, 0, 0, bkk)
	if _, err := sync(t, now, currencyRateService.SyncOptions{}); err != nil {
		t.Fatal(err)
	}
	res, err := sync(t, now, currencyRateService.SyncOptions{})
	if err != nil || !res.AlreadyComplete {
		t.Fatalf("รอบสอง: %+v %v", res, err)
	}
	if fake.Hits() != 1 {
		t.Errorf("ครบแล้วต้องไม่ยิงอีก: hits = %d", fake.Hits())
	}
}

func TestSyncNeverOverwritesAndFillsMissing(t *testing.T) { // CR-04 · CR-05
	fake := setup(t)
	now := time.Date(2026, 10, 16, 12, 0, 0, 0, bkk)

	rates := testutil.DefaultAskmelottoRates()
	delete(rates, "KRW")
	fake.Set(func(c *testutil.FakeAskmelottoConfig) { c.Rates = rates })
	res, err := sync(t, now, currencyRateService.SyncOptions{})
	if !errors.Is(err, currencyRateService.ErrIncomplete) || strings.Join(res.Missing, ",") != "KRW" || res.Saved != 26 {
		t.Fatalf("ขาด KRW: res %+v err %v", res, err)
	}
	before := ratesOf(t, monthDate(2026, 10))
	if len(before) != 26 {
		t.Fatalf("ต้องบันทึก 26 สกุล: %d", len(before))
	}
	if _, has := before["KRW"]; has {
		t.Fatal("KRW ต้องไม่มี (ไม่ใช้เรทเดือนก่อน)")
	}

	// รอบถัดไป askmelotto ตอบค่าอื่น (THB เปลี่ยน · มี KRW) → เติมเฉพาะ KRW · ของเดิมห้ามเปลี่ยน
	newRates := testutil.DefaultAskmelottoRates()
	newRates["THB"] = "40"
	newRates["USD"] = "2"
	fake.Set(func(c *testutil.FakeAskmelottoConfig) { c.Rates = newRates })
	res, err = sync(t, now, currencyRateService.SyncOptions{})
	if err != nil || res.Saved != 1 {
		t.Fatalf("รอบเติม: res %+v err %v", res, err)
	}
	after := ratesOf(t, monthDate(2026, 10))
	if len(after) != 27 {
		t.Fatalf("แถว = %d want 27", len(after))
	}
	for c, old := range before {
		if got := after[c]; got.USDTRate != old.USDTRate || got.THBRate != old.THBRate || !got.SyncedAt.Equal(old.SyncedAt) {
			t.Errorf("%s ถูกแก้ทั้งที่ห้ามทับ: %+v → %+v", c, old, got)
		}
	}
	// KRW ใช้ THB ของรอบที่เติม (40) — คิด thb_rate จาก response ของรอบนั้น ไม่ผสมกับ THB เดิมของรอบก่อน
	if k := after["KRW"]; math.Abs(k.THBRate-40.0/1337.9) > 5e-9 {
		t.Errorf("KRW thb_rate = %v", k.THBRate)
	}
}

func TestSyncNoTHBSavesNothing(t *testing.T) { // CR-04
	fake := setup(t)
	rates := testutil.DefaultAskmelottoRates()
	delete(rates, "THB")
	fake.Set(func(c *testutil.FakeAskmelottoConfig) { c.Rates = rates })
	_, err := sync(t, time.Date(2026, 10, 16, 12, 0, 0, 0, bkk), currencyRateService.SyncOptions{})
	if err == nil || countAll(t) != 0 {
		t.Fatalf("ไม่มี THB ต้องไม่บันทึกอะไรเลย: err %v rows %d", err, countAll(t))
	}

	rates["THB"] = `"33.62"` // THB เป็น string = ค่าผิด = ไม่มีฐานคิดเหมือนกัน
	fake.Set(func(c *testutil.FakeAskmelottoConfig) { c.Rates = rates })
	if _, err := sync(t, time.Date(2026, 10, 16, 12, 0, 0, 0, bkk), currencyRateService.SyncOptions{}); err == nil || countAll(t) != 0 {
		t.Fatalf("THB ผิดรูป ต้องไม่บันทึกอะไรเลย: err %v rows %d", err, countAll(t))
	}
}

func TestSyncResponseProblemsSaveNothing(t *testing.T) { // CR-15
	fake := setup(t)
	now := time.Date(2026, 10, 16, 12, 0, 0, 0, bkk)
	tests := []struct {
		name string
		set  func(c *testutil.FakeAskmelottoConfig)
		want string
	}{
		{"430503 ยังไม่มีเรท", func(c *testutil.FakeAskmelottoConfig) { c.Code, c.Msg = 430503, "no rate" }, "430503"},
		{"430502 รูปแบบวันที่ผิด", func(c *testutil.FakeAskmelottoConfig) { c.Code = 430502 }, "430502"},
		{"date ไม่ตรงที่ขอ", func(c *testutil.FakeAskmelottoConfig) { c.Date = "2026-10-16" }, "does not match"},
		{"base ไม่ใช่ USDT", func(c *testutil.FakeAskmelottoConfig) { c.Base = "USD" }, "not USDT"},
		{"HTTP 500", func(c *testutil.FakeAskmelottoConfig) { c.Status = 500 }, "500"},
		{"body ไม่ใช่ JSON", func(c *testutil.FakeAskmelottoConfig) { c.RawBody = "<html>oops</html>" }, "decode"},
		{"ไม่มี json_rate", func(c *testutil.FakeAskmelottoConfig) {
			c.RawBody = `{"code":200,"data":{"date":"2026-10-01","base_currency":"USDT"}}`
		}, "json_rate"},
	}
	for _, tt := range tests {
		fake.Reset()
		fake.Set(tt.set)
		_, err := sync(t, now, currencyRateService.SyncOptions{})
		if err == nil || !strings.Contains(err.Error(), tt.want) {
			t.Errorf("%s: err = %v want มี %q", tt.name, err, tt.want)
		}
		if n := countAll(t); n != 0 {
			t.Errorf("%s: บันทึก %d แถว ต้องไม่บันทึกอะไร", tt.name, n)
		}
	}
}

func TestSyncRejectsBadValues(t *testing.T) { // CR-15 — สกุลที่ค่าผิดไม่บันทึก · สกุลอื่นบันทึกตามปกติ
	fake := setup(t)
	rates := testutil.DefaultAskmelottoRates()
	rates["USD"] = `"0.99"`         // string
	rates["EUR"] = `0`              // ศูนย์
	rates["JPY"] = `-158.15`        // ติดลบ
	rates["KRW"] = `1337.123456789` // ทศนิยม 9
	rates["XYZ"] = `5`              // นอกรายการ — ไม่สนใจ
	fake.Set(func(c *testutil.FakeAskmelottoConfig) { c.Rates = rates })

	res, err := sync(t, time.Date(2026, 10, 16, 12, 0, 0, 0, bkk), currencyRateService.SyncOptions{})
	if !errors.Is(err, currencyRateService.ErrIncomplete) {
		t.Fatalf("ไม่ครบต้อง ErrIncomplete: %v", err)
	}
	if strings.Join(res.Rejected, ",") != "EUR,JPY,KRW,USD" || strings.Join(res.Missing, ",") != "EUR,JPY,KRW,USD" || res.Saved != 23 {
		t.Errorf("res = %+v", res)
	}
	rows := ratesOf(t, monthDate(2026, 10))
	for _, c := range []string{"USD", "EUR", "JPY", "KRW", "XYZ"} {
		if _, has := rows[c]; has {
			t.Errorf("%s ต้องไม่ถูกบันทึก", c)
		}
	}
	if len(rows) != 23 {
		t.Errorf("แถว = %d want 23", len(rows))
	}
}

func TestSyncForceOverwritesWholeMonth(t *testing.T) { // CR-05 · CR-17
	fake := setup(t)
	now := time.Date(2026, 10, 16, 12, 0, 0, 0, bkk)
	if _, err := sync(t, now, currencyRateService.SyncOptions{}); err != nil {
		t.Fatal(err)
	}
	old := ratesOf(t, monthDate(2026, 10))

	newRates := testutil.DefaultAskmelottoRates()
	newRates["THB"], newRates["USD"] = "35", "0.5"
	fake.Set(func(c *testutil.FakeAskmelottoConfig) { c.Rates = newRates })
	month := monthDate(2026, 10)

	// ไม่ใส่ Force = ไม่ยิงและไม่ทับ (เดือนครบแล้ว)
	if res, err := sync(t, now, currencyRateService.SyncOptions{Month: month}); err != nil || !res.AlreadyComplete {
		t.Fatalf("ไม่ใส่ force: %+v %v", res, err)
	}
	if got := ratesOf(t, month)["USD"]; got.THBRate != old["USD"].THBRate {
		t.Fatal("ไม่ใส่ force ห้ามทับ")
	}
	// Force = เขียนทับทั้งเดือน + synced_at ใหม่
	time.Sleep(5 * time.Millisecond)
	res, err := sync(t, now, currencyRateService.SyncOptions{Month: month, Force: true})
	if err != nil || res.Saved != 27 {
		t.Fatalf("force: %+v %v", res, err)
	}
	got := ratesOf(t, month)
	if got["USD"].USDTRate != 0.5 || got["USD"].THBRate != 70 || got["THB"].USDTRate != 35 || !got["USD"].SyncedAt.After(old["USD"].SyncedAt) {
		t.Errorf("USD หลัง force = %+v", got["USD"])
	}
	if len(got) != 27 {
		t.Errorf("แถว = %d", len(got))
	}
}

func TestSyncSpecificMonth(t *testing.T) { // CR-17: -month ขอด้วยวันที่ 1 ของเดือนนั้น ไม่ผูกกับ now
	fake := setup(t)
	now := time.Date(2026, 10, 16, 12, 0, 0, 0, bkk)
	if _, err := sync(t, now, currencyRateService.SyncOptions{Month: monthDate(2026, 9)}); err != nil {
		t.Fatal(err)
	}
	if got := fake.Dates(); len(got) != 1 || got[0] != "2026-09-01" {
		t.Fatalf("date = %v", got)
	}
	if len(ratesOf(t, monthDate(2026, 9))) != 27 || len(ratesOf(t, monthDate(2026, 10))) != 0 {
		t.Error("ต้องบันทึกเดือน ก.ย. เท่านั้น")
	}
}

func TestSyncLockHeldSkips(t *testing.T) { // CR-13
	fake := setup(t)
	if err := database.DBRedis.Set(context.Background(), "currency_rate:sync:lock", "other-pod", time.Minute).Err(); err != nil {
		t.Fatal(err)
	}
	res, err := sync(t, time.Date(2026, 10, 16, 12, 0, 0, 0, bkk), currencyRateService.SyncOptions{})
	if err != nil || !res.Skipped || fake.Hits() != 0 || countAll(t) != 0 {
		t.Fatalf("res %+v err %v hits %d rows %d", res, err, fake.Hits(), countAll(t))
	}
	if v, _ := database.DBRedis.Get(context.Background(), "currency_rate:sync:lock").Result(); v != "other-pod" {
		t.Errorf("ห้ามลบ lock ของ pod อื่น: %q", v)
	}
}

func TestSyncReleasesLock(t *testing.T) { // CR-13 — ปล่อย lock หลังจบรอบ (สำเร็จและล้ม)
	fake := setup(t)
	now := time.Date(2026, 10, 16, 12, 0, 0, 0, bkk)
	fake.Set(func(c *testutil.FakeAskmelottoConfig) { c.Status = 500 })
	_, _ = sync(t, now, currencyRateService.SyncOptions{})
	if n, _ := database.DBRedis.Exists(context.Background(), "currency_rate:sync:lock").Result(); n != 0 {
		t.Error("รอบที่ล้มต้องปล่อย lock")
	}
	fake.Reset()
	if _, err := sync(t, now, currencyRateService.SyncOptions{}); err != nil {
		t.Fatal(err)
	}
	if n, _ := database.DBRedis.Exists(context.Background(), "currency_rate:sync:lock").Result(); n != 0 {
		t.Error("รอบที่สำเร็จต้องปล่อย lock")
	}
}

// ---- ConvertToTHBService ----

type queryCounter struct{ n atomic.Int32 }

func (q *queryCounter) LogMode(gormlogger.LogLevel) gormlogger.Interface { return q }
func (q *queryCounter) Info(context.Context, string, ...any)             {}
func (q *queryCounter) Warn(context.Context, string, ...any)             {}
func (q *queryCounter) Error(context.Context, string, ...any)            {}
func (q *queryCounter) Trace(context.Context, time.Time, func() (string, int64), error) {
	q.n.Add(1)
}

func seedRate(t *testing.T, month time.Time, currency string, usdt, thb float64) {
	t.Helper()
	if err := database.DBConn.Create(&models.CurrencyRate{RateMonth: month, Currency: currency, USDTRate: usdt, THBRate: thb, SyncedAt: time.Now()}).Error; err != nil {
		t.Fatal(err)
	}
}

func TestConvertToTHB(t *testing.T) { // CR-20 – CR-22
	setup(t)
	seedRate(t, monthDate(2026, 10), "USD", 0.99, 33.95959596)
	seedRate(t, monthDate(2026, 11), "USD", 1, 34)

	at := time.Date(2026, 10, 16, 12, 0, 0, 0, bkk)
	got, err := currencyRateService.ConvertToTHBService(context.Background(), database.DBConn, 100, "USD", at)
	if err != nil {
		t.Fatal(err)
	}
	if got.AmountTHB != 3395.9596 || got.THBRate != 33.95959596 || !got.RateMonth.Equal(monthDate(2026, 10)) {
		t.Errorf("100 USD = %+v", got)
	}

	// เดือนตาม at เวลาไทย: 31 ต.ค. 23:59 ใช้เรท ต.ค. · 1 พ.ย. 00:01 ใช้เรท พ.ย.
	oct, _ := currencyRateService.ConvertToTHBService(context.Background(), database.DBConn, 100, "USD", time.Date(2026, 10, 31, 23, 59, 0, 0, bkk))
	nov, _ := currencyRateService.ConvertToTHBService(context.Background(), database.DBConn, 100, "USD", time.Date(2026, 11, 1, 0, 1, 0, 0, bkk))
	if oct.THBRate != 33.95959596 || nov.THBRate != 34 || nov.AmountTHB != 3400 || !nov.RateMonth.Equal(monthDate(2026, 11)) {
		t.Errorf("ข้ามเดือน: oct %+v nov %+v", oct, nov)
	}
	// 31 ต.ค. 17:30 UTC = 1 พ.ย. 00:30 ไทย → เรท พ.ย.
	utc, _ := currencyRateService.ConvertToTHBService(context.Background(), database.DBConn, 100, "USD", time.Date(2026, 10, 31, 17, 30, 0, 0, time.UTC))
	if utc.THBRate != 34 {
		t.Errorf("ขอบเขต UTC → ไทย: %+v", utc)
	}
}

func TestConvertTHBSkipsDB(t *testing.T) { // CR-23 · CR-26
	setup(t)
	seedRate(t, monthDate(2026, 10), "USD", 0.99, 33.95959596)
	at := time.Date(2026, 10, 16, 12, 0, 0, 0, bkk)

	thb := &queryCounter{}
	got, err := currencyRateService.ConvertToTHBService(context.Background(), database.DBConn.Session(&gorm.Session{Logger: thb}), 250.5, "THB", at)
	if err != nil || got.AmountTHB != 250.5 || got.THBRate != 1 || thb.n.Load() != 0 {
		t.Errorf("THB: %+v err %v queries %d (ต้อง 0)", got, err, thb.n.Load())
	}
	usd := &queryCounter{}
	if _, err := currencyRateService.ConvertToTHBService(context.Background(), database.DBConn.Session(&gorm.Session{Logger: usd}), 10, "USD", at); err != nil || usd.n.Load() != 1 {
		t.Errorf("USD: err %v queries %d (ต้อง 1 แถวด้วย PK)", err, usd.n.Load())
	}
}

func TestConvertUsesCallerTx(t *testing.T) { // CR-26 — ใช้ tx ของผู้เรียก ไม่เปิดเอง
	setup(t)
	at := time.Date(2026, 10, 16, 12, 0, 0, 0, bkk)
	err := database.DBConn.Transaction(func(tx *gorm.DB) error {
		seedRateTx := models.CurrencyRate{RateMonth: monthDate(2026, 10), Currency: "USD", USDTRate: 1, THBRate: 30, SyncedAt: time.Now()}
		if err := tx.Create(&seedRateTx).Error; err != nil { // เห็นเฉพาะใน tx นี้ (ยังไม่ commit)
			return err
		}
		got, err := currencyRateService.ConvertToTHBService(context.Background(), tx, 2, "USD", at)
		if err != nil || got.AmountTHB != 60 {
			t.Errorf("ใน tx: %+v %v", got, err)
		}
		return errors.New("rollback")
	})
	if err == nil || err.Error() != "rollback" {
		t.Fatal(err)
	}
	if _, err := currencyRateService.ConvertToTHBService(context.Background(), database.DBConn, 2, "USD", at); !errors.Is(err, apperr.ErrCurrencyRateUnavailable) {
		t.Errorf("rollback แล้วต้องไม่มีเรท: %v", err)
	}
}

func TestConvertUnavailable(t *testing.T) { // CR-24 · CR-25 — ไม่มีเรทเดือนนั้น = error · ไม่ใช้เรทเดือนก่อน
	setup(t)
	seedRate(t, monthDate(2026, 10), "USD", 0.99, 33.95959596) // มีแต่เดือน ต.ค.
	ctx := context.Background()

	_, err := currencyRateService.ConvertToTHBService(ctx, database.DBConn, 100, "USD", time.Date(2026, 11, 1, 0, 10, 0, 0, bkk))
	if !errors.Is(err, apperr.ErrCurrencyRateUnavailable) {
		t.Errorf("เดือน พ.ย. ยังไม่มีเรท ต้อง ErrCurrencyRateUnavailable (ไม่ใช้เรท ต.ค.): %v", err)
	}
	at := time.Date(2026, 10, 16, 12, 0, 0, 0, bkk)
	for _, cur := range []string{"VND", "XYZ", "usd", ""} { // ไม่มีแถว · สกุลที่ไม่รู้จัก · ตัวพิมพ์เล็ก (ไม่ normalize) · ว่าง
		if _, err := currencyRateService.ConvertToTHBService(ctx, database.DBConn, 100, cur, at); !errors.Is(err, apperr.ErrCurrencyRateUnavailable) {
			t.Errorf("สกุล %q: %v", cur, err)
		}
	}
	if apperr.ErrCurrencyRateUnavailable.Code != 409401 || apperr.ErrCurrencyRateUnavailable.HTTPStatus != 200 {
		t.Errorf("code = %d HTTP %d", apperr.ErrCurrencyRateUnavailable.Code, apperr.ErrCurrencyRateUnavailable.HTTPStatus)
	}
}

func TestConvertInvalidAmountIsInternalError(t *testing.T) { // CR-27 — amount ผิดไม่ใช่ "เรทหาย"
	setup(t)
	seedRate(t, monthDate(2026, 10), "USD", 0.99, 33.95959596)
	at := time.Date(2026, 10, 16, 12, 0, 0, 0, bkk)
	for _, amount := range []float64{math.NaN(), math.Inf(1), -1} {
		for _, cur := range []string{"USD", "THB", "VND"} {
			_, err := currencyRateService.ConvertToTHBService(context.Background(), database.DBConn, amount, cur, at)
			if err == nil || errors.Is(err, apperr.ErrCurrencyRateUnavailable) || !errors.Is(err, apperr.ErrInternal) {
				t.Errorf("amount %v %s: err = %v ต้องเป็น ErrInternal ไม่ใช่ ErrCurrencyRateUnavailable", amount, cur, err)
			}
		}
	}
}

func TestConvertRoundsOnceHalfUp(t *testing.T) { // CR-22
	setup(t)
	seedRate(t, monthDate(2026, 10), "USD", 1, 0.00005)
	at := time.Date(2026, 10, 16, 12, 0, 0, 0, bkk)
	got, err := currencyRateService.ConvertToTHBService(context.Background(), database.DBConn, 1, "USD", at)
	if err != nil || got.AmountTHB != 0.0001 { // 0.00005 → ปัดครึ่งขึ้น
		t.Errorf("ครึ่งขึ้น: %+v %v", got, err)
	}
	if currencyRateCore.IsSupported("USD") != true {
		t.Fatal("sanity")
	}
}

// ---- ระดับ log (CR-16) ----

func TestSyncLogLevelByTime(t *testing.T) { // ก่อน 03:00 ของวันที่ 1 = info · หลังจากนั้นยังไม่ครบ = warn
	fake := setup(t)
	fake.Set(func(c *testutil.FakeAskmelottoConfig) { c.Code, c.Msg = 430503, "no rate yet" })

	origLogger, origSugar := logger.Logger, logger.SugarLogger
	core, logs := observer.New(zapcore.DebugLevel)
	logger.Logger = zap.New(core)
	logger.SugarLogger = logger.Logger.Sugar()
	t.Cleanup(func() { logger.Logger, logger.SugarLogger = origLogger, origSugar })

	tests := []struct {
		name string
		now  time.Time
		want zapcore.Level
	}{
		{"1 พ.ย. 00:10 ไทย", time.Date(2026, 11, 1, 0, 10, 0, 0, bkk), zapcore.InfoLevel},
		{"1 พ.ย. 02:59 ไทย", time.Date(2026, 11, 1, 2, 59, 0, 0, bkk), zapcore.InfoLevel},
		{"1 พ.ย. 03:00 ไทย", time.Date(2026, 11, 1, 3, 0, 0, 0, bkk), zapcore.WarnLevel},
		{"1 พ.ย. 12:00 ไทย", time.Date(2026, 11, 1, 12, 0, 0, 0, bkk), zapcore.WarnLevel},
		{"2 พ.ย. 01:00 ไทย", time.Date(2026, 11, 2, 1, 0, 0, 0, bkk), zapcore.WarnLevel},
		{"กลางเดือน", time.Date(2026, 10, 16, 1, 0, 0, 0, bkk), zapcore.WarnLevel},
	}
	for _, tt := range tests {
		logs.TakeAll()
		if _, err := sync(t, tt.now, currencyRateService.SyncOptions{}); err == nil {
			t.Fatalf("%s: 430503 ต้อง error", tt.name)
		}
		var found bool
		for _, e := range logs.All() {
			if strings.Contains(e.Message, "askmelotto returned an error code") {
				found = true
				if e.Level != tt.want {
					t.Errorf("%s: level = %v want %v", tt.name, e.Level, tt.want)
				}
				m := e.ContextMap()
				if m["code"] != int64(430503) || m["msg"] != "no rate yet" { // log มี code / msg ของ askmelotto
					t.Errorf("%s: ฟิลด์ log = %v", tt.name, m)
				}
			}
		}
		if !found {
			t.Errorf("%s: ไม่พบ log ของ error code", tt.name)
		}
	}
}

func TestSyncMissingCurrencyLogsWarnWithNames(t *testing.T) { // CR-15 / CR-16 — log บอกสกุลที่ขาด / ผิด
	fake := setup(t)
	rates := testutil.DefaultAskmelottoRates()
	rates["KRW"] = `0`
	delete(rates, "VND")
	fake.Set(func(c *testutil.FakeAskmelottoConfig) { c.Rates = rates })

	origLogger, origSugar := logger.Logger, logger.SugarLogger
	core, logs := observer.New(zapcore.DebugLevel)
	logger.Logger = zap.New(core)
	logger.SugarLogger = logger.Logger.Sugar()
	t.Cleanup(func() { logger.Logger, logger.SugarLogger = origLogger, origSugar })

	if _, err := sync(t, time.Date(2026, 10, 16, 12, 0, 0, 0, bkk), currencyRateService.SyncOptions{}); !errors.Is(err, currencyRateService.ErrIncomplete) {
		t.Fatal(err)
	}
	var rejectedLog, missingLog bool
	for _, e := range logs.All() {
		switch {
		case strings.Contains(e.Message, "value invalid") && e.Level == zapcore.WarnLevel:
			rejectedLog = strings.Contains(strings.Join(toStrings(e.ContextMap()["currencies"]), ","), "KRW")
		case strings.Contains(e.Message, "incomplete") && e.Level == zapcore.WarnLevel:
			missingLog = strings.Contains(strings.Join(toStrings(e.ContextMap()["missing"]), ","), "VND")
		}
	}
	if !rejectedLog || !missingLog {
		t.Errorf("ต้องมี warn บอกสกุลที่ค่าผิด (KRW) และสกุลที่ขาด (VND): rejected %v missing %v", rejectedLog, missingLog)
	}
}

func toStrings(v any) []string {
	switch s := v.(type) {
	case []any:
		out := make([]string, 0, len(s))
		for _, x := range s {
			if str, ok := x.(string); ok {
				out = append(out, str)
			}
		}
		return out
	case []string:
		return s
	}
	return nil
}

// RateMonth ที่คืนต้องเป็นค่าเดียวกับ rate_month ที่ repo / model ใช้ (เที่ยงคืน UTC) — bet เก็บไว้กับรายการเดิมพันแล้วเอาไปเทียบ / join ได้ตรง (ตรวจรอบ 2 ของ lead)
func TestConvertRateMonthMatchesStoredRow(t *testing.T) {
	setup(t)
	seedRate(t, monthDate(2026, 10), "USD", 0.99, 33.95959596)

	at := time.Date(2026, 10, 1, 0, 30, 0, 0, bkk) // 30 ก.ย. 17:30 UTC — เวลาไทยเป็นเดือน ต.ค.
	for _, cur := range []string{"USD", "THB"} {
		got, err := currencyRateService.ConvertToTHBService(context.Background(), database.DBConn, 1, cur, at)
		if err != nil {
			t.Fatal(err)
		}
		if got.RateMonth.Location() != time.UTC || !got.RateMonth.Equal(time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)) {
			t.Errorf("%s: RateMonth = %v (%v) want 2026-10-01 00:00 UTC", cur, got.RateMonth, got.RateMonth.Location())
		}
	}
	var row models.CurrencyRate
	if err := database.DBConn.Where("currency = ?", "USD").First(&row).Error; err != nil {
		t.Fatal(err)
	}
	got, _ := currencyRateService.ConvertToTHBService(context.Background(), database.DBConn, 1, "USD", at)
	if !got.RateMonth.Equal(row.RateMonth) {
		t.Errorf("RateMonth %v ไม่ตรงกับแถวที่เก็บ %v", got.RateMonth, row.RateMonth)
	}
}
