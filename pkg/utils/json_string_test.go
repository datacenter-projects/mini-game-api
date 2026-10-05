package utils

import (
	"testing"

	"github.com/goccy/go-json"
)

func TestJSONString(t *testing.T) {
	tests := []struct {
		body              string
		value             string
		present, isString bool
	}{
		{`{"p":"012345"}`, "012345", true, true},
		{`{"p":""}`, "", true, true},
		{`{"p":12345}`, "", true, false},
		{`{"p":true}`, "", true, false},
		{`{"p":["1"]}`, "", true, false},
		{`{"p":null}`, "", false, false},
		{`{}`, "", false, false},
	}
	for _, tt := range tests {
		var out struct {
			P JSONString `json:"p"`
		}
		if err := json.Unmarshal([]byte(tt.body), &out); err != nil {
			t.Fatalf("%s: body ต้อง parse ผ่าน: %v", tt.body, err)
		}
		if out.P.Value != tt.value || out.P.Present != tt.present || out.P.IsString != tt.isString {
			t.Errorf("%s: got %+v", tt.body, out.P)
		}
	}
}

func TestRandomString(t *testing.T) {
	s, err := RandomString("ab", 50)
	if err != nil || len(s) != 50 {
		t.Fatalf("got %q, %v", s, err)
	}
	for _, ch := range s {
		if ch != 'a' && ch != 'b' {
			t.Fatalf("ตัวอักษรนอก alphabet: %q", ch)
		}
	}
}
