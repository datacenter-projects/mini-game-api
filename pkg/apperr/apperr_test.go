package apperr

import (
	"errors"
	"fmt"
	"net/http"
	"testing"
)

func TestWrapKeepsIdentity(t *testing.T) {
	cause := errors.New("db timeout")
	wrapped := fmt.Errorf("service: %w", ErrConflict.Wrap(cause))

	if !errors.Is(wrapped, ErrConflict) {
		t.Fatal("errors.Is ต้องจับ ErrConflict ได้แม้ถูก Wrap ซ้อน")
	}
	if !errors.Is(wrapped, cause) {
		t.Fatal("errors.Is ต้องจับ cause เดิมได้")
	}
	if errors.Is(wrapped, ErrNotFound) {
		t.Fatal("ต้องไม่ match code อื่น")
	}
	if ErrConflict.Unwrap() != nil {
		t.Fatal("Wrap ต้องไม่แก้ตัวแปร global")
	}
}

func TestFrom(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		wantCode int
	}{
		{"apperr ตรงๆ", ErrForbidden, 403},
		{"apperr ที่ถูก fmt wrap", fmt.Errorf("x: %w", ErrValidation), 422},
		{"error ธรรมดา → internal", errors.New("boom"), 500},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := From(tt.err).Code; got != tt.wantCode {
				t.Fatalf("code = %d, want %d", got, tt.wantCode)
			}
		})
	}
	if From(errors.New("boom")).HTTPStatus != http.StatusInternalServerError {
		t.Fatal("error ที่ไม่รู้จักต้องได้ HTTP 500")
	}
}

func TestDuplicateCodePanics(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("ประกาศ code ซ้ำต้อง panic")
		}
	}()
	New(400, "ซ้ำ", "dup")
}
