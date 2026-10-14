package depositwithdraw

import (
	"context"
	"errors"
	"time"

	"app/app/models"
	"app/platform/database"

	goredis "github.com/redis/go-redis/v9"
)

// AcquireDWLockRepository — SET key token NX PX ttl (DW-20) · ได้ lock = true · มีคนถืออยู่ = false
// error = Redis ใช้ไม่ได้ (ผู้เรียกต้องปฏิเสธรายการ — DW-21 fail closed)
func AcquireDWLockRepository(ctx context.Context, owner models.BalanceOwnerType, id uint, token string, ttl time.Duration) (bool, error) {
	err := database.DBRedis.SetArgs(ctx, keyDWLock(owner, id), token, goredis.SetArgs{Mode: "NX", TTL: ttl}).Err()
	if errors.Is(err, goredis.Nil) {
		return false, nil
	}
	return err == nil, err
}

// releaseScript — ลบเฉพาะเมื่อค่ายังเป็น token ของตัวเอง (DW-22) · lock หมดอายุแล้วคนอื่นได้ไป = ไม่ลบ
var releaseScript = goredis.NewScript(`
if redis.call("GET", KEYS[1]) == ARGV[1] then
	return redis.call("DEL", KEYS[1])
end
return 0`)

// ReleaseDWLockRepository — ปลด lock ด้วย compare-and-delete · lock ไม่ใช่ของตัวเอง / หมดอายุแล้ว = nil (ไม่ใช่ error)
func ReleaseDWLockRepository(ctx context.Context, owner models.BalanceOwnerType, id uint, token string) error {
	return releaseScript.Run(ctx, database.DBRedis, []string{keyDWLock(owner, id)}, token).Err()
}
