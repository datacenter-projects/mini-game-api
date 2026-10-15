//go:build integration

package main

import (
	"strings"
	"testing"
	"time"

	"app/app/models"
	"app/pkg/testutil"
	"app/platform/database"
)

// คำสั่งมือใช้ service เดียวกับ job (docs/modules/currency_rate.md CR-17) — test ผ่าน run() ตรงๆ
// run() อ่าน env เอง (configs.Load) จึงชี้ askmelotto จำลองด้วย env ASKMELOTTO_API_BASE_URL

func setup(t *testing.T) *testutil.FakeAskmelotto {
	t.Helper()
	testutil.Setup(t) // ต่อ DB / Redis · env ของ test env ถูกโหลดเข้า process แล้ว
	fake := testutil.NewFakeAskmelotto(t)
	t.Setenv("ASKMELOTTO_API_BASE_URL", fake.URL)
	return fake
}

func ratesOf(t *testing.T, month string) map[string]models.CurrencyRate {
	t.Helper()
	d, err := time.Parse("2006-01", month)
	if err != nil {
		t.Fatal(err)
	}
	var rows []models.CurrencyRate
	if err := database.DBConn.Where("rate_month = ?", time.Date(d.Year(), d.Month(), 1, 0, 0, 0, 0, time.UTC)).Find(&rows).Error; err != nil {
		t.Fatal(err)
	}
	out := map[string]models.CurrencyRate{}
	for _, r := range rows {
		out[r.Currency] = r
	}
	return out
}

func TestRunRejectsBadMonth(t *testing.T) {
	fake := setup(t)
	for _, bad := range []string{"2026", "2026-13", "2026-10-01", "oct", "26-10"} {
		if err := run(bad, false, false); err == nil || !strings.Contains(err.Error(), "YYYY-MM") {
			t.Errorf("-month %q ต้อง error: %v", bad, err)
		}
	}
	if fake.Hits() != 0 {
		t.Errorf("month ผิดต้องไม่ยิง: %d", fake.Hits())
	}
}

func TestRunFillsOnlyMissing(t *testing.T) { // ไม่ใส่ -force = เติมเฉพาะสกุลที่ขาด ไม่ทับของเดิม
	fake := setup(t)

	rates := testutil.DefaultAskmelottoRates()
	delete(rates, "KRW")
	fake.Set(func(c *testutil.FakeAskmelottoConfig) { c.Rates = rates })
	if err := run("2026-09", false, false); err == nil || !strings.Contains(err.Error(), "KRW") { // ไม่ครบ = error บอกสกุลที่ขาด
		t.Fatalf("ขาด KRW ต้อง error ที่บอก KRW: %v", err)
	}
	if got := fake.Dates(); len(got) != 1 || got[0] != "2026-09-01" { // ขอด้วยวันที่ 1 ของเดือนที่ระบุ
		t.Fatalf("date = %v", got)
	}
	before := ratesOf(t, "2026-09")
	if len(before) != 26 {
		t.Fatalf("บันทึก %d want 26", len(before))
	}

	newRates := testutil.DefaultAskmelottoRates()
	newRates["THB"], newRates["USD"] = "40", "2"
	fake.Set(func(c *testutil.FakeAskmelottoConfig) { c.Rates = newRates })
	if err := run("2026-09", false, false); err != nil {
		t.Fatal(err)
	}
	after := ratesOf(t, "2026-09")
	if len(after) != 27 {
		t.Fatalf("แถว = %d want 27", len(after))
	}
	for c, old := range before {
		if got := after[c]; got.THBRate != old.THBRate || got.USDTRate != old.USDTRate || !got.SyncedAt.Equal(old.SyncedAt) {
			t.Errorf("%s ถูกทับทั้งที่ไม่ได้ใส่ -force: %+v → %+v", c, old, got)
		}
	}

	hits := fake.Hits()
	if err := run("2026-09", false, false); err != nil || fake.Hits() != hits { // ครบแล้ว = ไม่ยิง ไม่ error
		t.Errorf("เดือนครบแล้วต้องไม่ยิง: err %v hits %d→%d", err, hits, fake.Hits())
	}
}

func TestRunForceOverwritesWholeMonth(t *testing.T) { // -force = เขียนทับทั้งเดือน
	fake := setup(t)
	if err := run("2026-10", false, false); err != nil {
		t.Fatal(err)
	}
	old := ratesOf(t, "2026-10")

	newRates := testutil.DefaultAskmelottoRates()
	newRates["THB"], newRates["USD"] = "35", "0.5"
	fake.Set(func(c *testutil.FakeAskmelottoConfig) { c.Rates = newRates })
	time.Sleep(5 * time.Millisecond)
	if err := run("2026-10", true, false); err != nil { // APP_ENV=local ไม่ต้อง -yes
		t.Fatal(err)
	}
	got := ratesOf(t, "2026-10")
	if len(got) != 27 || got["USD"].THBRate != 70 || got["THB"].USDTRate != 35 || !got["USD"].SyncedAt.After(old["USD"].SyncedAt) {
		t.Errorf("USD หลัง -force = %+v (แถว %d)", got["USD"], len(got))
	}
}

func TestRunForceOnSharedEnvNeedsYes(t *testing.T) { // -force บน env ร่วมต้องใส่ -yes ด้วย
	fake := setup(t)
	t.Setenv("APP_ENV", "dev")
	err := run("2026-10", true, false)
	if err == nil || !strings.Contains(err.Error(), "-yes") {
		t.Fatalf("ต้อง error ที่ขอ -yes: %v", err)
	}
	if fake.Hits() != 0 || len(ratesOf(t, "2026-10")) != 0 {
		t.Errorf("ห้ามทำอะไรก่อนได้ -yes: hits %d", fake.Hits())
	}
	if err := run("2026-10", true, true); err != nil || len(ratesOf(t, "2026-10")) != 27 {
		t.Errorf("ใส่ -yes แล้วต้องทำงาน: %v", err)
	}
	// ไม่ใส่ -force บน env ร่วมไม่ต้อง -yes (เติมเฉพาะที่ขาด ปลอดภัย)
	if err := run("2026-09", false, false); err != nil {
		t.Errorf("เติมเฉพาะที่ขาดบน env ร่วมไม่ต้อง -yes: %v", err)
	}
}

func TestRunDefaultsToCurrentThaiMonth(t *testing.T) { // ไม่ใส่ -month = เดือนปัจจุบัน (เวลาไทย)
	fake := setup(t)
	if err := run("", false, false); err != nil {
		t.Fatal(err)
	}
	want := time.Now().In(time.FixedZone("Asia/Bangkok", 7*3600)).Format("2006-01") + "-01"
	if got := fake.Dates(); len(got) != 1 || got[0] != want {
		t.Errorf("date = %v want [%s]", got, want)
	}
}
