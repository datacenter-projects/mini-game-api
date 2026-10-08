package configs

import (
	"encoding/base64"
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

	// MetricsAddr คือ address ของ server /metrics (พอร์ตภายใน แยกจาก API) — ว่าง = ปิด metrics
	MetricsAddr string

	DB      DBConfig
	Redis   RedisConfig
	Auth    AuthConfig
	Account AccountConfig

	MigrateOnStart bool
}

// AccountConfig — docs/modules/account.md (ACC-04)
type AccountConfig struct {
	// APIKeyEncryptionKey — กุญแจ AES-256 (32 byte) สำหรับเข้ารหัส Key ของข้อมูลรับรอง API
	// env API_KEY_ENCRYPTION_KEY เป็น base64 · บังคับใน dev / uat / prod · local ไม่ตั้งได้ (เส้น 1.3 จะใช้ไม่ได้)
	APIKeyEncryptionKey []byte
}

// AuthConfig — ค่าตาม spec docs/modules/agent_auth.md (AUTH-07, AUTH-10, AUTH-11)
// และ docs/modules/agent_auth_phase2.md (AUTH-35, AUTH-46)
type AuthConfig struct {
	JWTSecret string

	// PasswordCost คือ bcrypt cost ของ password/passcode — prod 12, CI/test 4 (ช่วงที่รับ 4–14)
	PasswordCost int

	SessionIdleTimeout     time.Duration
	SessionAbsoluteTimeout time.Duration

	LoginFailLimit     int
	LoginFailWindow    time.Duration
	LoginBlockDuration time.Duration
	LoginIPLimit       int // ครั้ง/นาที ต่อ IP

	PasscodeFailLimit     int
	PasscodeFailWindow    time.Duration
	PasscodeBlockDuration time.Duration
	TempCredentialTTL     time.Duration // อายุค่าชั่วคราวจาก admin reset
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
		MetricsAddr:        getEnvOrEmpty("METRICS_ADDR", ":9090"),

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

		Auth: AuthConfig{
			JWTSecret:              required("JWT_SECRET"),
			PasswordCost:           getEnvInt("PASSWORD_COST", 12),
			SessionIdleTimeout:     getEnvDuration("SESSION_IDLE_TIMEOUT", 60*time.Minute),
			SessionAbsoluteTimeout: getEnvDuration("SESSION_ABSOLUTE_TIMEOUT", 12*time.Hour),
			LoginFailLimit:         getEnvInt("LOGIN_FAIL_LIMIT", 5),
			LoginFailWindow:        getEnvDuration("LOGIN_FAIL_WINDOW", 15*time.Minute),
			LoginBlockDuration:     getEnvDuration("LOGIN_BLOCK_DURATION", 15*time.Minute),
			LoginIPLimit:           getEnvInt("LOGIN_IP_LIMIT_PER_MINUTE", 20),
			PasscodeFailLimit:      getEnvInt("PASSCODE_FAIL_LIMIT", 5),
			PasscodeFailWindow:     getEnvDuration("PASSCODE_FAIL_WINDOW", 24*time.Hour),
			PasscodeBlockDuration:  getEnvDuration("PASSCODE_BLOCK_DURATION", time.Hour),
			TempCredentialTTL:      getEnvDuration("TEMP_CREDENTIAL_TTL", 24*time.Hour),
		},

		MigrateOnStart: getEnvBool("MIGRATE_ON_START", false),
	}

	if err := loadAPIKeyEncryptionKey(cfg, required); err != nil {
		return err
	}

	if len(missing) > 0 {
		return fmt.Errorf("missing required env: %s", strings.Join(missing, ", "))
	}
	if len(cfg.Auth.JWTSecret) < 32 {
		return fmt.Errorf("JWT_SECRET must be at least 32 characters")
	}
	if cfg.Auth.PasswordCost < 4 || cfg.Auth.PasswordCost > 14 {
		return fmt.Errorf("PASSWORD_COST must be between 4 and 14")
	}
	if cfg.MetricsAddr != "" && cfg.MetricsAddr == cfg.ServerAddr {
		return fmt.Errorf("METRICS_ADDR must differ from SERVER_ADDR")
	}
	if cfg.Auth.SessionIdleTimeout > cfg.Auth.SessionAbsoluteTimeout {
		return fmt.Errorf("SESSION_IDLE_TIMEOUT must not exceed SESSION_ABSOLUTE_TIMEOUT")
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

// ต่างจาก getEnv ตรงที่ตั้งเป็นค่าว่างได้ (ไม่ตั้ง key = default, ตั้งเป็นว่าง = ว่าง)
func getEnvOrEmpty(key, def string) string {
	if v, ok := os.LookupEnv(key); ok {
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

// loadAPIKeyEncryptionKey — API_KEY_ENCRYPTION_KEY ต้องเป็น base64 ของ 32 byte (AES-256 — account ACC-04)
// บังคับใน dev / uat / prod · local (รวม CI และ test env) ไม่ตั้งได้
func loadAPIKeyEncryptionKey(cfg *Config, required func(string) string) error {
	var raw string
	switch cfg.AppEnv {
	case "dev", "uat", "prod":
		raw = required("API_KEY_ENCRYPTION_KEY")
	default:
		raw = os.Getenv("API_KEY_ENCRYPTION_KEY")
	}
	if raw == "" {
		return nil
	}
	key, err := base64.StdEncoding.DecodeString(raw)
	if err != nil || len(key) != 32 {
		return fmt.Errorf("API_KEY_ENCRYPTION_KEY must be base64 of exactly 32 bytes")
	}
	cfg.Account.APIKeyEncryptionKey = key
	return nil
}
