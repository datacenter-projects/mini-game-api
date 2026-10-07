package configs

import (
	"fmt"
	"net/url"
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

	DB           DBConfig
	Redis        RedisConfig
	Auth         AuthConfig
	CurrencyRate CurrencyRateConfig

	MigrateOnStart bool
}

// CurrencyRateConfig — docs/modules/currency_rate.md (CR-02, หัวข้อ 6)
type CurrencyRateConfig struct {
	SyncEnabled bool   // IS_CURRENCY_RATE_SYNC (ค่าเริ่มต้น true)
	BaseURL     string // CURRENCY_RATE_API_BASE_URL — origin เปล่า ไม่มี path
}

// ค่าเริ่มต้นของ CURRENCY_RATE_API_BASE_URL (host ฝั่ง dev) — ใช้ได้เฉพาะ local / dev
const defaultCurrencyRateBaseURL = "https://dev-api.amblotto.net"

// AuthConfig — ค่าตาม spec docs/modules/agent_auth.md (AUTH-07, AUTH-10, AUTH-11)
// และ docs/modules/agent_auth_phase2.md (AUTH-35, AUTH-46)
type AuthConfig struct {
	JWTSecret string

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

	if err := loadCurrencyRate(cfg, required); err != nil {
		return err
	}

	if len(missing) > 0 {
		return fmt.Errorf("missing required env: %s", strings.Join(missing, ", "))
	}
	if len(cfg.Auth.JWTSecret) < 32 {
		return fmt.Errorf("JWT_SECRET must be at least 32 characters")
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

// loadCurrencyRate — CURRENCY_RATE_API_BASE_URL ต้องเป็น origin เปล่า (scheme://host[:port]) · บังคับใน uat / prod
// local / dev ไม่ตั้ง = ใช้ค่าเริ่มต้น (host ฝั่ง dev) — docs/modules/currency_rate.md หัวข้อ 6
func loadCurrencyRate(cfg *Config, required func(string) string) error {
	cfg.CurrencyRate.SyncEnabled = getEnvBool("IS_CURRENCY_RATE_SYNC", true)
	var raw string
	switch cfg.AppEnv {
	case "uat", "prod":
		raw = required("CURRENCY_RATE_API_BASE_URL")
		if raw == "" {
			return nil // แจ้งรวมกับ env อื่นที่ขาด
		}
	default:
		raw = getEnv("CURRENCY_RATE_API_BASE_URL", defaultCurrencyRateBaseURL)
	}
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" ||
		u.Path != "" || u.RawQuery != "" || u.Fragment != "" || u.User != nil {
		return fmt.Errorf("CURRENCY_RATE_API_BASE_URL must be a bare origin like https://dev-api.amblotto.net (no path or trailing slash)")
	}
	cfg.CurrencyRate.BaseURL = raw
	return nil
}
