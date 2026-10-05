package configs

import (
	"os"
	"path/filepath"
	"testing"
)

func writeEnvFile(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), ".env.test")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

// unsetEnv ลบ env ระหว่าง test แล้วคืนค่าเดิมตอนจบ (t.Setenv ตั้งเป็นค่าว่างได้แต่ลบไม่ได้)
func unsetEnv(t *testing.T, keys ...string) {
	t.Helper()
	for _, k := range keys {
		old, had := os.LookupEnv(k)
		if err := os.Unsetenv(k); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			if had {
				_ = os.Setenv(k, old)
			} else {
				_ = os.Unsetenv(k)
			}
		})
	}
}

func TestLoadTestEnvFileLoadsWhenEnvEmpty(t *testing.T) {
	unsetEnv(t, "DB_HOST", "DB_PORT", "DB_NAME")
	t.Setenv("DB_NAME", "from_env") // ตั้งไว้แล้วต้องไม่ถูกทับ
	path := writeEnvFile(t, "DB_HOST=localhost\nDB_PORT=55432\nDB_NAME=from_file\n")

	if err := LoadTestEnvFile(path); err != nil {
		t.Fatal(err)
	}
	if os.Getenv("DB_PORT") != "55432" || os.Getenv("DB_HOST") != "localhost" {
		t.Fatalf("ต้องโหลดค่าจากไฟล์ ได้ DB_HOST=%q DB_PORT=%q", os.Getenv("DB_HOST"), os.Getenv("DB_PORT"))
	}
	if os.Getenv("DB_NAME") != "from_env" {
		t.Fatalf("ค่าที่ตั้งไว้แล้วต้องไม่ถูกทับ ได้ %q", os.Getenv("DB_NAME"))
	}
}

func TestLoadTestEnvFileSkipsWhenDBHostSet(t *testing.T) { // แบบ CI
	unsetEnv(t, "DB_PORT")
	t.Setenv("DB_HOST", "ci-host")
	path := writeEnvFile(t, "DB_HOST=localhost\nDB_PORT=55432\n")

	if err := LoadTestEnvFile(path); err != nil {
		t.Fatal(err)
	}
	if _, set := os.LookupEnv("DB_PORT"); set {
		t.Fatal("มี DB_HOST แล้วต้องไม่โหลดค่าใดจากไฟล์ (DB_PORT ต้องว่าง)")
	}
}

func TestLoadTestEnvFileMissingFile(t *testing.T) {
	unsetEnv(t, "DB_HOST")
	if err := LoadTestEnvFile(filepath.Join(t.TempDir(), "nope")); err != nil {
		t.Fatalf("ไม่มีไฟล์ต้องไม่ error: %v", err)
	}
}
