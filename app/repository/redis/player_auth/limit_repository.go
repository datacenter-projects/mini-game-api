package playerauth

import (
	"context"
	"time"

	"app/platform/database"
)

// incrInWindow นับครั้งใน fixed window — TTL ถูกตั้งครั้งแรกครั้งเดียว (EXPIRE NX) ไม่ถูกยืดทุกครั้งที่นับ
func incrInWindow(ctx context.Context, key string, window time.Duration) (int, error) {
	pipe := database.DBRedis.TxPipeline()
	incr := pipe.Incr(ctx, key)
	pipe.ExpireNX(ctx, key, window)
	if _, err := pipe.Exec(ctx); err != nil {
		return 0, err
	}
	return int(incr.Val()), nil
}

// IncrLoginIPRepository นับจำนวนครั้งที่ IP นี้เรียก player/pb/login ในนาทีปัจจุบัน (PA-40)
func IncrLoginIPRepository(ctx context.Context, ip string) (int, error) {
	return incrInWindow(ctx, keyLoginIP(ip), time.Minute)
}

// IncrOperatorIPRepository นับจำนวนครั้งที่ IP นี้เรียก transfer/* ในนาทีปัจจุบัน (PA-42)
func IncrOperatorIPRepository(ctx context.Context, ip string) (int, error) {
	return incrInWindow(ctx, keyOperatorIP(ip), time.Minute)
}

// IncrLoginFailRepository นับครั้งที่ใส่รหัสผิดของ username ใน window (PA-41 · login + เปลี่ยนรหัส)
func IncrLoginFailRepository(ctx context.Context, username string, window time.Duration) (int, error) {
	return incrInWindow(ctx, keyLoginFail(username), window)
}

func ClearLoginFailRepository(ctx context.Context, username string) error {
	return database.DBRedis.Del(ctx, keyLoginFail(username)).Err()
}

// BlockLoginRepository ระงับ username ชั่วคราว และล้างตัวนับ
func BlockLoginRepository(ctx context.Context, username string, d time.Duration) error {
	pipe := database.DBRedis.TxPipeline()
	pipe.Set(ctx, keyLoginBlock(username), 1, d)
	pipe.Del(ctx, keyLoginFail(username))
	_, err := pipe.Exec(ctx)
	return err
}

func IsLoginBlockedRepository(ctx context.Context, username string) (bool, error) {
	n, err := database.DBRedis.Exists(ctx, keyLoginBlock(username)).Result()
	return n > 0, err
}
