package utils

import "testing"

func TestNewPage(t *testing.T) {
	tests := []struct {
		page, limit int
		want        Page
	}{
		{0, 0, Page{1, DefaultPageLimit}},
		{2, 10, Page{2, 10}},
		{-1, -5, Page{1, DefaultPageLimit}},
		{3, 500, Page{3, MaxPageLimit}},
	}
	for _, tt := range tests {
		if got := NewPage(tt.page, tt.limit); got != tt.want {
			t.Errorf("NewPage(%d, %d) = %+v, want %+v", tt.page, tt.limit, got, tt.want)
		}
	}
}
