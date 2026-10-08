package utils

import (
	"encoding/json"
	"testing"
)

func TestDecimalFixed2(t *testing.T) { // ACC-18 · MGMT-17
	tests := []struct {
		json    string
		present bool
		want    int64
		ok      bool
	}{
		{`70`, true, 7000, true},
		{`0.5`, true, 50, true},
		{`0`, true, 0, true},
		{`1234.56`, true, 123456, true},
		{`10000.00`, true, 1000000, true},
		{`-1.5`, true, -150, true},
		{`30.123`, true, 0, false},
		{`1e2`, true, 0, false},
		{`"70"`, true, 0, false},
		{`true`, true, 0, false},
		{`1234567890123456`, true, 0, false},
		{`null`, false, 0, false},
	}
	for _, tt := range tests {
		var d Decimal
		if err := json.Unmarshal([]byte(tt.json), &d); err != nil {
			t.Fatalf("%s: %v", tt.json, err)
		}
		got, ok := d.Fixed2()
		if d.Present != tt.present || got != tt.want || ok != tt.ok {
			t.Errorf("%s: present=%v Fixed2=%d,%v want present=%v %d,%v", tt.json, d.Present, got, ok, tt.present, tt.want, tt.ok)
		}
	}
}

func TestDecimalMissing(t *testing.T) {
	var v struct {
		A Decimal `json:"a"`
	}
	if err := json.Unmarshal([]byte(`{}`), &v); err != nil {
		t.Fatal(err)
	}
	if v.A.Present {
		t.Error("missing key must not be Present")
	}
}
