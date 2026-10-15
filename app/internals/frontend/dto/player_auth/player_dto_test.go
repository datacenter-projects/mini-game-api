package playerauth

import (
	"errors"
	"strings"
	"testing"

	"app/pkg/apperr"
)

func TestLoginRequestValidate(t *testing.T) {
	tests := []struct {
		name string
		req  LoginRequest
		ok   bool
	}{
		{"ปกติ", LoginRequest{Username: "mem01", Password: "x"}, true},
		{"ไม่ส่ง username", LoginRequest{Password: "x"}, false},
		{"username ว่างช่องว่าง", LoginRequest{Username: "   ", Password: "x"}, false},
		{"username ยาวเกิน", LoginRequest{Username: strings.Repeat("a", 101), Password: "x"}, false},
		{"ไม่ส่ง password", LoginRequest{Username: "mem01"}, false},
		{"password เกิน 72 byte", LoginRequest{Username: "mem01", Password: strings.Repeat("a", 73)}, false},
	}
	for _, tt := range tests {
		err := tt.req.Validate()
		if (err == nil) != tt.ok {
			t.Errorf("%s: err = %v want ok = %v", tt.name, err, tt.ok)
		}
		if err != nil && !errors.Is(err, apperr.ErrValidation) {
			t.Errorf("%s: ต้องเป็น ErrValidation: %v", tt.name, err)
		}
	}
}

func TestPasswordUpdateRequestValidate(t *testing.T) { // rule: PA-50
	tests := []struct {
		name string
		req  PasswordUpdateRequest
		ok   bool
		msg  string // ข้อความต้องบอก field
	}{
		{"ปกติ", PasswordUpdateRequest{OldPassword: "old", NewPassword: "aA4b4c4d4e4f"}, true, ""},
		{"ไม่ส่ง old", PasswordUpdateRequest{NewPassword: "aA4b4c4d4e4f"}, false, "old_password"},
		{"old เกิน 72", PasswordUpdateRequest{OldPassword: strings.Repeat("a", 73), NewPassword: "aA4b4c4d4e4f"}, false, "old_password"},
		{"ไม่ส่ง new", PasswordUpdateRequest{OldPassword: "old"}, false, "new_password"},
		{"new สั้นเกิน", PasswordUpdateRequest{OldPassword: "old", NewPassword: "aA1"}, false, "new_password"},
		{"new ไม่มีตัวเลข", PasswordUpdateRequest{OldPassword: "old", NewPassword: "abcdefghij"}, false, "new_password"},
		{"new มีช่องว่าง", PasswordUpdateRequest{OldPassword: "old", NewPassword: "aA4b 4c4d4e"}, false, "new_password"},
		{"new ซ้ำตัวเดิม 4 ตัว", PasswordUpdateRequest{OldPassword: "old", NewPassword: "aaaa1234Ab"}, false, "new_password"},
	}
	for _, tt := range tests {
		err := tt.req.Validate()
		if (err == nil) != tt.ok {
			t.Errorf("%s: err = %v want ok = %v", tt.name, err, tt.ok)
			continue
		}
		var ae *apperr.Error
		if err != nil && (!errors.As(err, &ae) || !strings.Contains(ae.MsgTH, tt.msg) || !strings.Contains(ae.MsgEN, tt.msg)) {
			t.Errorf("%s: ข้อความต้องบอก field %q: %v", tt.name, tt.msg, err)
		}
	}
}
