package playerauth

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"app/pkg/apperr"
	"app/platform/database"

	goredis "github.com/redis/go-redis/v9"
)

// ErrKicked — session นี้ถูก login ใหม่เตะออก (PA-22 → 407204)
var ErrKicked = errors.New("player session replaced by a newer login")

// Session คือ session ผู้เล่น 1 รายการ (PA-20 – PA-22)
type Session struct {
	MemberID  uint      `json:"member_id"`
	IP        string    `json:"ip"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"` // absolute deadline (PA-21)
}

// CreateSessionRepository สร้าง session ใหม่และเตะ session เดิมของ Member ออก (PA-22)
//
// ลำดับ: เขียน session ใหม่ → สลับ pointer แบบ atomic (SET ... GET) → ลบ session เดิมที่ได้คืนมาพร้อมวางเครื่องหมาย kicked
// login พร้อมกันหลายที่ ทุกตัวได้ "ของเดิม" คนละตัวจาก SET GET จึงไม่มี session เก่าตัวไหนหลงเหลือ
func CreateSessionRepository(ctx context.Context, sid string, s Session, idleTTL time.Duration) error {
	payload, err := json.Marshal(s)
	if err != nil {
		return err
	}
	rdb := database.DBRedis
	if err := rdb.Set(ctx, keySession(sid), payload, idleTTL).Err(); err != nil {
		return err
	}
	absolute := time.Until(s.ExpiresAt)
	old, err := rdb.SetArgs(ctx, keyMemberSession(s.MemberID), sid, goredis.SetArgs{Get: true, TTL: absolute}).Result()
	if err != nil && !errors.Is(err, goredis.Nil) {
		rdb.Del(ctx, keySession(sid)) // ไม่ทิ้ง session ที่ไม่มี pointer ชี้ไว้
		return err
	}
	if old != "" && old != sid {
		pipe := rdb.TxPipeline()
		pipe.Del(ctx, keySession(old))
		pipe.Set(ctx, keyKicked(old), 1, absolute) // อยู่ได้เท่า absolute ของ session ใหม่ (เกินของเก่าเสมอ)
		if _, err := pipe.Exec(ctx); err != nil {
			return err
		}
	}
	return nil
}

// GetSessionRepository คืน session ที่ยังเป็น session ปัจจุบันของ Member เท่านั้น
// ไม่มี → apperr.ErrNotFound · ถูก login ใหม่เตะ (session หายและมีเครื่องหมาย kicked) → ErrKicked
func GetSessionRepository(ctx context.Context, sid string, memberID uint) (Session, error) {
	var s Session
	pipe := database.DBRedis.Pipeline()
	sessCmd := pipe.Get(ctx, keySession(sid))
	ptrCmd := pipe.Get(ctx, keyMemberSession(memberID))
	kickedCmd := pipe.Exists(ctx, keyKicked(sid))
	if _, err := pipe.Exec(ctx); err != nil && !errors.Is(err, goredis.Nil) {
		return s, err
	}
	raw, err := sessCmd.Bytes()
	if errors.Is(err, goredis.Nil) {
		if kickedCmd.Val() > 0 {
			return s, ErrKicked
		}
		return s, apperr.ErrNotFound
	}
	if err != nil {
		return s, err
	}
	if err := json.Unmarshal(raw, &s); err != nil {
		return s, err
	}
	if s.MemberID != memberID {
		return s, apperr.ErrNotFound
	}
	if ptrCmd.Val() != sid { // session ยังอยู่แต่ไม่ใช่ตัวล่าสุด (กรณีแข่งกัน) = ถูกเตะ
		return s, ErrKicked
	}
	return s, nil
}

// TouchSessionRepository ต่ออายุ idle (PA-21) — session ที่หายไปแล้วจะไม่ถูกสร้างใหม่ · PExpire (ms) กันหลุดก่อนเวลา
func TouchSessionRepository(ctx context.Context, sid string, ttl time.Duration) error {
	return database.DBRedis.PExpire(ctx, keySession(sid), ttl).Err()
}

// ลบ session และลบ pointer เฉพาะเมื่อยังชี้ไปที่ session นี้ (กันลบ session ใหม่ที่เพิ่ง login)
var deleteSessionScript = goredis.NewScript(`
redis.call('DEL', KEYS[1])
if redis.call('GET', KEYS[2]) == ARGV[1] then
	redis.call('DEL', KEYS[2])
end
return 1`)

// DeleteSessionRepository — logout (PA-26) · เรียกซ้ำได้
func DeleteSessionRepository(ctx context.Context, sid string, memberID uint) error {
	return deleteSessionScript.Run(ctx, database.DBRedis, []string{keySession(sid), keyMemberSession(memberID)}, sid).Err()
}

var deleteMemberSessionsScript = goredis.NewScript(`
local sid = redis.call('GET', KEYS[1])
if sid then
	redis.call('DEL', ARGV[1] .. sid)
end
redis.call('DEL', KEYS[1])
return 1`)

// DeleteSessionsOfMemberRepository ลบ session ของ Member — ใช้ตอนถูกล็อก (PA-32 / PA-34)
func DeleteSessionsOfMemberRepository(ctx context.Context, memberID uint) error {
	return deleteMemberSessionsScript.Run(ctx, database.DBRedis, []string{keyMemberSession(memberID)}, keySession("")).Err()
}
