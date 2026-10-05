package agentauth

import (
	"strings"
	"testing"
	"time"

	"app/app/models"
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

func TestSplitSubaccountUsername(t *testing.T) { // AUTH-18
	tests := []struct {
		in          string
		owner, name string
		ok          bool
	}{
		{"agent01@staff", "agent01", "staff", true},
		{"agent01@abc", "agent01", "abc", true},
		{"agent01@" + strings.Repeat("a", 20), "agent01", strings.Repeat("a", 20), true},
		{"agent01@ab", "", "", false},                         // สั้นกว่า 3
		{"agent01@" + strings.Repeat("a", 21), "", "", false}, // ยาวกว่า 20
		{"agent01@sta_ff", "", "", false},
		{"agent01@Staff", "", "", false}, // ต้องผ่าน NormalizeUsername ก่อน
		{"@staff", "", "", false},
		{"a@b@staff", "", "", false},
		{"agent01", "", "", false},
	}
	for _, tt := range tests {
		owner, name, ok := SplitSubaccountUsername(tt.in)
		if owner != tt.owner || name != tt.name || ok != tt.ok {
			t.Errorf("SplitSubaccountUsername(%q) = %q, %q, %v", tt.in, owner, name, ok)
		}
	}
}

func TestWorstStatus(t *testing.T) { // AUTH-53
	const a, s, l = models.AgentStatusActive, models.AgentStatusSuspended, models.AgentStatusLocked
	tests := []struct {
		name string
		in   []models.AgentStatus
		want models.AgentStatus
	}{
		{"ไม่มี upline", nil, a},
		{"ตัวเองอย่างเดียว", []models.AgentStatus{s}, s},
		{"ทั้งสาย ACTIVE", []models.AgentStatus{a, a, a}, a},
		{"upline SUSPENDED", []models.AgentStatus{a, s, a}, s},
		{"ตัวเอง SUSPENDED upline ACTIVE", []models.AgentStatus{s, a}, s},
		{"upline LOCKED", []models.AgentStatus{a, s, l}, l},
		{"LOCKED ชนะ SUSPENDED", []models.AgentStatus{l, s}, l},
	}
	for _, tt := range tests {
		if got := WorstStatus(tt.in...); got != tt.want {
			t.Errorf("%s: got %s, want %s", tt.name, got, tt.want)
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
