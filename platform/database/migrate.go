package database

import (
	"context"
	"fmt"

	"app/database/migrations"

	"github.com/pressly/goose/v3"
)

// Migrate รัน goose migration ทั้งหมดที่ยังไม่ได้รัน (ไฟล์ .sql ใน database/migrations ถูก embed เข้า binary)
// command: "up" | "down" | "status" | "version" | "redo"
func Migrate(ctx context.Context, command string) error {
	if DBConn == nil {
		return fmt.Errorf("postgres not connected")
	}
	sqlDB, err := DBConn.DB()
	if err != nil {
		return err
	}

	goose.SetBaseFS(migrations.FS)
	if err := goose.SetDialect("postgres"); err != nil {
		return err
	}
	return goose.RunContext(ctx, command, sqlDB, ".")
}
