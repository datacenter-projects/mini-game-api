// Package askmelotto คือ client เดียวของทุกเส้นที่เรียกระบบ askmelotto (ระบบแม่) — spec: docs/modules/currency_rate.md หัวข้อ 7 (J3)
// ตอนนี้มีเส้นเดียว (เรท USDT) · เส้นใหม่เพิ่มที่ package นี้ · service เรียกผ่านฟังก์ชันของ package นี้เท่านั้น ไม่ยิง HTTP เอง
// ฐาน URL อ่านจาก configs.Cfg.Askmelotto.BaseURL ทุกครั้งที่เรียก (test ชี้ไป httptest server ได้)
package askmelotto

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"app/pkg/configs"
)

const (
	usdtRatePath = "/api/ex/rate/usdt"
	timeout      = 10 * time.Second // CR-02
	maxBody      = 1 << 20          // อ่าน response ไม่เกิน 1 MB
)

var httpClient = &http.Client{Timeout: timeout}

// USDTRateResponse คือส่วนของ response เรท USDT ที่ใช้ — error ของ askmelotto ตอบ HTTP 200 แต่ Code ≠ 200
// (เช่น 430502 รูปแบบวันที่ผิด · 430503 ยังไม่มีเรทของวันนั้น) จึงคืนเป็นผลปกติ ให้ผู้เรียกอ่าน Code / Msg เอง
type USDTRateResponse struct {
	Code         int
	Msg          string // msg.en ของ askmelotto (ไว้ log)
	Date         string
	BaseCurrency string
	JSONRate     map[string]json.RawMessage // สกุล → ตัวเลขดิบ (ยังไม่แปลง — ผู้เรียกตรวจว่าเป็น JSON number)
}

// wire คือรูป JSON จริงของ response
type usdtRateWire struct {
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

// FetchUSDTRates ยิง `POST {ASKMELOTTO_API_BASE_URL}/api/ex/rate/usdt` ด้วย `{"date": date}` (date = YYYY-MM-01 — CR-02)
// คืน error เฉพาะเชื่อมต่อไม่ได้ / timeout / HTTP ไม่ใช่ 200 / body ไม่ใช่ JSON · ไม่ต้องยืนยันตัวตน
// ห้ามใส่ body ดิบลงใน error (มีแต่ตัวเลขเรท แต่ไม่ควรหลุดลง log เกินจำเป็น)
func FetchUSDTRates(ctx context.Context, date string) (USDTRateResponse, error) {
	var out USDTRateResponse
	body, err := json.Marshal(map[string]string{"date": date})
	if err != nil {
		return out, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, configs.Cfg.Askmelotto.BaseURL+usdtRatePath, bytes.NewReader(body))
	if err != nil {
		return out, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := httpClient.Do(req)
	if err != nil {
		return out, fmt.Errorf("call askmelotto: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, maxBody))
	if err != nil {
		return out, fmt.Errorf("read askmelotto response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return out, fmt.Errorf("askmelotto http status %d", resp.StatusCode)
	}
	var w usdtRateWire
	if err := json.Unmarshal(raw, &w); err != nil {
		return out, fmt.Errorf("decode askmelotto response: %w", err)
	}
	return USDTRateResponse{Code: w.Code, Msg: w.Msg.EN, Date: w.Data.Date, BaseCurrency: w.Data.BaseCurrency, JSONRate: w.Data.JSONRate}, nil
}
