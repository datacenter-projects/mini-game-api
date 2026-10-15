package currencyrate

import (
	"context"
	"time"

	"app/platform/database"

	goredis "github.com/redis/go-redis/v9"
)

// Redis ของ currency_rate เหลือแค่ lock กันหลาย pod ยิง askmelotto พร้อมกัน (CR-13) — เรทเก็บใน Postgres (CR-06)

// AcquireCurrencyRateLockRepository — SET NX PX (CR-13) · คืน true เมื่อได้ lock
func AcquireCurrencyRateLockRepository(ctx context.Context, token string, ttl time.Duration) (bool, error) {
	return database.DBRedis.SetNX(ctx, keyLock(), token, ttl).Result()
}

// ปล่อย lock เฉพาะถ้า token ยังเป็นของตัวเอง — กันลบ lock ที่ pod อื่นได้ไปหลังของเราหมดอายุ
var releaseLockScript = goredis.NewScript(`if redis.call("GET", KEYS[1]) == ARGV[1] then return redis.call("DEL", KEYS[1]) end return 0`)

// ReleaseCurrencyRateLockRepository — compare-and-delete (CR-13)
func ReleaseCurrencyRateLockRepository(ctx context.Context, token string) error {
	return releaseLockScript.Run(ctx, database.DBRedis, []string{keyLock()}, token).Err()
}
