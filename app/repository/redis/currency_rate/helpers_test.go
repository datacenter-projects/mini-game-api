package currencyrate

import (
	"testing"

	"app/platform/database"

	"github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"
)

// setupRedis — Redis จำลองสำหรับ test ของ package นี้ (แพ็กเกจแยกต่อ module จึงไม่ใช้ helper ร่วมกัน)
func setupRedis(t *testing.T) *miniredis.Miniredis {
	t.Helper()
	mr := miniredis.RunT(t)
	database.DBRedis = goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = database.DBRedis.Close(); database.DBRedis = nil })
	return mr
}
