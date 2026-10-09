package utils

import (
	"encoding/json"
	"testing"
)

func TestMoneyMarshalJSON(t *testing.T) { // ACC-18
	tests := []struct {
		in   Money
		want string
	}{
		{0, "0.00"},
		{5, "0.05"},
		{50, "0.50"},
		{100, "1.00"},
		{96205600, "962056.00"},
		{123456789, "1234567.89"},
		{-150, "-1.50"},
		{-5, "-0.05"},
	}
	for _, tt := range tests {
		b, err := json.Marshal(tt.in)
		if err != nil {
			t.Fatal(err)
		}
		if string(b) != tt.want {
			t.Errorf("Money(%d) = %s, want %s", int64(tt.in), b, tt.want)
		}
	}
}

func TestCreditConversion(t *testing.T) { // user_members.credit (float) ↔ หน่วยย่อย 1/100
	tests := []struct {
		minor  int64
		credit float64
	}{
		{0, 0},
		{2550, 25.5},
		{1, 0.01},
		{96205600, 962056},
		{123456789, 1234567.89},
	}
	for _, tt := range tests {
		if got := MinorToCredit(tt.minor); got != tt.credit {
			t.Errorf("MinorToCredit(%d) = %v, want %v", tt.minor, got, tt.credit)
		}
		if got := CreditToMinor(tt.credit); got != tt.minor {
			t.Errorf("CreditToMinor(%v) = %d, want %d", tt.credit, got, tt.minor)
		}
	}
	if got := CreditToMinor(0.1 + 0.2); got != 30 { // 0.30000000000000004 ปัดเป็น 30
		t.Errorf("CreditToMinor(0.1+0.2) = %d, want 30", got)
	}
}
