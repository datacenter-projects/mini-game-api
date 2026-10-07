package redis

import (
	"context"
	"testing"
	"time"
)

func TestCurrencyRateLock(t *testing.T) { // CR-03
	mr := setupRedis(t)
	ctx := context.Background()

	ok, err := AcquireCurrencyRateLockRepository(ctx, "a", time.Minute)
	if err != nil || !ok {
		t.Fatalf("ครั้งแรกต้องได้ lock: %v %v", ok, err)
	}
	if ok, _ := AcquireCurrencyRateLockRepository(ctx, "b", time.Minute); ok {
		t.Fatal("instance ที่สองต้องไม่ได้ lock")
	}
	// lock ของ a หมดอายุ แล้ว b ได้ไป — a ปล่อย lock ต้องไม่ลบของ b
	mr.FastForward(2 * time.Minute)
	if ok, _ := AcquireCurrencyRateLockRepository(ctx, "b", time.Minute); !ok {
		t.Fatal("หลังหมดอายุ b ต้องได้ lock")
	}
	if err := ReleaseCurrencyRateLockRepository(ctx, "a"); err != nil {
		t.Fatal(err)
	}
	if v, _ := mr.Get(keyCurrencyRateLock); v != "b" {
		t.Fatalf("lock ของ b ถูกลบ: %q", v)
	}
	if err := ReleaseCurrencyRateLockRepository(ctx, "b"); err != nil {
		t.Fatal(err)
	}
	if mr.Exists(keyCurrencyRateLock) {
		t.Fatal("b ปล่อย lock แล้วต้องหาย")
	}
}

func TestCurrencyRatesSaveGet(t *testing.T) { // CR-08, CR-10
	setupRedis(t)
	ctx := context.Background()

	if r, err := GetCurrencyRatesRepository(ctx); err != nil || len(r) != 0 {
		t.Fatalf("ยังไม่เคย sync ต้องว่าง: %v %v", r, err)
	}
	at := time.Date(2026, 10, 7, 1, 2, 3, 0, time.UTC)
	if err := SaveCurrencyRatesRepository(ctx, map[string]int64{"THB": 3362000000, "KRW": 133790000000}, CurrencyRateMeta{Date: "2026-10-07", SyncedAt: at}); err != nil {
		t.Fatal(err)
	}
	// รอบถัดไปไม่มี KRW → KRW คงค่าเดิม
	if err := SaveCurrencyRatesRepository(ctx, map[string]int64{"THB": 3400000000}, CurrencyRateMeta{Date: "2026-10-08", SyncedAt: at.Add(time.Hour)}); err != nil {
		t.Fatal(err)
	}
	r, err := GetCurrencyRatesRepository(ctx)
	if err != nil || r["THB"] != 3400000000 || r["KRW"] != 133790000000 {
		t.Fatalf("เรทไม่ถูก: %v %v", r, err)
	}
	m, err := GetCurrencyRateMetaRepository(ctx)
	if err != nil || m.Date != "2026-10-08" || !m.SyncedAt.Equal(at.Add(time.Hour)) {
		t.Fatalf("meta ไม่ถูก: %+v %v", m, err)
	}
}
