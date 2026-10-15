package configs

import (
	"errors"
	"os"

	"github.com/joho/godotenv"
)

// LoadTestEnvFile โหลดไฟล์ env ของ test env (.env.test) เข้า environment — ใช้โดย pkg/testutil
//
// ถ้า DB_HOST ตั้งไว้แล้ว (เช่น CI ตั้ง env ใน workflow) ไม่โหลดไฟล์เลยทั้งไฟล์
// เพื่อไม่ให้ค่าบางตัวในไฟล์ (เช่น DB_PORT ของเครื่อง local) หลุดไปผสมกับ env ของ CI
// ไฟล์ไม่มีอยู่ = ใช้ env ที่ตั้งไว้อย่างเดียว · ค่าที่ตั้งไว้แล้วไม่ถูกทับ (godotenv.Load)
func LoadTestEnvFile(path string) error {
	if os.Getenv("DB_HOST") != "" {
		return nil
	}
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return godotenv.Load(path)
}
