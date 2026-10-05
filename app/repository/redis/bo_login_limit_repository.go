package redis

import (
	"context"
	"time"

	"app/app/models"
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

// ClearBOLoginBlockRepository ล้างทั้งบล็อกและตัวนับ login ผิดของ username — ใช้ตอน admin รีเซ็ตรหัสผ่าน (AUTH-48)
func ClearBOLoginBlockRepository(ctx context.Context, username string) error {
	return database.DBRedis.Del(ctx, keyBOLoginBlock(username), keyBOLoginFail(username)).Err()
}

// ---- passcode ผิด (AUTH-35) — ตัวนับเดียวระดับบัญชี ----

// IncrBOPasscodeFailRepository นับครั้งที่ passcode ผิด — TTL นับจากครั้งแรกที่ผิด
func IncrBOPasscodeFailRepository(ctx context.Context, t models.AccountType, id uint, window time.Duration) (int, error) {
	return incrInWindow(ctx, keyBOPasscodeFail(t, id), window)
}

func ClearBOPasscodeFailRepository(ctx context.Context, t models.AccountType, id uint) error {
	return database.DBRedis.Del(ctx, keyBOPasscodeFail(t, id)).Err()
}

// BlockBOPasscodeRepository บล็อกบัญชีเพราะ passcode ผิดครบ และล้างตัวนับ
func BlockBOPasscodeRepository(ctx context.Context, t models.AccountType, id uint, d time.Duration) error {
	pipe := database.DBRedis.TxPipeline()
	pipe.Set(ctx, keyBOPasscodeBlock(t, id), 1, d)
	pipe.Del(ctx, keyBOPasscodeFail(t, id))
	_, err := pipe.Exec(ctx)
	return err
}

func IsBOPasscodeBlockedRepository(ctx context.Context, t models.AccountType, id uint) (bool, error) {
	n, err := database.DBRedis.Exists(ctx, keyBOPasscodeBlock(t, id)).Result()
	return n > 0, err
}

// ClearBOPasscodeBlockRepository ล้างทั้งบล็อกและตัวนับ passcode — ใช้ตอน admin รีเซ็ต passcode (AUTH-47)
func ClearBOPasscodeBlockRepository(ctx context.Context, t models.AccountType, id uint) error {
	return database.DBRedis.Del(ctx, keyBOPasscodeBlock(t, id), keyBOPasscodeFail(t, id)).Err()
}
