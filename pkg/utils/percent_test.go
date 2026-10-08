package utils

import (
	"encoding/json"
	"testing"
)

func TestPercentMarshalJSON(t *testing.T) { // ACC-18 · MGMT-17
	tests := []struct {
		in   Percent
		want string
	}{
		{0, "0"},
		{50, "0.5"},
		{5, "0.05"},
		{7000, "70"},
		{9550, "95.5"},
		{10000, "100"},
		{3025, "30.25"},
		{-150, "-1.5"},
	}
	for _, tt := range tests {
		b, err := json.Marshal(tt.in)
		if err != nil {
			t.Fatal(err)
		}
		if string(b) != tt.want {
			t.Errorf("Percent(%d) = %s, want %s", int(tt.in), b, tt.want)
		}
	}
}
