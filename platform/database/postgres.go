package database

import (
	"context"
	"errors"
	"fmt"

	"app/pkg/configs"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
	"gorm.io/plugin/dbresolver"
)

// DBConn เป็น global connection ของ PostgreSQL (ชื่อเดียวกับ askmelotto-api)
//
// กฎ: repository ห้ามอ้าง DBConn ตรงๆ — รับ db *gorm.DB เป็น param แรกเสมอ
// service เป็นคนเลือกว่าจะส่ง database.DBConn (อ่านธรรมดา) หรือ tx (ใน Transaction)
var DBConn *gorm.DB

func dsn(c configs.DBConfig, host string) string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s TimeZone=Asia/Bangkok",
		host, c.Port, c.User, c.Password, c.Name, c.SSLMode)
}

func PostgreSQLConnection(c configs.DBConfig) error {
	logLevel := gormlogger.Warn
	if !configs.Cfg.IsProd() {
		logLevel = gormlogger.Info
	}

	db, err := gorm.Open(postgres.Open(dsn(c, c.Host)), &gorm.Config{
		Logger:                 gormlogger.Default.LogMode(logLevel),
		SkipDefaultTransaction: true, // เปิด tx เองที่ service เท่านั้น
		TranslateError:         true, // ได้ gorm.ErrDuplicatedKey ฯลฯ แทน error string ของ driver
	})
	if err != nil {
		return fmt.Errorf("open postgres: %w", err)
	}

	if c.ReadHost != "" {
		if err := db.Use(dbresolver.Register(dbresolver.Config{
			Replicas: []gorm.Dialector{postgres.Open(dsn(c, c.ReadHost))},
			Policy:   dbresolver.RandomPolicy{},
		})); err != nil {
			return fmt.Errorf("register read replica: %w", err)
		}
	}

	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	sqlDB.SetMaxOpenConns(c.MaxOpenConns)
	sqlDB.SetMaxIdleConns(c.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(c.ConnMaxLifetime)

	if err := sqlDB.Ping(); err != nil {
		return fmt.Errorf("ping postgres: %w", err)
	}
	DBConn = db
	return nil
}

func PingPostgreSQL(ctx context.Context) error {
	if DBConn == nil {
		return errors.New("postgres not connected")
	}
	sqlDB, err := DBConn.DB()
	if err != nil {
		return err
	}
	return sqlDB.PingContext(ctx)
}

func ClosePostgreSQL() error {
	if DBConn == nil {
		return nil
	}
	sqlDB, err := DBConn.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
