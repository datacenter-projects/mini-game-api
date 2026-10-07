package utils

import "testing"

func TestFindJSONNull(t *testing.T) { // ACC-32
	tests := []struct {
		body string
		path string
		ok   bool
	}{
		{`{"a":1,"b":"x"}`, "", false},
		{`{"phone":null}`, "phone", true},
		{`{"pt":{"game":{"force":null,"pt_from_parent":70}}}`, "pt.game.force", true},
		{`{"currencies":["THB",null]}`, "currencies[1]", true},
		{`{"b":null,"a":null}`, "a", true},
		{`null`, "", true},
		{`{"a":`, "", false},
		{`{"a":[],"b":{}}`, "", false},
	}
	for _, tt := range tests {
		path, ok := FindJSONNull([]byte(tt.body))
		if path != tt.path || ok != tt.ok {
			t.Errorf("FindJSONNull(%s) = %q, %v want %q, %v", tt.body, path, ok, tt.path, tt.ok)
		}
	}
}
