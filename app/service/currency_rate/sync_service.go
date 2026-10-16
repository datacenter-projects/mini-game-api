// Package currencyrate คือ business logic ของอัตราแลกเปลี่ยน: sync เรทรายเดือนจาก askmelotto และแปลงยอดเป็นบาท
// spec: docs/modules/currency_rate.md · job ตามเวลาอยู่ที่ app/cronjob (เรียก service นี้) · client ภายนอกอยู่ที่ platform/askmelotto
package currencyrate

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	currencyRateCore "app/app/core/currency_rate"
	"app/app/models"
	currencyRatePostgres "app/app/repository/postgres/currency_rate"
	currencyRateRedis "app/app/repository/redis/currency_rate"
	"app/pkg/utils"
	"app/platform/askmelotto"
	"app/platform/database"
	"app/platform/logger"
)

// lockTTL — อายุ lock ของรอบ sync (CR-13) · ใช้เป็น timeout ของทั้งรอบด้วย
const lockTTL = 5 * time.Minute

// SyncOptions — ตัวเลือกของรอบ sync (job ใช้ค่าว่าง · คำสั่งมือใช้ Month / Force — CR-17)
type SyncOptions struct {
	Month time.Time // เดือนที่จะ sync (core.MonthDate) · ศูนย์ = เดือนปัจจุบันตามเวลาไทย
	Force bool      // เขียนทับทั้งเดือน (คำสั่งมือ -force) · false = เติมเฉพาะสกุลที่ขาด
}

// SyncResult — ผลของรอบ sync ใช้ log และ test
type SyncResult struct {
	Skipped         bool      // ไม่ได้ lock — pod อื่นกำลังรัน (CR-13)
	AlreadyComplete bool      // เดือนนี้ครบ 27 สกุลแล้ว ไม่ยิง askmelotto (CR-14)
	Month           time.Time // เดือนที่ sync
	Saved           int64     // จำนวนแถวที่เขียนรอบนี้
	Rejected        []string  // สกุลที่มีใน response แต่ค่าผิด — ไม่บันทึก (CR-15)
	Missing         []string  // สกุลที่ยังไม่มีเรทของเดือนนี้หลังรอบนี้ (CR-04)
}

// ErrIncomplete — รอบนี้ยังไม่ครบ 27 สกุล (job ตั้ง FAILED · บอกสกุลที่ขาด) — บันทึกสกุลที่ได้ไปแล้ว (CR-14)
var ErrIncomplete = errors.New("currency rates incomplete")

// SyncCurrencyRateService — รอบ sync 1 ครั้ง (CR-02 – CR-05, CR-13 – CR-16)
// flow: lock → เดือนครบแล้วจบ → ยิงด้วยวันที่ 1 → ตรวจ response → INSERT ON CONFLICT DO NOTHING (หรือ -force เขียนทับ) → ครบ = SUCCESS · ไม่ครบ = ErrIncomplete
func SyncCurrencyRateService(ctx context.Context, now time.Time, opts SyncOptions) (SyncResult, error) {
	month := opts.Month
	if month.IsZero() {
		month = currencyRateCore.MonthDate(now)
	}
	res := SyncResult{Month: month}

	token, err := utils.RandomString("abcdefghijklmnopqrstuvwxyz0123456789", 32)
	if err != nil {
		return res, err
	}
	ok, err := currencyRateRedis.AcquireCurrencyRateLockRepository(ctx, token, lockTTL)
	if err != nil {
		return res, fmt.Errorf("acquire lock: %w", err)
	}
	if !ok {
		res.Skipped = true
		return res, nil
	}
	defer func() {
		// ปล่อย lock แม้ ctx ของรอบนี้ถูกยกเลิกแล้ว
		if err := currencyRateRedis.ReleaseCurrencyRateLockRepository(context.WithoutCancel(ctx), token); err != nil {
			logger.Ctx(ctx).Warnw("release currency rate lock failed", "error", err)
		}
	}()

	db := database.DBConn.WithContext(ctx)
	have, err := currencyRatePostgres.ListCurrenciesOfMonthRepository(db, month)
	if err != nil {
		return res, fmt.Errorf("list rates of month: %w", err)
	}
	if !opts.Force && len(currencyRateCore.MissingOf(have)) == 0 { // CR-14 (1): ครบแล้ว ไม่ยิง
		res.AlreadyComplete = true
		return res, nil
	}

	date := currencyRateCore.RequestDate(month)
	resp, err := askmelotto.FetchUSDTRates(ctx, date)
	if err != nil {
		logProblem(ctx, now, month, "askmelotto call failed", "error", err)
		return res, err
	}
	switch currencyRateCore.CheckResponse(currencyRateCore.Response{Code: resp.Code, BaseCurrency: resp.BaseCurrency, Date: resp.Date, JSONRate: resp.JSONRate}, date) {
	case currencyRateCore.ResponseOK:
	case currencyRateCore.ResponseBadCode:
		// 430503 = askmelotto ยังไม่มีเรทของวันนั้น (ปกติก่อนตี 1 ของวันที่ 1) · 430502 = รูปแบบวันที่ผิด (บั๊กฝั่งเรา)
		logProblem(ctx, now, month, "askmelotto returned an error code", "code", resp.Code, "msg", resp.Msg, "date", date)
		return res, fmt.Errorf("askmelotto code %d: %s", resp.Code, resp.Msg)
	case currencyRateCore.ResponseBadBase:
		logProblem(ctx, now, month, "askmelotto base_currency is not USDT", "base_currency", resp.BaseCurrency)
		return res, fmt.Errorf("askmelotto base_currency %q is not USDT", resp.BaseCurrency)
	case currencyRateCore.ResponseDateMismatch:
		logProblem(ctx, now, month, "askmelotto date does not match the requested date", "requested", date, "got", resp.Date)
		return res, fmt.Errorf("askmelotto date %q does not match requested %q", resp.Date, date)
	default:
		logProblem(ctx, now, month, "askmelotto json_rate is missing")
		return res, errors.New("askmelotto json_rate is missing")
	}

	accepted, rejected := currencyRateCore.ParseRates(resp.JSONRate)
	res.Rejected = rejected
	if len(rejected) > 0 {
		logger.Ctx(ctx).Warnw("currency rate value invalid — not saved", "month", date, "currencies", rejected)
	}
	rates, ok := currencyRateCore.BuildRates(accepted)
	if !ok { // ไม่มี THB = ไม่มีฐานคิด thb_rate → ไม่บันทึกทั้งเดือน (CR-04)
		logProblem(ctx, now, month, "THB rate missing or invalid — nothing saved", "month", date)
		return res, errors.New("THB rate missing or invalid — nothing saved")
	}

	syncedAt := time.Now()
	rows := make([]models.CurrencyRate, 0, len(rates))
	for _, r := range rates {
		rows = append(rows, models.CurrencyRate{RateMonth: month, Currency: r.Currency, USDTRate: r.USDTRate, THBRate: r.THBRate, SyncedAt: syncedAt})
	}
	if opts.Force { // CR-17: เขียนทับทั้งเดือนเท่าที่ได้
		if err := currencyRatePostgres.ReplaceCurrencyRatesRepository(db, rows); err != nil {
			return res, fmt.Errorf("replace rates: %w", err)
		}
		res.Saved = int64(len(rows))
	} else { // CR-05: ไม่ทับของเดิม — เขียนเฉพาะสกุลที่ยังไม่มี
		exists := make(map[string]bool, len(have))
		for _, c := range have {
			exists[c] = true
		}
		var fresh []models.CurrencyRate
		for _, r := range rows {
			if !exists[r.Currency] {
				fresh = append(fresh, r)
			}
		}
		// ON CONFLICT DO NOTHING กันซ้ำอีกชั้น (เผื่อมีใครเขียนเดือนนี้ระหว่างรอบ)
		if res.Saved, err = currencyRatePostgres.InsertCurrencyRatesIfAbsentRepository(db, fresh); err != nil {
			return res, fmt.Errorf("insert rates: %w", err)
		}
	}

	after, err := currencyRatePostgres.ListCurrenciesOfMonthRepository(db, month)
	if err != nil {
		return res, fmt.Errorf("list rates of month: %w", err)
	}
	res.Missing = currencyRateCore.MissingOf(after)
	if len(res.Missing) > 0 {
		logProblem(ctx, now, month, "currency rates incomplete", "month", date, "missing", res.Missing, "saved", res.Saved)
		return res, fmt.Errorf("%w: missing %s", ErrIncomplete, strings.Join(res.Missing, ","))
	}
	logger.Ctx(ctx).Infow("currency rates synced", "month", date, "saved", res.Saved)
	return res, nil
}

// logProblem — ระดับ log ตาม CR-16: ก่อน 03:00 ของวันที่ 1 (เดือนปัจจุบัน) = info · หลังจากนั้น = warn
func logProblem(ctx context.Context, now, month time.Time, msg string, kv ...any) {
	l := logger.Ctx(ctx)
	if month.Equal(currencyRateCore.MonthDate(now)) && currencyRateCore.IsEarlyMonth(now) {
		l.Infow(msg, kv...)
		return
	}
	l.Warnw(msg, kv...)
}
