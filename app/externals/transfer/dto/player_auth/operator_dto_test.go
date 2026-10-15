package playerauth

import (
	"errors"
	"strings"
	"testing"

	"app/pkg/apperr"
)

const goodKey = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

func TestOperatorCredsValidate(t *testing.T) { // rule: PA-10
	tests := []struct {
		name string
		in   OperatorCreds
		ok   bool
	}{
		{"ปกติ", OperatorCreds{AgentUsername: "agent01", Key: goodKey}, true},
		{"ไม่ส่ง agent_username", OperatorCreds{Key: goodKey}, false},
		{"agent_username ว่างช่องว่าง", OperatorCreds{AgentUsername: "  ", Key: goodKey}, false},
		{"agent_username ยาวเกิน", OperatorCreds{AgentUsername: strings.Repeat("a", 101), Key: goodKey}, false},
		{"ไม่ส่ง key", OperatorCreds{AgentUsername: "agent01"}, false},
		{"key ยาวผิดปกติ", OperatorCreds{AgentUsername: "agent01", Key: strings.Repeat("a", 201)}, false},
	}
	for _, tt := range tests {
		err := tt.in.Validate()
		if (err == nil) != tt.ok {
			t.Errorf("%s: err = %v want ok = %v", tt.name, err, tt.ok)
		}
		if err != nil && !errors.Is(err, apperr.ErrValidation) {
			t.Errorf("%s: ต้องเป็น ErrValidation: %v", tt.name, err)
		}
	}
}

func TestTransferLoginRequestValidate(t *testing.T) { // rule: PA-05, PA-08A, L2
	base := TransferLoginRequest{AgentUsername: "agent01", Key: goodKey, PlayerUsername: "p001"}
	with := func(f func(r *TransferLoginRequest)) TransferLoginRequest {
		r := base
		f(&r)
		return r
	}
	tests := []struct {
		name string
		req  TransferLoginRequest
		ok   bool
	}{
		{"ปกติ", base, true},
		{"รับตัวใหญ่", with(func(r *TransferLoginRequest) { r.PlayerUsername = "ABC" }), true},
		{"player สั้นเกิน", with(func(r *TransferLoginRequest) { r.PlayerUsername = "ab" }), false},
		{"player ยาวเกิน", with(func(r *TransferLoginRequest) { r.PlayerUsername = strings.Repeat("a", 33) }), false},
		{"player มีอักขระพิเศษ", with(func(r *TransferLoginRequest) { r.PlayerUsername = "ab_c" }), false},
		{"player มี @", with(func(r *TransferLoginRequest) { r.PlayerUsername = "a@bc" }), false},
		{"player ไม่ส่ง", with(func(r *TransferLoginRequest) { r.PlayerUsername = "" }), false},
		{"currency ไม่ส่ง", with(func(r *TransferLoginRequest) { r.Currency = "" }), true},
		{"currency THB", with(func(r *TransferLoginRequest) { r.Currency = "THB" }), true},
		{"currency สั้นเกิน", with(func(r *TransferLoginRequest) { r.Currency = "TH" }), false},
		{"currency ยาวเกิน", with(func(r *TransferLoginRequest) { r.Currency = "THBXX" }), false},
		{"language th/en", with(func(r *TransferLoginRequest) { r.Language = "en" }), true},
		{"language ผิด", with(func(r *TransferLoginRequest) { r.Language = "fr" }), false},
		{"ไม่ส่ง key", with(func(r *TransferLoginRequest) { r.Key = "" }), false},
	}
	for _, tt := range tests {
		err := tt.req.Validate()
		if (err == nil) != tt.ok {
			t.Errorf("%s: err = %v want ok = %v", tt.name, err, tt.ok)
		}
	}
}

func TestTransferRegisterRequestValidate(t *testing.T) { // rule: PA-08
	base := TransferRegisterRequest{AgentUsername: "agent01", Key: goodKey, PlayerUsername: "p001"}
	with := func(pw string) TransferRegisterRequest {
		r := base
		r.Password = pw
		return r
	}
	tests := []struct {
		name string
		req  TransferRegisterRequest
		ok   bool
	}{
		{"ไม่ส่ง password", base, true},
		{"password ว่าง = ไม่ตั้งรหัส", with(""), true},
		{"password ผ่าน AUTH-36", with("aA4b4c4d4e4f"), true},
		{"password สั้นเกิน", with("aA1"), false},
		{"password ไม่มีตัวเลข", with("abcdefghij"), false},
		{"password มีช่องว่าง", with("aA4b 4c4d4e"), false},
		{"password เกิน 72", with(strings.Repeat("aA1", 30)), false},
	}
	for _, tt := range tests {
		err := tt.req.Validate()
		if (err == nil) != tt.ok {
			t.Errorf("%s: err = %v want ok = %v", tt.name, err, tt.ok)
		}
		var ae *apperr.Error
		if err != nil && strings.HasPrefix(tt.name, "password") && (!errors.As(err, &ae) || !strings.Contains(ae.MsgEN, "password")) {
			t.Errorf("%s: ข้อความต้องบอก field password: %v", tt.name, err)
		}
	}
}
