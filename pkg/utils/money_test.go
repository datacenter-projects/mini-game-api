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
