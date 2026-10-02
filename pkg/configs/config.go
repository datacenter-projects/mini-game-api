package configs

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config คือ config ทั้งหมดของระบบ — โหลดครั้งเดียวตอน boot ผ่าน Load()
// ห้ามเรียก os.Getenv ที่อื่นในโปรเจกต์ ให้อ่านจาก configs.Cfg แทน
type Config struct {
	AppEnv string // local | dev | uat | prod

	ServerAddr         string
	ServerReadTimeout  time.Duration
	ServerWriteTimeout time.Duration
	ShutdownTimeout    time.Duration
	TrustedProxies     []string
	CORSAllowOrigins   string

	DB    DBConfig
	Redis RedisConfig

	MigrateOnStart bool
}

type DBConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	Name     string
	SSLMode  string

	// ReadHost ว่าง = ไม่ใช้ read replica
	ReadHost string

	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

type RedisConfig struct {
	Addr     string
	Password string
	DB       int
}

// Cfg เป็น global ให้ทุก layer อ่านได้ (แบบเดียวกับ database.DBConn)
var Cfg *Config

// Load อ่าน env ทั้งหมดและ validate — key ที่จำเป็นขาด = boot ไม่ขึ้น (fail fast)
func Load() error {
	var missing []string
	required := func(key string) string {
		v := os.Getenv(key)
		if v == "" {
			missing = append(missing, key)
		}
		return v
	}

	cfg := &Config{
		AppEnv:             getEnv("APP_ENV", "local"),
		ServerAddr:         getEnv("SERVER_ADDR", ":8181"),
		ServerReadTimeout:  getEnvDuration("SERVER_READ_TIMEOUT", 30*time.Second),
		ServerWriteTimeout: getEnvDuration("SERVER_WRITE_TIMEOUT", 30*time.Second),
		ShutdownTimeout:    getEnvDuration("SHUTDOWN_TIMEOUT", 30*time.Second),
		TrustedProxies:     getEnvList("TRUSTED_PROXIES"),
		CORSAllowOrigins:   getEnv("CORS_ALLOW_ORIGINS", "*"),

		DB: DBConfig{
			Host:            required("DB_HOST"),
			Port:            getEnvInt("DB_PORT", 5432),
			User:            required("DB_USER"),
			Password:        os.Getenv("DB_PASSWORD"),
			Name:            required("DB_NAME"),
			SSLMode:         getEnv("DB_SSLMODE", "require"),
			ReadHost:        os.Getenv("DB_READ_HOST"),
			MaxOpenConns:    getEnvInt("DB_MAX_OPEN_CONNS", 50),
			MaxIdleConns:    getEnvInt("DB_MAX_IDLE_CONNS", 10),
			ConnMaxLifetime: getEnvDuration("DB_CONN_MAX_LIFETIME", 30*time.Minute),
		},
		Redis: RedisConfig{
			Addr:     required("REDIS_ADDR"),
			Password: os.Getenv("REDIS_PASSWORD"),
			DB:       getEnvInt("REDIS_DB", 0),
		},

		MigrateOnStart: getEnvBool("MIGRATE_ON_START", false),
	}

	if len(missing) > 0 {
		return fmt.Errorf("missing required env: %s", strings.Join(missing, ", "))
	}
	Cfg = cfg
	return nil
}

func (c *Config) IsProd() bool { return c.AppEnv == "prod" }

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getEnvInt(key string, def int) int {
	if v, err := strconv.Atoi(os.Getenv(key)); err == nil {
		return v
	}
	return def
}

func getEnvBool(key string, def bool) bool {
	if v, err := strconv.ParseBool(os.Getenv(key)); err == nil {
		return v
	}
	return def
}

// รับได้ทั้ง "30s" / "5m" และตัวเลขล้วน (ตีความเป็นวินาที)
func getEnvDuration(key string, def time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	if d, err := time.ParseDuration(v); err == nil {
		return d
	}
	if n, err := strconv.Atoi(v); err == nil {
		return time.Duration(n) * time.Second
	}
	return def
}

func getEnvList(key string) []string {
	var out []string
	for _, s := range strings.Split(os.Getenv(key), ",") {
		if s = strings.TrimSpace(s); s != "" {
			out = append(out, s)
		}
	}
	return out
}
