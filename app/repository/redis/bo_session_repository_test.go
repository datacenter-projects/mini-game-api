package redis

import (
	"context"
	"errors"
	"testing"
	"time"

	"app/app/models"
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

const (
	agent = models.AccountTypeAgent
	sub   = models.AccountTypeSub
)

func newSession(agentID uint) BOSession {
	now := time.Now()
	return BOSession{AccountType: agent, AccountID: agentID, AgentID: agentID, IP: "1.1.1.1", CreatedAt: now, ExpiresAt: now.Add(12 * time.Hour)}
}

func newSubSession(subID, ownerID uint) BOSession {
	s := newSession(ownerID)
	s.AccountType, s.AccountID = sub, subID
	return s
}

func TestBOSessionSubAndAgentWithSameID(t *testing.T) { // AUTH-23: id ซ้ำกันได้ข้ามตาราง
	mr := setupRedis(t)
	ctx := context.Background()

	_ = CreateBOSessionRepository(ctx, "agent7", newSession(7), time.Hour)
	_ = CreateBOSessionRepository(ctx, "sub7", newSubSession(7, 3), time.Hour)

	if _, err := GetBOSessionRepository(ctx, "agent7", agent, 7); err != nil {
		t.Fatalf("sub login ต้องไม่เตะ agent ที่ id ตรงกัน: %v", err)
	}
	s, err := GetBOSessionRepository(ctx, "sub7", sub, 7)
	if err != nil || s.AgentID != 3 {
		t.Fatalf("got %+v, %v", s, err)
	}
	if _, err := GetBOSessionRepository(ctx, "sub7", agent, 7); !errors.Is(err, apperr.ErrNotFound) {
		t.Fatal("session ของ sub ต้องใช้ในนาม agent ไม่ได้")
	}

	_ = DeleteBOSessionRepository(ctx, "sub7", sub, 7) // sub logout
	if mr.Exists(keyBOAccountSession(sub, 7)) {
		t.Fatal("pointer ของ sub ต้องถูกลบ")
	}
	if _, err := GetBOSessionRepository(ctx, "agent7", agent, 7); err != nil {
		t.Fatalf("sub logout ต้องไม่กระทบ agent: %v", err)
	}
}

func TestBOPasscodeLimits(t *testing.T) { // AUTH-35, AUTH-47
	mr := setupRedis(t)
	ctx := context.Background()

	for i := 1; i <= 3; i++ {
		if n, err := IncrBOPasscodeFailRepository(ctx, sub, 7, 24*time.Hour); err != nil || n != i {
			t.Fatalf("ครั้งที่ %d ได้ %d, %v", i, n, err)
		}
	}
	if n, _ := IncrBOPasscodeFailRepository(ctx, agent, 7, 24*time.Hour); n != 1 {
		t.Fatalf("agent กับ sub ที่ id ตรงกันต้องนับแยก ได้ %d", n)
	}
	mr.FastForward(25 * time.Hour)
	if n, _ := IncrBOPasscodeFailRepository(ctx, sub, 7, 24*time.Hour); n != 1 {
		t.Fatalf("ครบ 24 ชม. ต้องเริ่มนับใหม่ ได้ %d", n)
	}

	_ = BlockBOPasscodeRepository(ctx, sub, 7, time.Hour)
	if blocked, _ := IsBOPasscodeBlockedRepository(ctx, sub, 7); !blocked {
		t.Fatal("ต้องถูกบล็อก")
	}
	if mr.Exists(keyBOPasscodeFail(sub, 7)) {
		t.Fatal("บล็อกแล้วต้องล้างตัวนับ")
	}
	mr.FastForward(61 * time.Minute)
	if blocked, _ := IsBOPasscodeBlockedRepository(ctx, sub, 7); blocked {
		t.Fatal("ครบ 1 ชม. ต้องปลดเอง")
	}

	_, _ = IncrBOPasscodeFailRepository(ctx, sub, 7, 24*time.Hour)
	_ = BlockBOPasscodeRepository(ctx, sub, 7, time.Hour)
	_, _ = IncrBOPasscodeFailRepository(ctx, sub, 7, 24*time.Hour)
	if err := ClearBOPasscodeBlockRepository(ctx, sub, 7); err != nil {
		t.Fatal(err)
	}
	if mr.Exists(keyBOPasscodeFail(sub, 7)) || mr.Exists(keyBOPasscodeBlock(sub, 7)) {
		t.Fatal("admin reset ต้องล้างทั้งตัวนับและบล็อก")
	}
}

func TestBOSessionCreateAndGet(t *testing.T) {
	setupRedis(t)
	ctx := context.Background()

	if err := CreateBOSessionRepository(ctx, "s1", newSession(7), time.Hour); err != nil {
		t.Fatal(err)
	}
	s, err := GetBOSessionRepository(ctx, "s1", agent, 7)
	if err != nil || s.AgentID != 7 || s.IP != "1.1.1.1" {
		t.Fatalf("got %+v, %v", s, err)
	}
	if _, err := GetBOSessionRepository(ctx, "s1", agent, 8); !errors.Is(err, apperr.ErrNotFound) {
		t.Fatalf("session ของ agent อื่นต้องไม่ผ่าน: %v", err)
	}
	if _, err := GetBOSessionRepository(ctx, "nope", agent, 7); !errors.Is(err, apperr.ErrNotFound) {
		t.Fatalf("session ที่ไม่มีต้องได้ ErrNotFound: %v", err)
	}
}

func TestBOSessionNewLoginKicksOld(t *testing.T) { // AUTH-06
	mr := setupRedis(t)
	ctx := context.Background()

	_ = CreateBOSessionRepository(ctx, "old", newSession(7), time.Hour)
	_ = CreateBOSessionRepository(ctx, "new", newSession(7), time.Hour)

	if _, err := GetBOSessionRepository(ctx, "old", agent, 7); !errors.Is(err, apperr.ErrNotFound) {
		t.Fatalf("session เก่าต้องใช้ไม่ได้: %v", err)
	}
	if mr.Exists(keyBOSession("old")) {
		t.Fatal("session เก่าต้องถูกลบออกจาก Redis")
	}
	if _, err := GetBOSessionRepository(ctx, "new", agent, 7); err != nil {
		t.Fatalf("session ใหม่ต้องใช้ได้: %v", err)
	}
	if _, err := GetBOSessionRepository(ctx, "x", agent, 8); !errors.Is(err, apperr.ErrNotFound) {
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
	if _, err := GetBOSessionRepository(ctx, "s1", agent, 7); err != nil {
		t.Fatalf("session ที่ถูกต่ออายุต้องยังอยู่: %v", err)
	}
	mr.FastForward(11 * time.Minute)
	if _, err := GetBOSessionRepository(ctx, "s1", agent, 7); !errors.Is(err, apperr.ErrNotFound) {
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
		if err := DeleteBOSessionRepository(ctx, "s1", agent, 7); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := GetBOSessionRepository(ctx, "s1", agent, 7); !errors.Is(err, apperr.ErrNotFound) {
		t.Fatal("logout แล้วต้องใช้ไม่ได้")
	}
}

func TestBOSessionDeleteStaleDoesNotTouchNewSession(t *testing.T) {
	setupRedis(t)
	ctx := context.Background()

	_ = CreateBOSessionRepository(ctx, "old", newSession(7), time.Hour)
	_ = CreateBOSessionRepository(ctx, "new", newSession(7), time.Hour)
	_ = DeleteBOSessionRepository(ctx, "old", agent, 7) // logout ด้วย token เก่า

	if _, err := GetBOSessionRepository(ctx, "new", agent, 7); err != nil {
		t.Fatalf("logout ด้วย token เก่าต้องไม่เตะ session ใหม่: %v", err)
	}
}

func TestBOSessionDeleteAllOfAgent(t *testing.T) { // AUTH-15
	mr := setupRedis(t)
	ctx := context.Background()

	_ = CreateBOSessionRepository(ctx, "s1", newSession(7), time.Hour)
	if err := DeleteBOSessionsOfAccountRepository(ctx, agent, 7); err != nil {
		t.Fatal(err)
	}
	if mr.Exists(keyBOSession("s1")) || mr.Exists(keyBOAccountSession(agent, 7)) {
		t.Fatal("ต้องไม่เหลือ session ของ agent")
	}
	if err := DeleteBOSessionsOfAccountRepository(ctx, agent, 99); err != nil {
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
