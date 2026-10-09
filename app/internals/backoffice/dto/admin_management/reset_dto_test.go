package adminmanagement

import (
	"errors"
	"testing"

	"app/pkg/apperr"

	"github.com/goccy/go-json"
)

func TestResetCredentialRequestValidate(t *testing.T) { // AUTH-45
	ok := ResetCredentialRequest{}
	if err := json.Unmarshal([]byte(`{"username":"agent01@staff","passcode":"123456"}`), &ok); err != nil || ok.Validate() != nil {
		t.Fatalf("ครบ ต้องผ่าน: %v", ok.Validate())
	}
	blank := ResetCredentialRequest{}
	_ = json.Unmarshal([]byte(`{"username":"  "}`), &blank)
	err := blank.Validate()
	if !errors.Is(err, apperr.ErrValidation) || apperr.From(err).MsgTH != "กรุณากรอก username" {
		t.Fatalf("username ว่าง: %v", err)
	}
}

func TestAccountSearchRequestValidate(t *testing.T) { // MGMT-27B
	r := AccountSearchRequest{Username: "  COMP01 "}
	if err := r.Validate(); err != nil || r.Username != "comp01" {
		t.Fatalf("ตัดช่องว่าง + ตัวเล็ก: %q %v", r.Username, err)
	}
	if err := (&AccountSearchRequest{}).Validate(); !errors.Is(err, apperr.ErrValidation) {
		t.Fatalf("ว่างต้อง 422: %v", err)
	}
}
