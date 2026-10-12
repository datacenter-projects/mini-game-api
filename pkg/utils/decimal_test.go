package utils

import (
	"encoding/json"
	"testing"
)

func TestDecimalFloat4(t *testing.T) { // ACC-18 · MGMT-17 · กฎข้อ 9 (แก้ 2026-10-09)
	tests := []struct {
		json    string
		present bool
		want    float64
		ok      bool
	}{
		{`70`, true, 70, true},
		{`0.5`, true, 0.5, true},
		{`0`, true, 0, true},
		{`1234.56`, true, 1234.56, true},
		{`10000.1234`, true, 10000.1234, true},
		{`10000.00`, true, 10000, true},
		{`-1.5`, true, -1.5, true},
		{`30.12345`, true, 0, false},
		{`1e2`, true, 0, false},
		{`"70"`, true, 0, false},
		{`true`, true, 0, false},
		{`123456789012`, true, 0, false},
		{`null`, false, 0, false},
	}
	for _, tt := range tests {
		var d Decimal
		if err := json.Unmarshal([]byte(tt.json), &d); err != nil {
			t.Fatalf("%s: %v", tt.json, err)
		}
		got, ok := d.Float4()
		if d.Present != tt.present || got != tt.want || ok != tt.ok {
			t.Errorf("%s: present=%v Float4=%v,%v want present=%v %v,%v", tt.json, d.Present, got, ok, tt.present, tt.want, tt.ok)
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
