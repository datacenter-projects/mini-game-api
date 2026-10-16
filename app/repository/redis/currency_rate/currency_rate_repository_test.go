package currencyrate

import (
	"context"
	"testing"
	"time"
)

func TestCurrencyRateLock(t *testing.T) { // rule: CR-13
	mr := setupRedis(t)
	ctx := context.Background()

	ok, err := AcquireCurrencyRateLockRepository(ctx, "a", time.Minute)
	if err != nil || !ok {
		t.Fatalf("ครั้งแรกต้องได้ lock: %v %v", ok, err)
	}
	if ok, _ := AcquireCurrencyRateLockRepository(ctx, "b", time.Minute); ok {
		t.Fatal("pod ที่สองต้องไม่ได้ lock")
	}
	// lock ของ a หมดอายุ แล้ว b ได้ไป — a ปล่อย lock ต้องไม่ลบของ b
	mr.FastForward(2 * time.Minute)
	if ok, _ := AcquireCurrencyRateLockRepository(ctx, "b", time.Minute); !ok {
		t.Fatal("หลังหมดอายุ b ต้องได้ lock")
	}
	if err := ReleaseCurrencyRateLockRepository(ctx, "a"); err != nil {
		t.Fatal(err)
	}
	if v, _ := mr.Get(keyLock()); v != "b" {
		t.Fatalf("a ปล่อย lock ต้องไม่ลบของ b: %q", v)
	}
	if err := ReleaseCurrencyRateLockRepository(ctx, "b"); err != nil {
		t.Fatal(err)
	}
	if mr.Exists(keyLock()) {
		t.Fatal("b ปล่อย lock ของตัวเองต้องลบจริง")
	}
	if ok, _ := AcquireCurrencyRateLockRepository(ctx, "c", time.Minute); !ok { // ปล่อยแล้วคนอื่นเอาได้ทันที
		t.Fatal("ปล่อยแล้วต้องมีคนได้ lock")
	}
}

func TestLockHasTTL(t *testing.T) { // กฎ redis/doc.go: key ต้องมี TTL
	mr := setupRedis(t)
	if ok, _ := AcquireCurrencyRateLockRepository(context.Background(), "a", 5*time.Minute); !ok {
		t.Fatal("ต้องได้ lock")
	}
	if ttl := mr.TTL(keyLock()); ttl <= 0 || ttl > 5*time.Minute {
		t.Errorf("ttl = %v", ttl)
	}
}
