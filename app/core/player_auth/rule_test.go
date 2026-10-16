package playerauth

import (
	"strings"
	"testing"

	agentManagementCore "app/app/core/agent_management"
	"app/app/models"
)

func TestIsPlayerUsername(t *testing.T) { // rule: PA-05
	tests := []struct {
		in string
		ok bool
	}{
		{"abc", true}, {"p001", true}, {strings.Repeat("a", 32), true},
		{"ab", false}, {strings.Repeat("a", 33), false}, {"abc!", false}, {"ab c", false}, {"ABC", false}, // ต้อง normalize ก่อน
		{"a@bc", false}, {"", false}, {"กขค", false},
	}
	for _, tt := range tests {
		if got := IsPlayerUsername(tt.in); got != tt.ok {
			t.Errorf("IsPlayerUsername(%q) = %v want %v", tt.in, got, tt.ok)
		}
	}
}

func TestNormalizeAndCompose(t *testing.T) { // rule: PA-04, PA-05, PA-06
	if got := NormalizeUsername("  Mem01 "); got != "mem01" {
		t.Errorf("NormalizeUsername = %q", got)
	}
	if got := ComposeUsername("Agent01", "p001"); got != "agent01@p001" {
		t.Errorf("ComposeUsername = %q", got)
	}
	long := ComposeUsername(strings.Repeat("a", 32), strings.Repeat("b", 32))
	if len(long) != StoredUsernameMax {
		t.Errorf("len = %d want %d", len(long), StoredUsernameMax)
	}
}

func TestCanUseTransfer(t *testing.T) { // rule: PA-14
	tests := map[agentManagementCore.UserType]bool{
		agentManagementCore.UserTypeShareB2C: true, agentManagementCore.UserTypeAgent: true,
		agentManagementCore.UserTypeCompanyTransfer: false, agentManagementCore.UserTypeShareB2B: false,
		agentManagementCore.UserTypeShareReseller: false, agentManagementCore.UserTypeShareMaster: false,
		agentManagementCore.UserTypeCompanySeamless1to1: false, agentManagementCore.UserTypeMember: false,
	}
	for ut, want := range tests {
		if got := CanUseTransfer(ut); got != want {
			t.Errorf("CanUseTransfer(%s) = %v want %v", ut, got, want)
		}
	}
}

func TestEffectiveStatusAndCanBet(t *testing.T) { // rule: PA-30, PA-35
	a, s, l := models.AgentStatusActive, models.AgentStatusSuspended, models.AgentStatusLocked
	tests := []struct {
		name    string
		own     models.AgentStatus
		upline  []models.AgentStatus
		want    models.AgentStatus
		win     float64
		wantBet bool
	}{
		{"ทุกชั้น ACTIVE", a, []models.AgentStatus{a, a}, a, 100, true},
		{"สายบน SUSPENDED", a, []models.AgentStatus{a, s}, s, 100, false},
		{"ตัวเอง SUSPENDED", s, nil, s, 100, false},
		{"สายบน LOCKED เหนือ SUSPENDED", s, []models.AgentStatus{l, s}, l, 100, false},
		{"ตัวเอง LOCKED", l, []models.AgentStatus{a}, l, 100, false},
		{"ACTIVE แต่ win limit 0", a, nil, a, 0, false},
		{"ACTIVE win limit เล็กที่สุด", a, nil, a, 0.0001, true},
	}
	for _, tt := range tests {
		got := EffectiveStatus(tt.own, tt.upline...)
		if got != tt.want {
			t.Errorf("%s: EffectiveStatus = %s want %s", tt.name, got, tt.want)
		}
		if bet := CanBet(got, tt.win); bet != tt.wantBet {
			t.Errorf("%s: CanBet = %v want %v", tt.name, bet, tt.wantBet)
		}
	}
}

func TestOperatorGateOf(t *testing.T) { // rule: PA-13
	a, s, l := models.AgentStatusActive, models.AgentStatusSuspended, models.AgentStatusLocked
	tests := []struct {
		name   string
		own    models.AgentStatus
		upline []models.AgentStatus
		want   OperatorGate
	}{
		{"ปกติ", a, []models.AgentStatus{a}, OperatorOK},
		{"agent SUSPENDED", s, nil, OperatorSuspended},
		{"สายบน SUSPENDED", a, []models.AgentStatus{s}, OperatorSuspended},
		{"agent LOCKED", l, nil, OperatorLocked},
		{"สายบน LOCKED", a, []models.AgentStatus{l}, OperatorLocked},
		{"LOCKED เหนือ SUSPENDED", s, []models.AgentStatus{l}, OperatorLocked},
	}
	for _, tt := range tests {
		if got := OperatorGateOf(tt.own, tt.upline...); got != tt.want {
			t.Errorf("%s: got %v want %v", tt.name, got, tt.want)
		}
	}
}

func TestKeyEqual(t *testing.T) { // rule: PA-11
	if !KeyEqual("abc", "abc") || KeyEqual("abc", "abd") || KeyEqual("abc", "") || KeyEqual("", "abc") {
		t.Error("KeyEqual ผิด")
	}
}

func TestIPAllowed(t *testing.T) { // rule: PA-10 · ACC-07
	tests := []struct {
		name  string
		ip    string
		cidrs []string
		want  bool
	}{
		{"IP เดี่ยว /32", "203.0.113.10", []string{"203.0.113.10/32"}, true},
		{"ในช่วง /24", "198.51.100.77", []string{"203.0.113.10/32", "198.51.100.0/24"}, true},
		{"นอกช่วง", "198.51.101.1", []string{"198.51.100.0/24"}, false},
		{"whitelist ว่าง = ปฏิเสธ", "203.0.113.10", nil, false},
		{"ip อ่านไม่ได้", "not-an-ip", []string{"0.0.0.0/0"}, false},
		{"cidr เสียถูกข้าม", "203.0.113.10", []string{"bad", "203.0.113.0/24"}, true},
		{"IPv4-mapped IPv6", "::ffff:203.0.113.10", []string{"203.0.113.10/32"}, true},
		{"IPv6 ไม่อยู่ใน IPv4 whitelist", "2001:db8::1", []string{"203.0.113.0/24"}, false},
	}
	for _, tt := range tests {
		if got := IPAllowed(tt.ip, tt.cidrs); got != tt.want {
			t.Errorf("%s: IPAllowed(%q, %v) = %v want %v", tt.name, tt.ip, tt.cidrs, got, tt.want)
		}
	}
}

func TestNormalizeLanguage(t *testing.T) { // rule: L2
	tests := []struct {
		in   string
		want string
		ok   bool
	}{{"", "th", true}, {"th", "th", true}, {"EN", "en", true}, {" en ", "en", true}, {"fr", "", false}, {"thai", "", false}}
	for _, tt := range tests {
		got, ok := NormalizeLanguage(tt.in)
		if got != tt.want || ok != tt.ok {
			t.Errorf("NormalizeLanguage(%q) = %q, %v want %q, %v", tt.in, got, ok, tt.want, tt.ok)
		}
	}
}

func TestLobbyLink(t *testing.T) { // rule: L2 · LQ-c
	tests := []struct {
		base string
		want string
		ok   bool
	}{
		{"https://lobby.example.com/", "https://lobby.example.com/?token=abc.def&lang=th", true},
		{"https://lobby.example.com/play?x=1", "https://lobby.example.com/play?x=1&token=abc.def&lang=th", true},
		{"", "", false}, {"lobby.example.com", "", false}, {"/relative", "", false},
		{"ftp://lobby.example.com/", "", false}, {"https:///path", "", false}, {"javascript:alert(1)", "", false},
		{"http://localhost:3000/", "http://localhost:3000/?token=abc.def&lang=th", true},
	}
	for _, tt := range tests {
		got, ok := LobbyLink(tt.base, "abc.def", "th")
		if got != tt.want || ok != tt.ok {
			t.Errorf("LobbyLink(%q) = %q, %v want %q, %v", tt.base, got, ok, tt.want, tt.ok)
		}
	}
}

func TestResolveCurrency(t *testing.T) { // rule: PA-08A
	tests := []struct {
		name      string
		agent     []string
		requested string
		want      string
		v         CurrencyViolation
	}{
		{"สกุลเดียว ไม่ส่ง", []string{"THB"}, "", "THB", CurrencyOK},
		{"สกุลเดียว ส่งตรง", []string{"THB"}, "THB", "THB", CurrencyOK},
		{"ส่งตัวเล็ก", []string{"THB", "USD"}, " usd ", "USD", CurrencyOK},
		{"หลายสกุล ไม่ส่ง", []string{"THB", "USD"}, "", "", CurrencyRequired},
		{"ส่งสกุลนอก agent", []string{"THB"}, "USD", "", CurrencyNotAgents},
		{"agent ไม่มีสกุล ไม่ส่ง", nil, "", "", CurrencyNoneConfig},
		{"agent ไม่มีสกุล ส่งมา", nil, "THB", "", CurrencyNotAgents},
	}
	for _, tt := range tests {
		got, v := ResolveCurrency(tt.agent, tt.requested)
		if got != tt.want || v != tt.v {
			t.Errorf("%s: ResolveCurrency = %q, %v want %q, %v", tt.name, got, v, tt.want, tt.v)
		}
	}
}

func TestCurrencyMatchesPlayer(t *testing.T) { // rule: PA-08A
	if !CurrencyMatchesPlayer("THB", "") || !CurrencyMatchesPlayer("THB", "thb") || CurrencyMatchesPlayer("THB", "USD") {
		t.Error("CurrencyMatchesPlayer ผิด")
	}
}
