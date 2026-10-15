//go:build integration

package testutil

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"app/pkg/configs"
)

// FakeAskmelotto คือ askmelotto จำลอง (เส้น POST /api/ex/rate/usdt) สำหรับ test ของ currency_rate — ชี้ configs.Cfg.Askmelotto.BaseURL มาที่ server นี้
// ตั้งค่าได้ทุกชั้นของ response เพื่อทดสอบกรณีผิด: code · base · date · เรทรายสกุล (ข้อความ JSON ดิบ) · HTTP status · body ดิบ
type FakeAskmelotto struct {
	URL string

	mu    sync.Mutex
	dates []string // date ที่ถูกขอ (ตามลำดับ)

	hits atomic.Int32

	// ตั้งได้ก่อนยิง (ใช้ผ่าน Set เพื่อกัน race)
	cfg FakeAskmelottoConfig
}

type FakeAskmelottoConfig struct {
	Code    int               // 0 = 200
	Msg     string            // msg.en
	Base    string            // "" = USDT
	Date    string            // "" = สะท้อน date ที่ขอ
	Rates   map[string]string // สกุล → ข้อความ JSON ดิบของตัวเลข (เช่น `33.62` · `"33.62"` · `null`) · nil = ชุดเรทปกติ
	Status  int               // 0 = 200
	RawBody string            // ไม่ว่าง = ตอบ body นี้ตรงๆ
}

// DefaultAskmelottoRates — เรทปกติครบ 27 สกุล (ค่าเดียวกับที่ยิงจริง 2026-10-07) ข้อความ JSON ดิบ
func DefaultAskmelottoRates() map[string]string {
	return map[string]string{
		"THB": "33.62", "MMK": "2099.53", "CNY": "6.7", "EUR": "0.88", "GBP": "0.75", "IDR": "17849.27", "INR": "96.37",
		"JPY": "158.15", "KHR": "4046.24", "KRW": "1337.9", "LAK": "22394.92", "PHP": "62.77", "USD": "0.99", "VND": "25988.07",
		"AUD": "1.43", "TWD": "31.78", "MYR": "4.08", "HKD": "7.84", "MXN": "17.97", "USDT": "1", "BRL": "4.98", "BDT": "122.75",
		"PKR": "276.6", "CLP": "973.67", "ARS": "1520.54", "BOB": "11.5", "NGN": "1324.42",
	}
}

// NewFakeAskmelotto เปิด server จำลองและตั้ง configs.Cfg.Askmelotto.BaseURL ให้ชี้มา (test จบ server ปิดเอง)
func NewFakeAskmelotto(t *testing.T) *FakeAskmelotto {
	t.Helper()
	f := &FakeAskmelotto{}
	srv := httptest.NewServer(http.HandlerFunc(f.handle))
	t.Cleanup(srv.Close)
	f.URL = srv.URL
	configs.Cfg.Askmelotto.BaseURL = srv.URL
	return f
}

// Set แก้การตอบของ server (เรียกซ้ำได้ระหว่าง test)
func (f *FakeAskmelotto) Set(apply func(c *FakeAskmelottoConfig)) {
	f.mu.Lock()
	defer f.mu.Unlock()
	apply(&f.cfg)
}

// Reset ล้างการตั้งค่ากลับเป็นการตอบปกติ
func (f *FakeAskmelotto) Reset() {
	f.Set(func(c *FakeAskmelottoConfig) { *c = FakeAskmelottoConfig{} })
}

// Hits คือจำนวนครั้งที่ถูกเรียก · Dates คือ date ที่ถูกขอตามลำดับ
func (f *FakeAskmelotto) Hits() int { return int(f.hits.Load()) }

func (f *FakeAskmelotto) Dates() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.dates...)
}

func (f *FakeAskmelotto) handle(w http.ResponseWriter, r *http.Request) {
	f.hits.Add(1)
	var req struct {
		Date string `json:"date"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)

	f.mu.Lock()
	f.dates = append(f.dates, req.Date)
	c := f.cfg
	f.mu.Unlock()

	if r.Method != http.MethodPost || r.URL.Path != "/api/ex/rate/usdt" {
		http.NotFound(w, r)
		return
	}
	if c.Status != 0 && c.Status != http.StatusOK {
		http.Error(w, "boom", c.Status)
		return
	}
	if c.RawBody != "" {
		_, _ = w.Write([]byte(c.RawBody))
		return
	}
	code := c.Code
	if code == 0 {
		code = 200
	}
	base := c.Base
	if base == "" {
		base = "USDT"
	}
	date := c.Date
	if date == "" {
		date = req.Date
	}
	rates := c.Rates
	if rates == nil {
		rates = DefaultAskmelottoRates()
	}
	keys := make([]string, 0, len(rates))
	for k := range rates {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var rb strings.Builder
	for i, k := range keys {
		if i > 0 {
			rb.WriteString(",")
		}
		fmt.Fprintf(&rb, "%q:%s", k, rates[k])
	}
	w.Header().Set("Content-Type", "application/json")
	if code != 200 {
		fmt.Fprintf(w, `{"code":%d,"msg":{"en":%q,"th":"x"}}`, code, c.Msg)
		return
	}
	fmt.Fprintf(w, `{"code":200,"data":{"date":%q,"base_currency":%q,"json_rate":{%s}},"msg":{"en":%q,"th":"สำเร็จ"}}`, date, base, rb.String(), c.Msg)
}
