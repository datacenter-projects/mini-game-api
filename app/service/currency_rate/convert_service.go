package currencyrate

import (
	"context"
	"errors"
	"fmt"
	"time"

	currencyRateCore "app/app/core/currency_rate"
	currencyRatePostgres "app/app/repository/postgres/currency_rate"
	"app/pkg/apperr"
	"app/platform/logger"

	"gorm.io/gorm"
)

// THBConversion — ผลแปลงเป็นบาท (CR-20): bet เก็บ THBRate + RateMonth + ยอดบาทไว้กับรายการเดิมพัน (C3)
type THBConversion struct {
	AmountTHB float64   // Round4(amount × THBRate) — ปัดครั้งเดียวตอนท้าย ครึ่งขึ้น
	RateMonth time.Time // วันที่ 1 ของเดือนที่ใช้ เป็นเที่ยงคืน UTC ของวันนั้น (ค่าเดียวกับ rate_month ใน repo / model — bet เก็บไปเทียบ / join ได้ตรง)
	THBRate   float64   // 1 หน่วยของสกุลนั้น = ? บาท ของเดือนนั้น
}

// ConvertToTHBService แปลงยอดเป็นบาทด้วยเรทของเดือนของ at (เวลาเกิดรายการ ตามเวลาไทย — CR-21) · ให้ module เดิมพันเช็ค win_limit_per_day
//
//   - THB → คืนทันที ไม่อ่าน DB (THBRate = 1 — CR-23)
//   - ไม่มีเรทของเดือนนั้น / สกุลที่ระบบไม่รู้จัก → apperr.ErrCurrencyRateUnavailable + log warn · **ไม่ใช้เรทเดือนก่อน** (CR-24 / CR-25)
//     ผู้เรียก (bet) ต้องปฏิเสธการแทง (fail closed — lead C1)
//   - amount ต้องเป็นเลขจำกัดและไม่ติดลบ — ผิด = error ภายใน (CR-27)
//   - อ่านแถวเดียวด้วย PK · ใช้ db / tx ของผู้เรียก · ไม่ lock · ไม่มี endpoint (CR-26)
func ConvertToTHBService(ctx context.Context, db *gorm.DB, amount float64, currency string, at time.Time) (THBConversion, error) {
	month := currencyRateCore.MonthDate(at)

	if currency == currencyRateCore.CurrencyTHB {
		thb, ok := currencyRateCore.ConvertTHB(amount, 1)
		if !ok {
			return THBConversion{}, apperr.ErrInternal.Wrap(fmt.Errorf("invalid amount %v", amount))
		}
		return THBConversion{AmountTHB: thb, RateMonth: month, THBRate: 1}, nil
	}
	if _, ok := currencyRateCore.ConvertTHB(amount, 1); !ok { // ตรวจ amount ก่อน เพื่อไม่ให้ amount ผิดไปปนกับ "เรทหาย"
		return THBConversion{}, apperr.ErrInternal.Wrap(fmt.Errorf("invalid amount %v", amount))
	}
	if !currencyRateCore.IsSupported(currency) {
		logger.Ctx(ctx).Warnw("currency rate unavailable: unsupported currency", "currency", currency, "month", month.Format("2006-01"))
		return THBConversion{}, apperr.ErrCurrencyRateUnavailable
	}

	row, err := currencyRatePostgres.GetCurrencyRateRepository(db, month, currency)
	if errors.Is(err, apperr.ErrNotFound) {
		logger.Ctx(ctx).Warnw("currency rate unavailable", "currency", currency, "month", month.Format("2006-01"))
		return THBConversion{}, apperr.ErrCurrencyRateUnavailable
	}
	if err != nil {
		return THBConversion{}, err
	}
	thb, ok := currencyRateCore.ConvertTHB(amount, row.THBRate)
	if !ok {
		return THBConversion{}, apperr.ErrInternal.Wrap(fmt.Errorf("convert %v %s with rate %v", amount, currency, row.THBRate))
	}
	return THBConversion{AmountTHB: thb, RateMonth: month, THBRate: row.THBRate}, nil
}
