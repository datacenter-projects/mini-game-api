package agentauth

import (
	"testing"
	"time"
)

func TestNormalizeUsername(t *testing.T) { // AUTH-01
	tests := []struct{ in, want string }{
		{"agent01", "agent01"},
		{"  Agent01 ", "agent01"},
		{"AGENT01\t", "agent01"},
		{"", ""},
	}
	for _, tt := range tests {
		if got := NormalizeUsername(tt.in); got != tt.want {
			t.Errorf("NormalizeUsername(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestIsSubaccountUsername(t *testing.T) {
	tests := []struct {
		in   string
		want bool
	}{
		{"agent01", false},
		{"agent01@staff", true},
		{"@", true},
	}
	for _, tt := range tests {
		if got := IsSubaccountUsername(tt.in); got != tt.want {
			t.Errorf("IsSubaccountUsername(%q) = %v, want %v", tt.in, got, tt.want)
		}
	}
}

func TestSessionTTL(t *testing.T) { // AUTH-07, AUTH-08
	login := time.Date(2026, 10, 2, 8, 0, 0, 0, time.UTC)
	deadline := login.Add(12 * time.Hour) // 20:00
	idle := 60 * time.Minute

	tests := []struct {
		name string
		now  time.Time
		want time.Duration
	}{
		{"เพิ่ง login ได้ idle เต็ม", login, 60 * time.Minute},
		{"ผ่านไป 5 ชม. ยังได้ idle เต็ม", login.Add(5 * time.Hour), 60 * time.Minute},
		{"เหลือ 20 นาทีถึง absolute ได้แค่ 20 นาที", deadline.Add(-20 * time.Minute), 20 * time.Minute},
		{"ครบ absolute พอดี = หมดอายุ", deadline, 0},
		{"เลย absolute = หมดอายุ", deadline.Add(time.Minute), -time.Minute},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SessionTTL(tt.now, deadline, idle); got != tt.want {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsLoginBlocked(t *testing.T) { // AUTH-10
	tests := []struct {
		fails, limit int
		want         bool
	}{
		{0, 5, false},
		{4, 5, false},
		{5, 5, true},
		{9, 5, true},
		{100, 0, false}, // limit 0 = ปิดการล็อก
	}
	for _, tt := range tests {
		if got := IsLoginBlocked(tt.fails, tt.limit); got != tt.want {
			t.Errorf("IsLoginBlocked(%d, %d) = %v, want %v", tt.fails, tt.limit, got, tt.want)
		}
	}
}
