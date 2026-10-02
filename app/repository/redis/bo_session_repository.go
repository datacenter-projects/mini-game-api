package redis

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"app/pkg/apperr"
	"app/platform/database"

	goredis "github.com/redis/go-redis/v9"
)

// BOSession คือ session หลังบ้าน 1 รายการ — docs/modules/agent_auth.md หัวข้อ 6
type BOSession struct {
	AgentID   uint      `json:"agent_id"`
	IP        string    `json:"ip"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"` // absolute deadline
}

// CreateBOSessionRepository สร้าง session ใหม่และเตะ session เดิมของ agent ออก (AUTH-06)
//
// ลำดับ: เขียน session ใหม่ → สลับ pointer ของ agent แบบ atomic (SET ... GET) → ลบ session เดิมที่ได้คืนมา
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
	old, err := rdb.SetArgs(ctx, keyBOAgentSession(s.AgentID), sid, goredis.SetArgs{Get: true, ExpireAt: s.ExpiresAt}).Result()
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

// GetBOSessionRepository คืน session ที่ยังเป็น session ปัจจุบันของ agent เท่านั้น
// ไม่มี / ไม่ใช่ของ agent นี้ / ไม่ใช่ session ล่าสุด → apperr.ErrNotFound
func GetBOSessionRepository(ctx context.Context, sid string, agentID uint) (BOSession, error) {
	var s BOSession
	pipe := database.DBRedis.Pipeline()
	sessCmd := pipe.Get(ctx, keyBOSession(sid))
	ptrCmd := pipe.Get(ctx, keyBOAgentSession(agentID))
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
	if s.AgentID != agentID || ptrCmd.Val() != sid {
		return s, apperr.ErrNotFound
	}
	return s, nil
}

// TouchBOSessionRepository ต่ออายุ idle (AUTH-08) — session ที่หายไปแล้วจะไม่ถูกสร้างใหม่
func TouchBOSessionRepository(ctx context.Context, sid string, ttl time.Duration) error {
	return database.DBRedis.Expire(ctx, keyBOSession(sid), ttl).Err()
}

// ลบ session และลบ pointer เฉพาะเมื่อยังชี้ไปที่ session นี้ (กันลบ session ใหม่ที่เพิ่ง login)
var deleteSessionScript = goredis.NewScript(`
redis.call('DEL', KEYS[1])
if redis.call('GET', KEYS[2]) == ARGV[1] then
	redis.call('DEL', KEYS[2])
end
return 1`)

// DeleteBOSessionRepository — logout (AUTH-09) · เรียกซ้ำได้
func DeleteBOSessionRepository(ctx context.Context, sid string, agentID uint) error {
	return deleteSessionScript.Run(ctx, database.DBRedis, []string{keyBOSession(sid), keyBOAgentSession(agentID)}, sid).Err()
}

var deleteAgentSessionsScript = goredis.NewScript(`
local sid = redis.call('GET', KEYS[1])
if sid then
	redis.call('DEL', ARGV[1] .. sid)
end
redis.call('DEL', KEYS[1])
return 1`)

// DeleteBOSessionsOfAgentRepository ลบทุก session ของ agent — ใช้ตอนบัญชีถูกล็อก (AUTH-15)
func DeleteBOSessionsOfAgentRepository(ctx context.Context, agentID uint) error {
	return deleteAgentSessionsScript.Run(ctx, database.DBRedis, []string{keyBOAgentSession(agentID)}, keyBOSession("")).Err()
}
