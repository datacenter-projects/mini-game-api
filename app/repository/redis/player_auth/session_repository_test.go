package playerauth

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

func newSession(memberID uint) Session {
	now := time.Now()
	return Session{MemberID: memberID, IP: "1.1.1.1", CreatedAt: now, ExpiresAt: now.Add(4 * time.Hour)}
}

func TestSessionCreateGet(t *testing.T) { // rule: PA-20, PA-22
	setupRedis(t)
	ctx := context.Background()
	if err := CreateSessionRepository(ctx, "a", newSession(7), 30*time.Minute); err != nil {
		t.Fatal(err)
	}
	s, err := GetSessionRepository(ctx, "a", 7)
	if err != nil || s.MemberID != 7 {
		t.Fatalf("Get = %+v, %v", s, err)
	}
	if _, err := GetSessionRepository(ctx, "a", 8); !errors.Is(err, apperr.ErrNotFound) { // sid ของคนอื่น
		t.Errorf("sid ของ member อื่น = %v want ErrNotFound", err)
	}
	if _, err := GetSessionRepository(ctx, "nope", 7); !errors.Is(err, apperr.ErrNotFound) {
		t.Errorf("ไม่มี session = %v want ErrNotFound", err)
	}
}

func TestNewLoginKicksOldSession(t *testing.T) { // rule: PA-22 → 407204
	setupRedis(t)
	ctx := context.Background()
	_ = CreateSessionRepository(ctx, "old", newSession(7), 30*time.Minute)
	_ = CreateSessionRepository(ctx, "new", newSession(7), 30*time.Minute)

	if _, err := GetSessionRepository(ctx, "old", 7); !errors.Is(err, ErrKicked) {
		t.Errorf("session เก่า = %v want ErrKicked", err)
	}
	if _, err := GetSessionRepository(ctx, "new", 7); err != nil {
		t.Errorf("session ใหม่ต้องใช้ได้: %v", err)
	}
	_ = CreateSessionRepository(ctx, "other", newSession(8), 30*time.Minute) // คนอื่น login ไม่กระทบ
	if _, err := GetSessionRepository(ctx, "new", 7); err != nil {
		t.Errorf("member อื่น login ต้องไม่เตะ: %v", err)
	}
}

func TestDeleteSession(t *testing.T) { // rule: PA-26
	setupRedis(t)
	ctx := context.Background()
	_ = CreateSessionRepository(ctx, "a", newSession(7), 30*time.Minute)
	for i := 0; i < 2; i++ { // เรียกซ้ำได้
		if err := DeleteSessionRepository(ctx, "a", 7); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := GetSessionRepository(ctx, "a", 7); !errors.Is(err, apperr.ErrNotFound) { // logout ไม่ใช่ถูกเตะ
		t.Errorf("หลัง logout = %v want ErrNotFound", err)
	}
}

func TestLogoutOldTokenKeepsCurrent(t *testing.T) { // rule: PA-26 — logout ด้วย token เก่าไม่กระทบ session ปัจจุบัน
	setupRedis(t)
	ctx := context.Background()
	_ = CreateSessionRepository(ctx, "old", newSession(7), 30*time.Minute)
	_ = CreateSessionRepository(ctx, "new", newSession(7), 30*time.Minute)
	_ = DeleteSessionRepository(ctx, "old", 7)
	if _, err := GetSessionRepository(ctx, "new", 7); err != nil {
		t.Errorf("session ปัจจุบันต้องอยู่: %v", err)
	}
}

func TestDeleteSessionsOfMember(t *testing.T) { // rule: PA-32, PA-34 — ล็อกแล้ว token เดิม = session สิ้นสุด (ไม่ใช่ kicked)
	setupRedis(t)
	ctx := context.Background()
	_ = CreateSessionRepository(ctx, "a", newSession(7), 30*time.Minute)
	_ = CreateSessionRepository(ctx, "b", newSession(8), 30*time.Minute)
	if err := DeleteSessionsOfMemberRepository(ctx, 7); err != nil {
		t.Fatal(err)
	}
	if err := DeleteSessionsOfMemberRepository(ctx, 7); err != nil { // ซ้ำ / ไม่มี session ได้
		t.Fatal(err)
	}
	if _, err := GetSessionRepository(ctx, "a", 7); !errors.Is(err, apperr.ErrNotFound) {
		t.Errorf("หลังลบ = %v want ErrNotFound", err)
	}
	if _, err := GetSessionRepository(ctx, "b", 8); err != nil {
		t.Errorf("member อื่นต้องไม่ถูกลบ: %v", err)
	}
}

func TestTouchSession(t *testing.T) { // rule: PA-21
	mr := setupRedis(t)
	ctx := context.Background()
	_ = CreateSessionRepository(ctx, "a", newSession(7), 30*time.Minute)
	mr.FastForward(20 * time.Minute)
	if err := TouchSessionRepository(ctx, "a", 30*time.Minute); err != nil {
		t.Fatal(err)
	}
	mr.FastForward(20 * time.Minute) // ครบ 40 นาทีนับจากสร้าง แต่ต่ออายุที่นาที 20
	if _, err := GetSessionRepository(ctx, "a", 7); err != nil {
		t.Errorf("ต่ออายุแล้วต้องยังอยู่: %v", err)
	}
	mr.FastForward(31 * time.Minute)
	if _, err := GetSessionRepository(ctx, "a", 7); !errors.Is(err, apperr.ErrNotFound) {
		t.Errorf("idle หมด = %v want ErrNotFound", err)
	}
}

func TestLimits(t *testing.T) { // rule: PA-40, PA-41, PA-42
	mr := setupRedis(t)
	ctx := context.Background()
	for i := 1; i <= 3; i++ {
		n, err := IncrLoginIPRepository(ctx, "1.2.3.4")
		if err != nil || n != i {
			t.Fatalf("IncrLoginIP #%d = %d, %v", i, n, err)
		}
	}
	if n, _ := IncrOperatorIPRepository(ctx, "1.2.3.4"); n != 1 { // key คนละตัวกับ login
		t.Errorf("operator counter = %d want 1", n)
	}
	mr.FastForward(61 * time.Second)
	if n, _ := IncrLoginIPRepository(ctx, "1.2.3.4"); n != 1 {
		t.Errorf("หลังหมด window = %d want 1", n)
	}

	_, _ = IncrLoginFailRepository(ctx, "mem01", 15*time.Minute)
	_, _ = IncrLoginFailRepository(ctx, "mem01", 15*time.Minute)
	if err := BlockLoginRepository(ctx, "mem01", 15*time.Minute); err != nil {
		t.Fatal(err)
	}
	if blocked, _ := IsLoginBlockedRepository(ctx, "mem01"); !blocked {
		t.Error("ต้องถูกบล็อก")
	}
	if n, _ := IncrLoginFailRepository(ctx, "mem01", 15*time.Minute); n != 1 { // ตัวนับถูกล้างตอนบล็อก
		t.Errorf("ตัวนับหลังบล็อก = %d want 1", n)
	}
	mr.FastForward(16 * time.Minute)
	if blocked, _ := IsLoginBlockedRepository(ctx, "mem01"); blocked {
		t.Error("บล็อกต้องหมดอายุ")
	}
}
