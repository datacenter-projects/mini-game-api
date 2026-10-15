// currency_rate_sync เติม / เขียนทับเรทรายเดือนจาก askmelotto ด้วยมือ (docs/modules/currency_rate.md CR-17)
// ใช้ service เดียวกับ job `currency_rate_sync` (SyncCurrencyRateService) — ตรวจ response / ปัด 8 ตำแหน่ง / lock เหมือนกันทุกอย่าง
//
//	go run ./scripts/currency_rate_sync -month 2026-10            # เติมเฉพาะสกุลที่ขาด (ไม่ทับของเดิม)
//	go run ./scripts/currency_rate_sync -month 2026-10 -force     # เขียนทับทั้งเดือน (ต้องขอ lead บน env ร่วม)
//	go run ./scripts/currency_rate_sync                           # เดือนปัจจุบัน (เวลาไทย) เติมเฉพาะที่ขาด
//
// ขอ askmelotto ด้วยวันที่ 1 ของเดือนที่ระบุ · -month ต้องเป็นเดือนที่ askmelotto มีเรท
// env ร่วม (APP_ENV = dev / uat / prod) ต้องขอ lead ก่อนรัน · -force บน env ร่วมต้องใส่ -yes ยืนยันอีกชั้น
// ต้องมี env ตาม .env (DB / Redis / ASKMELOTTO_API_BASE_URL) — สคริปต์พิมพ์ env และ DB host ก่อนทำงานทุกครั้ง
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	currencyRateCore "app/app/core/currency_rate"
	currencyRateService "app/app/service/currency_rate"
	"app/pkg/configs"
	"app/platform/database"
	"app/platform/logger"

	_ "github.com/joho/godotenv/autoload"
)

func main() {
	month := flag.String("month", "", "เดือนที่จะ sync รูปแบบ YYYY-MM (ว่าง = เดือนปัจจุบันตามเวลาไทย)")
	force := flag.Bool("force", false, "เขียนทับทั้งเดือน (ไม่ใส่ = เติมเฉพาะสกุลที่ขาด)")
	yes := flag.Bool("yes", false, "ยืนยัน -force บน env ร่วม (dev / uat / prod)")
	flag.Parse()

	if err := run(*month, *force, *yes); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run(monthArg string, force, yes bool) error {
	opts := currencyRateService.SyncOptions{Force: force}
	if monthArg != "" {
		m, ok := currencyRateCore.ParseMonth(monthArg)
		if !ok {
			return fmt.Errorf("-month ต้องเป็นรูปแบบ YYYY-MM เช่น 2026-10 (ได้ %q)", monthArg)
		}
		opts.Month = m
	}
	if err := configs.Load(); err != nil {
		return err
	}
	logger.InitLogger(true)

	shared := configs.Cfg.AppEnv != "local"
	fmt.Printf("env=%s db=%s/%s askmelotto=%s force=%v\n", configs.Cfg.AppEnv, configs.Cfg.DB.Host, configs.Cfg.DB.Name, configs.Cfg.Askmelotto.BaseURL, force)
	if shared && force && !yes {
		return errors.New("-force บน env ร่วมต้องใส่ -yes ด้วย (และขอ lead ก่อน)")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	if err := database.PostgreSQLConnection(configs.Cfg.DB); err != nil {
		return err
	}
	if err := database.RedisConnection(ctx, configs.Cfg.Redis); err != nil {
		return err
	}

	res, err := currencyRateService.SyncCurrencyRateService(ctx, time.Now(), opts)
	month := res.Month.Format("2006-01")
	switch {
	case res.Skipped:
		return errors.New("มี job / คำสั่งอื่นกำลังรันอยู่ (lock) — ลองใหม่ภายหลัง")
	case res.AlreadyComplete:
		fmt.Printf("เดือน %s ครบ 27 สกุลแล้ว ไม่ได้ยิง askmelotto (ใช้ -force ถ้าต้องการเขียนทับ)\n", month)
		return nil
	}
	if len(res.Rejected) > 0 {
		fmt.Printf("สกุลที่ค่าผิด (ไม่บันทึก): %s\n", strings.Join(res.Rejected, ","))
	}
	fmt.Printf("เดือน %s: บันทึก %d แถว\n", month, res.Saved)
	if err != nil {
		if len(res.Missing) > 0 {
			fmt.Printf("ยังขาด: %s\n", strings.Join(res.Missing, ","))
		}
		return err
	}
	fmt.Printf("เดือน %s ครบ 27 สกุลแล้ว\n", month)
	return nil
}
