// seed_asset_global อัปรูปค่ากลาง (GLOBAL) ของทุกช่องจากไฟล์ที่ทีมเกม / ออกแบบส่งมา (docs/modules/asset.md AST-61)
//
//	go run ./scripts/seed_asset_global -dir ./global-assets
//	go run ./scripts/seed_asset_global -dir ./global-assets -force
//
// โครงโฟลเดอร์: {dir}/{game_code}/{asset_type}.{png|jpg|jpeg|gif|webp} เช่น global-assets/scratch_card/icon_button.png
// ชื่อไฟล์ต้องตรงชื่อช่องใน catalog ทุกตัวอักษร · ตรวจชนิด / ขนาดไฟล์ / ขนาดภาพเหมือน API (AST-14–17)
// ช่องที่ GLOBAL มีรูปแล้วถูกข้าม (รันซ้ำได้) · -force = อัปเวอร์ชันใหม่ + ตั้งเป็นรูปที่ใช้งาน (ไฟล์เดิมยังอยู่ในคลัง)
// จบด้วยรายการช่องที่ GLOBAL ยังไม่มีรูป — ต้องว่างก่อน UAT (AST-24) · ต้องตั้ง S3_BUCKET (และ credential ของ AWS) ก่อนรัน
// created_by = script:{user ของเครื่องที่รัน}
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/user"
	"time"

	assetService "app/app/service/asset"
	"app/pkg/configs"
	"app/pkg/utils"
	"app/platform/database"
	"app/platform/logger"
	"app/platform/storage"

	_ "github.com/joho/godotenv/autoload"
)

func main() {
	dir := flag.String("dir", "", "โฟลเดอร์ที่มี {game_code}/{asset_type}.{ext}")
	force := flag.Bool("force", false, "อัปทับช่องที่ GLOBAL มีรูปแล้ว")
	flag.Parse()

	if err := run(*dir, *force); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run(dir string, force bool) error {
	if dir == "" {
		return fmt.Errorf("ต้องระบุ -dir")
	}
	if err := configs.Load(); err != nil {
		return err
	}
	utils.SetPasswordCost(configs.Cfg.Auth.PasswordCost)
	logger.InitLogger(true)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	if err := database.PostgreSQLConnection(configs.Cfg.DB); err != nil {
		return err
	}
	defer database.ClosePostgreSQL() //nolint:errcheck
	if err := storage.Init(ctx, configs.Cfg.Storage); err != nil {
		return err
	}
	if !storage.Available() { // ไม่พร้อม = หยุดตั้งแต่ต้น ไม่ไล่อัปแล้วล้มทุกไฟล์ (505801)
		return fmt.Errorf("storage ยังไม่พร้อม: ตั้ง S3_BUCKET (และ S3_ENDPOINT ถ้าใช้ MinIO) พร้อม credential ของ AWS")
	}

	osUser := "unknown"
	if u, err := user.Current(); err == nil {
		osUser = u.Username
	}
	res, err := assetService.SeedGlobalService(ctx, dir, force, assetService.SeedCreatedBy(osUser), os.Stdout)
	if err != nil {
		return err
	}

	fmt.Printf("\nอัปแล้ว %d · ข้าม %d · ล้มเหลว %d\n", res.Uploaded, res.Skipped, res.Failed)
	if len(res.Gaps) == 0 {
		fmt.Println("ทุกช่องมีรูปค่ากลางครบแล้ว")
	} else {
		fmt.Printf("ช่องที่ GLOBAL ยังไม่มีรูป (%d ช่อง — ต้องว่างก่อน UAT):\n", len(res.Gaps))
		for _, g := range res.Gaps {
			fmt.Printf("  %s/%s\n", g.GameCode, g.AssetType)
		}
	}
	if res.Failed > 0 {
		return fmt.Errorf("%d ไฟล์ล้มเหลว", res.Failed)
	}
	return nil
}
