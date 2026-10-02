package agentauth

import (
	"errors"
	"strings"
	"testing"

	"app/pkg/apperr"
)

func TestLoginRequestValidate(t *testing.T) {
	tests := []struct {
		name    string
		req     LoginRequest
		wantErr bool
		wantMsg string
	}{
		{"ครบ", LoginRequest{Username: "agent01", Password: "x"}, false, ""},
		{"ไม่กรอก username", LoginRequest{Password: "x"}, true, "กรุณากรอก username"},
		{"username มีแต่ช่องว่าง", LoginRequest{Username: "   ", Password: "x"}, true, "กรุณากรอก username"},
		{"username ยาวเกิน", LoginRequest{Username: strings.Repeat("a", 101), Password: "x"}, true, "username ยาวเกิน 100 ตัวอักษร"},
		{"ไม่กรอกรหัส", LoginRequest{Username: "agent01"}, true, "กรุณากรอกรหัสผ่าน"},
		{"รหัสยาวเกิน", LoginRequest{Username: "agent01", Password: strings.Repeat("a", 73)}, true, "รหัสผ่านยาวเกิน 72 ตัวอักษร"},
		{"รหัสยาว 72 พอดี", LoginRequest{Username: "agent01", Password: strings.Repeat("a", 72)}, false, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate()
			if !tt.wantErr {
				if err != nil {
					t.Fatalf("ไม่ควร error: %v", err)
				}
				return
			}
			if !errors.Is(err, apperr.ErrValidation) {
				t.Fatalf("ต้องได้ ErrValidation, got %v", err)
			}
			if got := apperr.From(err).MsgTH; got != tt.wantMsg {
				t.Fatalf("msg = %q, want %q", got, tt.wantMsg)
			}
		})
	}
}
