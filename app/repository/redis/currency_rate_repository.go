package redis

import (
	"context"
	"strconv"
	"time"

	"app/platform/database"

	goredis "github.com/redis/go-redis/v9"
)

// อัตราแลกเปลี่ยน — docs/modules/currency_rate.md CR-03, CR-10, CR-11

// AcquireCurrencyRateLockRepository — SET NX PX (CR-03) · คืน true เมื่อได้ lock
func AcquireCurrencyRateLockRepository(ctx context.Context, token string, ttl time.Duration) (bool, error) {
	return database.DBRedis.SetNX(ctx, keyCurrencyRateLock, token, ttl).Result()
}

// ปล่อย lock เฉพาะถ้า token ยังเป็นของตัวเอง — กันลบ lock ที่ instance อื่นได้ไปหลังของเราหมดอายุ
var releaseLockScript = goredis.NewScript(`if redis.call("GET", KEYS[1]) == ARGV[1] then return redis.call("DEL", KEYS[1]) end return 0`)

// ReleaseCurrencyRateLockRepository — compare-and-delete (CR-03)
func ReleaseCurrencyRateLockRepository(ctx context.Context, token string) error {
	return releaseLockScript.Run(ctx, database.DBRedis, []string{keyCurrencyRateLock}, token).Err()
}

// CurrencyRateMeta — วันที่ของเรทจาก askme และเวลาที่ sync สำเร็จ
type CurrencyRateMeta struct {
	Date     string
	SyncedAt time.Time
}

// SaveCurrencyRatesRepository เขียนเรท (เฉพาะสกุลที่ส่งมา — สกุลอื่นคงค่าเดิม CR-08) และ meta ใน MULTI เดียว (CR-11)
func SaveCurrencyRatesRepository(ctx context.Context, rates map[string]int64, meta CurrencyRateMeta) error {
	fields := make(map[string]any, len(rates))
	for c, r := range rates {
		fields[c] = strconv.FormatInt(r, 10)
	}
	_, err := database.DBRedis.TxPipelined(ctx, func(p goredis.Pipeliner) error {
		if len(fields) > 0 {
			p.HSet(ctx, keyCurrencyRate, fields)
		}
		p.HSet(ctx, keyCurrencyRateMeta, "date", meta.Date, "synced_at", meta.SyncedAt.UTC().Format(time.RFC3339Nano))
		return nil
	})
	return err
}

// GetCurrencyRatesRepository — เรทล่าสุดทุกสกุล (ยังไม่เคย sync = map ว่าง) · ค่าที่อ่านไม่ได้ถูกข้าม
func GetCurrencyRatesRepository(ctx context.Context) (map[string]int64, error) {
	raw, err := database.DBRedis.HGetAll(ctx, keyCurrencyRate).Result()
	if err != nil {
		return nil, err
	}
	out := make(map[string]int64, len(raw))
	for c, s := range raw {
		if v, err := strconv.ParseInt(s, 10, 64); err == nil && v > 0 {
			out[c] = v
		}
	}
	return out, nil
}

// GetCurrencyRateMetaRepository — ยังไม่เคย sync คืนค่าว่าง
func GetCurrencyRateMetaRepository(ctx context.Context) (CurrencyRateMeta, error) {
	raw, err := database.DBRedis.HGetAll(ctx, keyCurrencyRateMeta).Result()
	if err != nil {
		return CurrencyRateMeta{}, err
	}
	m := CurrencyRateMeta{Date: raw["date"]}
	if t, err := time.Parse(time.RFC3339Nano, raw["synced_at"]); err == nil {
		m.SyncedAt = t
	}
	return m, nil
}
