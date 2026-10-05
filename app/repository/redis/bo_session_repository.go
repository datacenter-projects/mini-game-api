package redis

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"app/app/models"
	"app/pkg/apperr"
	"app/platform/database"

	goredis "github.com/redis/go-redis/v9"
)

// BOSession คือ session หลังบ้าน 1 รายการ — docs/modules/agent_auth_phase2.md หัวข้อ 6
type BOSession struct {
	AccountType models.AccountType `json:"account_type"`
	AccountID   uint               `json:"account_id"`
	AgentID     uint               `json:"agent_id"` // agent เอง หรือผู้สร้างถ้าเป็น sub (AUTH-26)
	IP          string             `json:"ip"`
	CreatedAt   time.Time          `json:"created_at"`
	ExpiresAt   time.Time          `json:"expires_at"` // absolute deadline (AUTH-07)
}

// CreateBOSessionRepository สร้าง session ใหม่และเตะ session เดิมของบัญชีออก (AUTH-06, AUTH-23)
//
// ลำดับ: เขียน session ใหม่ → สลับ pointer ของบัญชีแบบ atomic (SET ... GET) → ลบ session เดิมที่ได้คืนมา
// login พร้อมกันหลายที่ ทุกตัวได้ "ของเดิม" คนละตัวจาก SET GET จึงไม่มี session เก่าตัวไหนหลงเหลือ
func CreateBOSessionRepository(ctx context.Context, sid string, s BOSession, idleTTL time.Duration) error {
	payload, err := json.Marshal(s)
	if err != nil {
		return err
	}
	rdb := database.DBRedis
	if err := rdb.Set(ctx, keyBOSession(sid), payload, idleTTL).Err(); err != nil {
		return err
	}
	ptr := keyBOAccountSession(s.AccountType, s.AccountID)
	// ใช้ TTL (go-redis ส่งเป็น PX ระดับ ms) ไม่ใช้ ExpireAt ซึ่งส่งเป็น EXAT ระดับวินาที — ปัดเศษทำให้หลุดก่อน absolute ได้ถึง 1 วินาที
	old, err := rdb.SetArgs(ctx, ptr, sid, goredis.SetArgs{Get: true, TTL: time.Until(s.ExpiresAt)}).Result()
	if err != nil && !errors.Is(err, goredis.Nil) {
		rdb.Del(ctx, keyBOSession(sid)) // ไม่ทิ้ง session ที่ไม่มี pointer ชี้ไว้
		return err
	}
	if old != "" && old != sid {
		if err := rdb.Del(ctx, keyBOSession(old)).Err(); err != nil {
			return err
		}
	}
	return nil
}

// GetBOSessionRepository คืน session ที่ยังเป็น session ปัจจุบันของบัญชีเท่านั้น
// ไม่มี / ไม่ใช่ของบัญชีนี้ / ไม่ใช่ session ล่าสุด → apperr.ErrNotFound
func GetBOSessionRepository(ctx context.Context, sid string, t models.AccountType, id uint) (BOSession, error) {
	var s BOSession
	pipe := database.DBRedis.Pipeline()
	sessCmd := pipe.Get(ctx, keyBOSession(sid))
	ptrCmd := pipe.Get(ctx, keyBOAccountSession(t, id))
	if _, err := pipe.Exec(ctx); err != nil && !errors.Is(err, goredis.Nil) {
		return s, err
	}
	raw, err := sessCmd.Bytes()
	if errors.Is(err, goredis.Nil) {
		return s, apperr.ErrNotFound
	}
	if err != nil {
		return s, err
	}
	if err := json.Unmarshal(raw, &s); err != nil {
		return s, err
	}
	if s.AccountType != t || s.AccountID != id || ptrCmd.Val() != sid {
		return s, apperr.ErrNotFound
	}
	return s, nil
}

// TouchBOSessionRepository ต่ออายุ idle (AUTH-08) — session ที่หายไปแล้วจะไม่ถูกสร้างใหม่
func TouchBOSessionRepository(ctx context.Context, sid string, ttl time.Duration) error {
	// PExpire (ms) — Expire ของ go-redis ปัดเป็นวินาทีเต็ม ทำให้ session หลุดก่อนเวลาได้ถึง 1 วินาที
	return database.DBRedis.PExpire(ctx, keyBOSession(sid), ttl).Err()
}

// ลบ session และลบ pointer เฉพาะเมื่อยังชี้ไปที่ session นี้ (กันลบ session ใหม่ที่เพิ่ง login)
var deleteSessionScript = goredis.NewScript(`
redis.call('DEL', KEYS[1])
if redis.call('GET', KEYS[2]) == ARGV[1] then
	redis.call('DEL', KEYS[2])
end
return 1`)

// DeleteBOSessionRepository — logout (AUTH-09) · เรียกซ้ำได้
func DeleteBOSessionRepository(ctx context.Context, sid string, t models.AccountType, id uint) error {
	return deleteSessionScript.Run(ctx, database.DBRedis, []string{keyBOSession(sid), keyBOAccountSession(t, id)}, sid).Err()
}

var deleteAccountSessionsScript = goredis.NewScript(`
local sid = redis.call('GET', KEYS[1])
if sid then
	redis.call('DEL', ARGV[1] .. sid)
end
redis.call('DEL', KEYS[1])
return 1`)

// DeleteBOSessionsOfAccountRepository ลบทุก session ของบัญชี — ใช้ตอนถูกล็อก, passcode ผิดครบ, ถูกรีเซ็ต
func DeleteBOSessionsOfAccountRepository(ctx context.Context, t models.AccountType, id uint) error {
	return deleteAccountSessionsScript.Run(ctx, database.DBRedis, []string{keyBOAccountSession(t, id)}, keyBOSession("")).Err()
}
