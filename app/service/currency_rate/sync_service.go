// Package currencyrate ดึงอัตราแลกเปลี่ยนเทียบ USDT จาก askme และแปลงยอดข้ามสกุลเพื่อแสดงผล
// spec: docs/modules/currency_rate.md
package currencyrate

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	currencyRateCore "app/app/core/currency_rate"
	redisRepo "app/app/repository/redis"
	"app/pkg/configs"
	"app/pkg/utils"
	"app/platform/logger"
)

const (
	syncInterval = 15 * time.Minute // CR-01
	lockTTL      = 5 * time.Minute  // CR-03
	fetchTimeout = 10 * time.Second // CR-04
	ratePath     = "/api/ex/rate/usdt"
)

var httpClient = &http.Client{Timeout: fetchTimeout}

// ข้อมูลที่ต้องการจาก response ของ askme (หัวข้อ 5)
type askmeRateResponse struct {
	Code int `json:"code"`
	Data struct {
		Date         string                     `json:"date"`
		BaseCurrency string                     `json:"base_currency"`
		JSONRate     map[string]json.RawMessage `json:"json_rate"`
	} `json:"data"`
	Msg struct {
		EN string `json:"en"`
	} `json:"msg"`
}

// SyncResult — ผลของรอบ sync 1 ครั้ง ใช้ log และ test
type SyncResult struct {
	Skipped bool     // ไม่ได้ lock — instance อื่นกำลังดึง
	Date    string   // วันที่ของเรทจาก askme
	Updated int      // จำนวนสกุลที่อัปเดต
	Missing []string // สกุลที่ขาดหรือค่าผิด — คงค่าเดิม (CR-08)
}

// SyncCurrencyRateService — รอบ sync 1 ครั้ง (CR-03 – CR-08, CR-10, CR-11)
// ไม่สำเร็จคืน error และไม่แตะเรทเดิม (CR-07)
func SyncCurrencyRateService(ctx context.Context, now time.Time) (SyncResult, error) {
	token, err := utils.RandomString("abcdefghijklmnopqrstuvwxyz0123456789", 32)
	if err != nil {
		return SyncResult{}, err
	}
	ok, err := redisRepo.AcquireCurrencyRateLockRepository(ctx, token, lockTTL)
	if err != nil {
		return SyncResult{}, fmt.Errorf("acquire lock: %w", err)
	}
	if !ok {
		return SyncResult{Skipped: true}, nil
	}
	defer func() {
		// ปล่อย lock แม้ ctx ของรอบนี้ถูกยกเลิกแล้ว
		if err := redisRepo.ReleaseCurrencyRateLockRepository(context.WithoutCancel(ctx), token); err != nil {
			logger.Ctx(ctx).Warnw("release currency rate lock failed", "error", err)
		}
	}()

	res, err := fetchRates(ctx, currencyRateCore.RequestDate(now))
	if err != nil {
		return SyncResult{}, err
	}

	rates := make(map[string]int64, len(currencyRateCore.Currencies))
	for _, c := range currencyRateCore.Currencies {
		v, ok := parseRateValue(res.Data.JSONRate[c])
		if !ok {
			continue
		}
		rates[c] = v
	}
	result := SyncResult{Date: res.Data.Date, Updated: len(rates)}
	for _, c := range currencyRateCore.Currencies {
		if _, ok := rates[c]; !ok {
			result.Missing = append(result.Missing, c)
		}
	}
	meta := redisRepo.CurrencyRateMeta{Date: res.Data.Date, SyncedAt: now}
	if err := redisRepo.SaveCurrencyRatesRepository(ctx, rates, meta); err != nil {
		return SyncResult{}, fmt.Errorf("save rates: %w", err)
	}
	return result, nil
}

// fetchRates ยิง askme แล้วตรวจตาม CR-05 · error ของ askme ตอบ HTTP 200 แต่ code ไม่ใช่ 200
func fetchRates(ctx context.Context, date string) (askmeRateResponse, error) {
	var out askmeRateResponse
	body, _ := json.Marshal(map[string]string{"date": date})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, configs.Cfg.CurrencyRate.BaseURL+ratePath, bytes.NewReader(body))
	if err != nil {
		return out, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := httpClient.Do(req)
	if err != nil {
		return out, fmt.Errorf("call askme: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return out, fmt.Errorf("read askme response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return out, fmt.Errorf("askme http status %d", resp.StatusCode)
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return out, fmt.Errorf("decode askme response: %w", err)
	}
	if out.Code != http.StatusOK {
		return out, fmt.Errorf("askme code %d: %s", out.Code, out.Msg.EN)
	}
	if out.Data.BaseCurrency != "USDT" || out.Data.JSONRate == nil {
		return out, errors.New("askme response: base_currency is not USDT or json_rate missing")
	}
	return out, nil
}

// parseRateValue — ต้องเป็น JSON number (ไม่รับ string / null) แล้วแปลงตาม CR-06
func parseRateValue(raw json.RawMessage) (int64, bool) {
	if len(raw) == 0 || raw[0] < '0' || raw[0] > '9' {
		return 0, false
	}
	return currencyRateCore.ParseRate(string(raw))
}

// StartSyncWorker เริ่ม worker ดึงเรท (CR-01, CR-02) — ดึงทันที 1 รอบแล้วทุก 15 นาที จนกว่า ctx จะถูกยกเลิก
// IS_CURRENCY_RATE_SYNC=false ไม่เริ่มเลย · goroutine เดียว แต่ละรอบทำต่อกัน ไม่ซ้อน (กฎข้อ 27)
func StartSyncWorker(ctx context.Context) {
	if !configs.Cfg.CurrencyRate.SyncEnabled {
		logger.Ctx(ctx).Infow("currency rate sync disabled")
		return
	}
	go func() {
		ticker := time.NewTicker(syncInterval)
		defer ticker.Stop()
		for {
			runOnce(ctx)
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
		}
	}()
}

func runOnce(ctx context.Context) {
	rctx, cancel := context.WithTimeout(ctx, lockTTL)
	defer cancel()
	res, err := SyncCurrencyRateService(rctx, time.Now())
	switch {
	case err != nil:
		logger.Ctx(ctx).Warnw("currency rate sync failed — keep previous rates", "error", err)
	case res.Skipped:
		logger.Ctx(ctx).Debugw("currency rate sync skipped — another instance holds the lock")
	default:
		if len(res.Missing) > 0 {
			logger.Ctx(ctx).Warnw("currency rate missing or invalid — keep previous value", "currencies", res.Missing)
		}
		logger.Ctx(ctx).Infow("currency rate synced", "date", res.Date, "updated", res.Updated)
	}
}
