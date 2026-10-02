package redis

import (
	"context"
	"errors"
	"testing"
	"time"

	"app/pkg/apperr"
	"app/platform/database"

	"github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"
)

func setupRedis(t *testing.T) *miniredis.Miniredis {
	t.Helper()
	mr := miniredis.RunT(t)
	database.DBRedis = goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = database.DBRedis.Close(); database.DBRedis = nil })
	return mr
}

func newSession(agentID uint) BOSession {
	now := time.Now()
	return BOSession{AgentID: agentID, IP: "1.1.1.1", CreatedAt: now, ExpiresAt: now.Add(12 * time.Hour)}
}

func TestBOSessionCreateAndGet(t *testing.T) {
	setupRedis(t)
	ctx := context.Background()

	if err := CreateBOSessionRepository(ctx, "s1", newSession(7), time.Hour); err != nil {
		t.Fatal(err)
	}
	s, err := GetBOSessionRepository(ctx, "s1", 7)
	if err != nil || s.AgentID != 7 || s.IP != "1.1.1.1" {
		t.Fatalf("got %+v, %v", s, err)
	}
	if _, err := GetBOSessionRepository(ctx, "s1", 8); !errors.Is(err, apperr.ErrNotFound) {
		t.Fatalf("session ของ agent อื่นต้องไม่ผ่าน: %v", err)
	}
	if _, err := GetBOSessionRepository(ctx, "nope", 7); !errors.Is(err, apperr.ErrNotFound) {
		t.Fatalf("session ที่ไม่มีต้องได้ ErrNotFound: %v", err)
	}
}

func TestBOSessionNewLoginKicksOld(t *testing.T) { // AUTH-06
	mr := setupRedis(t)
	ctx := context.Background()

	_ = CreateBOSessionRepository(ctx, "old", newSession(7), time.Hour)
	_ = CreateBOSessionRepository(ctx, "new", newSession(7), time.Hour)

	if _, err := GetBOSessionRepository(ctx, "old", 7); !errors.Is(err, apperr.ErrNotFound) {
		t.Fatalf("session เก่าต้องใช้ไม่ได้: %v", err)
	}
	if mr.Exists(keyBOSession("old")) {
		t.Fatal("session เก่าต้องถูกลบออกจาก Redis")
	}
	if _, err := GetBOSessionRepository(ctx, "new", 7); err != nil {
		t.Fatalf("session ใหม่ต้องใช้ได้: %v", err)
	}
	if _, err := GetBOSessionRepository(ctx, "x", 8); !errors.Is(err, apperr.ErrNotFound) {
		t.Fatal("agent อื่นต้องไม่กระทบ")
	}
}

func TestBOSessionIdleExpiryAndTouch(t *testing.T) { // AUTH-07, AUTH-08
	mr := setupRedis(t)
	ctx := context.Background()

	_ = CreateBOSessionRepository(ctx, "s1", newSession(7), 10*time.Minute)
	mr.FastForward(9 * time.Minute)
	if err := TouchBOSessionRepository(ctx, "s1", 10*time.Minute); err != nil {
		t.Fatal(err)
	}
	mr.FastForward(9 * time.Minute) // รวม 18 นาที แต่ถูกต่ออายุแล้ว
	if _, err := GetBOSessionRepository(ctx, "s1", 7); err != nil {
		t.Fatalf("session ที่ถูกต่ออายุต้องยังอยู่: %v", err)
	}
	mr.FastForward(11 * time.Minute)
	if _, err := GetBOSessionRepository(ctx, "s1", 7); !errors.Is(err, apperr.ErrNotFound) {
		t.Fatalf("ไม่มีการใช้งานเกิน idle ต้องหลุด: %v", err)
	}
	if err := TouchBOSessionRepository(ctx, "s1", 10*time.Minute); err != nil {
		t.Fatal(err)
	}
	if mr.Exists(keyBOSession("s1")) {
		t.Fatal("touch ต้องไม่สร้าง session ที่หมดแล้วกลับมา")
	}
}

func TestBOSessionDelete(t *testing.T) { // AUTH-09
	setupRedis(t)
	ctx := context.Background()

	_ = CreateBOSessionRepository(ctx, "s1", newSession(7), time.Hour)
	for i := 0; i < 2; i++ { // เรียกซ้ำได้
		if err := DeleteBOSessionRepository(ctx, "s1", 7); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := GetBOSessionRepository(ctx, "s1", 7); !errors.Is(err, apperr.ErrNotFound) {
		t.Fatal("logout แล้วต้องใช้ไม่ได้")
	}
}

func TestBOSessionDeleteStaleDoesNotTouchNewSession(t *testing.T) {
	setupRedis(t)
	ctx := context.Background()

	_ = CreateBOSessionRepository(ctx, "old", newSession(7), time.Hour)
	_ = CreateBOSessionRepository(ctx, "new", newSession(7), time.Hour)
	_ = DeleteBOSessionRepository(ctx, "old", 7) // logout ด้วย token เก่า

	if _, err := GetBOSessionRepository(ctx, "new", 7); err != nil {
		t.Fatalf("logout ด้วย token เก่าต้องไม่เตะ session ใหม่: %v", err)
	}
}

func TestBOSessionDeleteAllOfAgent(t *testing.T) { // AUTH-15
	mr := setupRedis(t)
	ctx := context.Background()

	_ = CreateBOSessionRepository(ctx, "s1", newSession(7), time.Hour)
	if err := DeleteBOSessionsOfAgentRepository(ctx, 7); err != nil {
		t.Fatal(err)
	}
	if mr.Exists(keyBOSession("s1")) || mr.Exists(keyBOAgentSession(7)) {
		t.Fatal("ต้องไม่เหลือ session ของ agent")
	}
	if err := DeleteBOSessionsOfAgentRepository(ctx, 99); err != nil {
		t.Fatalf("agent ที่ไม่มี session ต้องไม่ error: %v", err)
	}
}

func TestBOLoginLimits(t *testing.T) { // AUTH-10, AUTH-11
	mr := setupRedis(t)
	ctx := context.Background()

	for i := 1; i <= 3; i++ {
		n, err := IncrBOLoginFailRepository(ctx, "agent01", 15*time.Minute)
		if err != nil || n != i {
			t.Fatalf("ครั้งที่ %d ได้ %d, %v", i, n, err)
		}
	}
	mr.FastForward(10 * time.Minute)
	_, _ = IncrBOLoginFailRepository(ctx, "agent01", 15*time.Minute)
	mr.FastForward(6 * time.Minute) // window เริ่มนับจากครั้งแรก ไม่ถูกยืด
	if n, _ := IncrBOLoginFailRepository(ctx, "agent01", 15*time.Minute); n != 1 {
		t.Fatalf("window หมดแล้วต้องเริ่มนับใหม่ ได้ %d", n)
	}

	if err := ClearBOLoginFailRepository(ctx, "agent01"); err != nil {
		t.Fatal(err)
	}
	if mr.Exists(keyBOLoginFail("agent01")) {
		t.Fatal("clear แล้วต้องไม่เหลือตัวนับ")
	}

	_ = BlockBOLoginRepository(ctx, "agent01", 15*time.Minute)
	if blocked, _ := IsBOLoginBlockedRepository(ctx, "agent01"); !blocked {
		t.Fatal("ต้องถูกระงับ")
	}
	mr.FastForward(16 * time.Minute)
	if blocked, _ := IsBOLoginBlockedRepository(ctx, "agent01"); blocked {
		t.Fatal("ครบเวลาแล้วต้องปลดอัตโนมัติ")
	}

	for i := 1; i <= 21; i++ {
		n, _ := IncrBOLoginIPRepository(ctx, "1.1.1.1")
		if n != i {
			t.Fatalf("ip ครั้งที่ %d ได้ %d", i, n)
		}
	}
	mr.FastForward(61 * time.Second)
	if n, _ := IncrBOLoginIPRepository(ctx, "1.1.1.1"); n != 1 {
		t.Fatalf("นาทีใหม่ต้องเริ่มนับใหม่ ได้ %d", n)
	}
}
