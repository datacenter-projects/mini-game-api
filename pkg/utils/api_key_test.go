package utils

import (
	"regexp"
	"testing"
)

var hex64 = regexp.MustCompile(`^[0-9a-f]{64}$`)

func TestGenerateAPIKey(t *testing.T) { // ACC-03
	a, err := GenerateAPIKey()
	if err != nil {
		t.Fatal(err)
	}
	b, _ := GenerateAPIKey()
	if !hex64.MatchString(a) || !hex64.MatchString(b) {
		t.Fatalf("ต้องเป็น hex ตัวเล็ก 64 ตัว: %q %q", a, b)
	}
	if a == b {
		t.Fatal("สุ่มสองครั้งได้ค่าเดียวกัน")
	}
}
