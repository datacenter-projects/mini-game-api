package currencyrate

import (
	"context"
	"errors"

	currencyRateCore "app/app/core/currency_rate"
	redisRepo "app/app/repository/redis"
)

// ErrRateUnavailable — ยังไม่มีเรทของสกุลที่ต้องใช้ (CR-14) · หน้าที่เรียกตัดสินเองว่าจะแสดงอย่างไร
var ErrRateUnavailable = errors.New("currency rate unavailable")

// ConvertAmountService แปลงยอด (หน่วยย่อย 1/100) จากสกุล from ไปสกุล to ด้วยเรทล่าสุด (CR-13)
// ใช้แสดงผลเท่านั้น ห้ามใช้ย้ายเงิน
func ConvertAmountService(ctx context.Context, amount int64, from, to string) (int64, error) {
	if from == to {
		return amount, nil
	}
	rates, err := redisRepo.GetCurrencyRatesRepository(ctx)
	if err != nil {
		return 0, err
	}
	rf, okFrom := rates[from]
	rt, okTo := rates[to]
	if !okFrom || !okTo || !currencyRateCore.IsSupported(from) || !currencyRateCore.IsSupported(to) {
		return 0, ErrRateUnavailable
	}
	return currencyRateCore.ConvertMinor(amount, rf, rt), nil
}
