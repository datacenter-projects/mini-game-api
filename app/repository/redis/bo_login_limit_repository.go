package redis

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

// IncrBOLoginIPRepository นับจำนวนครั้งที่ IP นี้เรียก login ในนาทีปัจจุบัน (AUTH-11)
func IncrBOLoginIPRepository(ctx context.Context, ip string) (int, error) {
	return incrInWindow(ctx, keyBOLoginIP(ip), time.Minute)
}

// IncrBOLoginFailRepository นับครั้งที่ login ผิดของ username ใน window (AUTH-10)
func IncrBOLoginFailRepository(ctx context.Context, username string, window time.Duration) (int, error) {
	return incrInWindow(ctx, keyBOLoginFail(username), window)
}

func ClearBOLoginFailRepository(ctx context.Context, username string) error {
	return database.DBRedis.Del(ctx, keyBOLoginFail(username)).Err()
}

// BlockBOLoginRepository ระงับการ login ของ username ชั่วคราว และล้างตัวนับ
func BlockBOLoginRepository(ctx context.Context, username string, d time.Duration) error {
	pipe := database.DBRedis.TxPipeline()
	pipe.Set(ctx, keyBOLoginBlock(username), 1, d)
	pipe.Del(ctx, keyBOLoginFail(username))
	_, err := pipe.Exec(ctx)
	return err
}

func IsBOLoginBlockedRepository(ctx context.Context, username string) (bool, error) {
	n, err := database.DBRedis.Exists(ctx, keyBOLoginBlock(username)).Result()
	return n > 0, err
}
