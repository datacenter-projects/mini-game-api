package utils

import (
	"bytes"
	"regexp"
	"strings"
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

func TestHashAPIKey(t *testing.T) { // ACC-04
	h := HashAPIKey("abc")
	if h != "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad" {
		t.Fatalf("sha256 ไม่ตรง: %s", h)
	}
}

func TestEncryptDecryptAPIKey(t *testing.T) { // ACC-04
	encKey := []byte(strings.Repeat("k", 32))
	key, _ := GenerateAPIKey()

	c1, err := EncryptAPIKey(encKey, key)
	if err != nil {
		t.Fatal(err)
	}
	c2, _ := EncryptAPIKey(encKey, key)
	if bytes.Equal(c1, c2) {
		t.Fatal("เข้ารหัสสองครั้งต้องได้ค่าต่างกัน (nonce สุ่ม)")
	}
	if bytes.Contains(c1, []byte(key)) {
		t.Fatal("ciphertext มี Key แบบ plain")
	}
	got, err := DecryptAPIKey(encKey, c1)
	if err != nil || got != key {
		t.Fatalf("ถอดรหัสได้ %q err %v", got, err)
	}

	tests := []struct {
		name   string
		encKey []byte
		data   []byte
	}{
		{"กุญแจผิด", []byte(strings.Repeat("x", 32)), c1},
		{"กุญแจยาวไม่ใช่ 32", []byte("short"), c1},
		{"ข้อมูลสั้นเกิน", encKey, []byte("abc")},
		{"ข้อมูลถูกแก้", encKey, append(append([]byte{}, c1[:len(c1)-1]...), c1[len(c1)-1]^1)},
	}
	for _, tt := range tests {
		if _, err := DecryptAPIKey(tt.encKey, tt.data); err == nil {
			t.Errorf("%s: ต้อง error", tt.name)
		}
	}
	if _, err := EncryptAPIKey([]byte("short"), key); err == nil {
		t.Error("กุญแจยาวไม่ใช่ 32 ต้อง error")
	}
}
