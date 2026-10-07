package currencyrate

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	redisRepo "app/app/repository/redis"
	"app/pkg/configs"
	"app/platform/database"
	"app/platform/logger"

	"github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"
)

// ทดสอบกับ askme จำลอง (httptest) + Redis จำลอง (miniredis) — docs/modules/currency_rate.md หัวข้อ 7

func setup(t *testing.T, handler http.HandlerFunc) (*miniredis.Miniredis, *int32) {
	t.Helper()
	logger.InitLogger(true)
	mr := miniredis.RunT(t)
	database.DBRedis = goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = database.DBRedis.Close(); database.DBRedis = nil })

	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		handler(w, r)
	}))
	t.Cleanup(srv.Close)
	prev := configs.Cfg
	configs.Cfg = &configs.Config{CurrencyRate: configs.CurrencyRateConfig{SyncEnabled: true, BaseURL: srv.URL}}
	t.Cleanup(func() { configs.Cfg = prev })
	return mr, &calls
}

func okBody(date string, rates string) string {
	return `{"code":200,"data":{"date":"` + date + `","base_currency":"USDT","json_rate":` + rates + `},"msg":{"en":"Success","th":"สำเร็จ"}}`
}

func TestSyncSuccess(t *testing.T) { // CR-04, CR-06, CR-08, CR-10
	var gotPath, gotDate string
	setup(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		var b map[string]string
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &b)
		gotDate = b["date"]
		_, _ = io.WriteString(w, okBody("2026-10-08", `{"THB":33.62,"USD":0.99,"USDT":1,"KRW":"1337.9","JPY":0,"XYZ":5}`))
	})
	now := time.Date(2026, 10, 7, 23, 30, 0, 0, time.UTC) // 06:30 วันที่ 8 เวลาไทย
	res, err := SyncCurrencyRateService(context.Background(), now)
	if err != nil {
		t.Fatal(err)
	}
	if gotPath != "/api/ex/rate/usdt" || gotDate != "2026-10-08" {
		t.Fatalf("request ผิด path=%s date=%s", gotPath, gotDate)
	}
	if res.Updated != 3 || res.Date != "2026-10-08" || len(res.Missing) != 24 {
		t.Fatalf("ผลไม่ถูก %+v", res)
	}
	rates, _ := redisRepo.GetCurrencyRatesRepository(context.Background())
	if rates["THB"] != 3362000000 || rates["USDT"] != 100000000 {
		t.Fatalf("เรทไม่ถูก %v", rates)
	}
	if _, ok := rates["KRW"]; ok { // string ไม่รับ
		t.Fatal("KRW ส่งเป็น string ต้องไม่รับ")
	}
	if _, ok := rates["XYZ"]; ok {
		t.Fatal("สกุลนอกระบบต้องไม่เก็บ")
	}
}

func TestSyncFailureKeepsPreviousRates(t *testing.T) { // CR-05, CR-07
	bodies := []struct {
		name   string
		status int
		body   string
	}{
		{"430503", 200, `{"code":430503,"msg":{"en":"USDT rate for the specified date not found"}}`},
		{"HTTP 500", 500, `oops`},
		{"ไม่ใช่ JSON", 200, `oops`},
		{"base ไม่ใช่ USDT", 200, `{"code":200,"data":{"date":"2026-10-07","base_currency":"USD","json_rate":{"THB":1}}}`},
	}
	for _, tt := range bodies {
		t.Run(tt.name, func(t *testing.T) {
			setup(t, func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.status)
				_, _ = io.WriteString(w, tt.body)
			})
			ctx := context.Background()
			_ = redisRepo.SaveCurrencyRatesRepository(ctx, map[string]int64{"THB": 3300000000}, redisRepo.CurrencyRateMeta{Date: "2026-10-06"})
			if _, err := SyncCurrencyRateService(ctx, time.Now()); err == nil {
				t.Fatal("ต้อง error")
			}
			rates, _ := redisRepo.GetCurrencyRatesRepository(ctx)
			if rates["THB"] != 3300000000 {
				t.Fatalf("เรทเดิมต้องไม่เปลี่ยน %v", rates)
			}
			if database.DBRedis.Exists(ctx, "currency_rate:sync:lock").Val() != 0 {
				t.Fatal("ต้องปล่อย lock หลังจบรอบ")
			}
		})
	}
}

func TestSyncSkippedWhenLocked(t *testing.T) { // CR-03
	mr, calls := setup(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, okBody("2026-10-07", `{"THB":33.62}`))
	})
	_ = mr.Set("currency_rate:sync:lock", "other")
	res, err := SyncCurrencyRateService(context.Background(), time.Now())
	if err != nil || !res.Skipped || atomic.LoadInt32(calls) != 0 {
		t.Fatalf("ต้องข้ามรอบโดยไม่ยิง: %+v %v calls=%d", res, err, *calls)
	}
	if v, _ := mr.Get("currency_rate:sync:lock"); v != "other" {
		t.Fatal("lock ของ instance อื่นต้องไม่ถูกลบ")
	}
}

func TestConvertAmount(t *testing.T) { // CR-13, CR-14
	setup(t, func(w http.ResponseWriter, r *http.Request) {})
	ctx := context.Background()
	if _, err := ConvertAmountService(ctx, 10000, "THB", "USD"); !errors.Is(err, ErrRateUnavailable) {
		t.Fatalf("ยังไม่มีเรทต้อง ErrRateUnavailable ได้ %v", err)
	}
	_ = redisRepo.SaveCurrencyRatesRepository(ctx, map[string]int64{"THB": 3362000000, "USD": 99000000}, redisRepo.CurrencyRateMeta{Date: "2026-10-07"})
	got, err := ConvertAmountService(ctx, 10000, "THB", "USD")
	if err != nil || got != 294 {
		t.Fatalf("100.00 THB → USD ต้องได้ 294 ได้ %d %v", got, err)
	}
	if got, _ := ConvertAmountService(ctx, 555, "VND", "VND"); got != 555 {
		t.Fatal("สกุลเดียวกันต้องได้ยอดเดิม")
	}
	if _, err := ConvertAmountService(ctx, 1, "THB", "VND"); !errors.Is(err, ErrRateUnavailable) {
		t.Fatal("ไม่มีเรท VND ต้อง ErrRateUnavailable")
	}
}
