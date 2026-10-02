// migrate รัน goose migration กับ DB ตาม .env
//
//	go run ./scripts/migrate up|down|status|version|redo
//	go run ./scripts/migrate create <module>_<what>   // สร้างไฟล์ .sql ใหม่ใน database/migrations
package main

import (
	"context"
	"fmt"
	"os"

	"app/pkg/configs"
	"app/platform/database"
	"app/platform/logger"

	_ "github.com/joho/godotenv/autoload"

	"github.com/pressly/goose/v3"
)

const migrationDir = "database/migrations"

func main() {
	if len(os.Args) < 2 {
		fmt.Println("usage: go run ./scripts/migrate up|down|status|version|redo|create <name>")
		os.Exit(2)
	}
	cmd := os.Args[1]

	if cmd == "create" {
		if len(os.Args) < 3 {
			fmt.Println("usage: go run ./scripts/migrate create <module>_<what>")
			os.Exit(2)
		}
		goose.SetSequential(false) // ชื่อไฟล์เป็น timestamp กันเลขชนกันเมื่อหลายคนสร้างพร้อมกัน
		if err := goose.Create(nil, migrationDir, os.Args[2], "sql"); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		return
	}

	if err := configs.Load(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	logger.InitLogger(true)
	if err := database.PostgreSQLConnection(configs.Cfg.DB); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	if err := database.Migrate(context.Background(), cmd); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
