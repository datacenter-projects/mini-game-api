package agentauth

import (
	"errors"
	"testing"

	"app/pkg/apperr"
	"app/pkg/utils"

	"github.com/goccy/go-json"
)

// parse ใช้ JSON decoder ตัวเดียวกับ Fiber (configs.FiberConfig)
func parse(t *testing.T, body string, out utils.Validatable) error {
	t.Helper()
	if err := json.Unmarshal([]byte(body), out); err != nil {
		t.Fatalf("body ต้อง parse ผ่าน: %v", err)
	}
	return out.Validate()
}

func expectMsg(t *testing.T, err error, want string) {
	t.Helper()
	if want == "" {
		if err != nil {
			t.Fatalf("ไม่ควร error: %v", err)
		}
		return
	}
	if !errors.Is(err, apperr.ErrValidation) {
		t.Fatalf("ต้องได้ ErrValidation, got %v", err)
	}
	if got := apperr.From(err).MsgTH; got != want {
		t.Fatalf("msg = %q, want %q", got, want)
	}
}

func TestSetupPasscodeRequestValidate(t *testing.T) { // AUTH-30, AUTH-32
	tests := []struct{ name, body, want string }{
		{"ครบ", `{"passcode":"012345","confirm_passcode":"012345"}`, ""},
		{"ไม่กรอก", `{"confirm_passcode":"012345"}`, "กรุณากรอก passcode"},
		{"ส่งเป็น number", `{"passcode":12345,"confirm_passcode":"012345"}`, "passcode ต้องส่งเป็นข้อความ (string)"},
		{"5 หลัก", `{"passcode":"12345","confirm_passcode":"12345"}`, "passcode ต้องเป็นตัวเลข 6 หลัก"},
		{"มีตัวอักษร", `{"passcode":"12345a","confirm_passcode":"12345a"}`, "passcode ต้องเป็นตัวเลข 6 หลัก"},
		{"ยืนยันไม่ตรง", `{"passcode":"123456","confirm_passcode":"654321"}`, "confirm_passcode ไม่ตรงกับ passcode"},
		{"ยืนยันเป็น number", `{"passcode":"123456","confirm_passcode":123456}`, "confirm_passcode ต้องส่งเป็นข้อความ (string)"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expectMsg(t, parse(t, tt.body, &SetupPasscodeRequest{}), tt.want)
		})
	}
}

func TestChangePasscodeRequestValidate(t *testing.T) { // AUTH-34
	tests := []struct{ name, body, want string }{
		{"ครบ", `{"old_passcode":"123456","new_passcode":"654321","confirm_passcode":"654321"}`, ""},
		{"ไม่กรอกของเดิม", `{"new_passcode":"654321","confirm_passcode":"654321"}`, "กรุณากรอก old_passcode"},
		{"ของใหม่ผิดรูปแบบ", `{"old_passcode":"123456","new_passcode":"65432","confirm_passcode":"65432"}`, "new_passcode ต้องเป็นตัวเลข 6 หลัก"},
		{"ยืนยันไม่ตรง", `{"old_passcode":"123456","new_passcode":"654321","confirm_passcode":"111111"}`, "confirm_passcode ไม่ตรงกับ new_passcode"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expectMsg(t, parse(t, tt.body, &ChangePasscodeRequest{}), tt.want)
		})
	}
}

func TestChangePasswordRequestValidate(t *testing.T) { // AUTH-36, AUTH-37
	tests := []struct{ name, body, want string }{
		{"ครบ", `{"old_password":"x","new_password":"Zx9!kq2m","confirm_password":"Zx9!kq2m"}`, ""},
		{"ไม่กรอกรหัสเดิม", `{"new_password":"Zx9!kq2m","confirm_password":"Zx9!kq2m"}`, "กรุณากรอก old_password"},
		{"ไม่กรอกรหัสใหม่", `{"old_password":"x"}`, "กรุณากรอก new_password"},
		{"สั้น", `{"old_password":"x","new_password":"Zx9!kq2","confirm_password":"Zx9!kq2"}`, "new_password ต้องยาวอย่างน้อย 8 ตัว"},
		{"ช่องว่าง", `{"old_password":"x","new_password":"Zx9 kq2m","confirm_password":"Zx9 kq2m"}`, "new_password ใช้ได้เฉพาะ a-z, A-Z, 0-9 และอักขระพิเศษ ห้ามมีช่องว่าง"},
		{"ไม่มีตัวเลข", `{"old_password":"x","new_password":"Zxkqmwpt","confirm_password":"Zxkqmwpt"}`, "new_password ต้องมีตัวเลขอย่างน้อย 1 ตัว"},
		{"ไม่มีตัวอักษร", `{"old_password":"x","new_password":"92837465","confirm_password":"92837465"}`, "new_password ต้องมีตัวอักษรอย่างน้อย 1 ตัว"},
		{"ซ้ำ 4", `{"old_password":"x","new_password":"Zx9aaaam","confirm_password":"Zx9aaaam"}`, "new_password ห้ามมีตัวเดียวกันซ้ำติดกัน 4 ตัวขึ้นไป"},
		{"ยืนยันไม่ตรง", `{"old_password":"x","new_password":"Zx9!kq2m","confirm_password":"Zx9!kq2n"}`, "confirm_password ไม่ตรงกับ new_password"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expectMsg(t, parse(t, tt.body, &ChangePasswordRequest{}), tt.want)
		})
	}
}
