package depositwithdraw

import (
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"app/app/models"
)

func TestOwnerOfRole(t *testing.T) { // DW-02
	tests := []struct {
		role string
		want models.BalanceOwnerType
		ok   bool
	}{
		{"COMPANY", models.BalanceOwnerAgent, true},
		{"SHAREHOLDER", models.BalanceOwnerAgent, true},
		{"AGENT", models.BalanceOwnerAgent, true},
		{"MEMBER", models.BalanceOwnerMember, true},
		{"member", "", false},
		{"ADMIN", "", false},
		{"SUPERADMIN", "", false},
		{"", "", false},
	}
	for _, tt := range tests {
		if got, ok := OwnerOfRole(tt.role); got != tt.want || ok != tt.ok {
			t.Errorf("OwnerOfRole(%q) = %q, %v · want %q, %v", tt.role, got, ok, tt.want, tt.ok)
		}
	}
}

func TestCanDeposit(t *testing.T) { // DW-05 · lead DQ1
	tests := []struct {
		status models.AgentStatus
		want   bool
	}{
		{models.AgentStatusActive, true},
		{models.AgentStatusSuspended, false},
		{models.AgentStatusLocked, false},
	}
	for _, tt := range tests {
		if got := CanDeposit(tt.status); got != tt.want {
			t.Errorf("CanDeposit(%q) = %v · want %v", tt.status, got, tt.want)
		}
	}
}

func TestSettle(t *testing.T) { // DW-10 · DW-12 – DW-14
	tests := []struct {
		name         string
		typ          models.DWTransactionType
		parent, tgt  float64
		amount       float64
		want         Balances
		wantViolated Violation
	}{
		{"ฝาก Company 50,000 → Share 10,000", models.DWDeposit, 50000, 0, 10000, Balances{50000, 40000, 0, 10000}, OK},
		{"ฝากหมดยอดพอดี", models.DWDeposit, 100, 5, 100, Balances{100, 0, 5, 105}, OK},
		{"ฝากเกินยอดแม่", models.DWDeposit, 50000, 0, 60000, Balances{50000, -10000, 0, 0}, ParentInsufficient},
		{"Superadmin ฝากจากยอดตั้งต้น 100,000,000", models.DWDeposit, 100000000, 0, 1000000, Balances{100000000, 99000000, 0, 1000000}, OK},
		{"0.1 + 0.2 = 0.3", models.DWDeposit, 1, 0.1, 0.2, Balances{1, 0.8, 0.1, 0.3}, OK},
		{"ถอนหมด", models.DWWithdraw, 4900, 100, 100, Balances{4900, 5000, 100, 0}, OK},
		{"ถอนเกิน 0.0001", models.DWWithdraw, 0, 100, 100.0001, Balances{0, 0, 100, -0.0001}, TargetInsufficient},
		{"ถอนเข้าแม่ที่ยอด 0", models.DWWithdraw, 0, 700, 500, Balances{0, 500, 700, 200}, OK},
	}
	for _, tt := range tests {
		got, v := Settle(tt.typ, tt.parent, tt.tgt, tt.amount)
		if v != tt.wantViolated || (v == OK && got != tt.want) {
			t.Errorf("%s: Settle = %+v, %v · want %+v, %v", tt.name, got, v, tt.want, tt.wantViolated)
		}
	}
}

func TestLedgerReasons(t *testing.T) { // DW-33
	tests := []struct {
		typ    models.DWTransactionType
		parent models.LedgerReason
		target models.LedgerReason
	}{
		{models.DWDeposit, models.LedgerDepositOut, models.LedgerDepositIn},
		{models.DWWithdraw, models.LedgerWithdrawIn, models.LedgerWithdrawOut},
	}
	for _, tt := range tests {
		if p, tg := LedgerReasons(tt.typ); p != tt.parent || tg != tt.target {
			t.Errorf("LedgerReasons(%q) = %q, %q · want %q, %q", tt.typ, p, tg, tt.parent, tt.target)
		}
	}
}

func TestSignedAmounts(t *testing.T) {
	if p, tg := SignedAmounts(models.DWDeposit, 100); p != -100 || tg != 100 {
		t.Errorf("deposit = %v, %v", p, tg)
	}
	if p, tg := SignedAmounts(models.DWWithdraw, 100); p != 100 || tg != -100 {
		t.Errorf("withdraw = %v, %v", p, tg)
	}
}

func TestDateRangeOK(t *testing.T) { // DW-47
	day := func(s string) time.Time {
		d, _ := time.Parse("2006-01-02", s)
		return d
	}
	tests := []struct {
		from, to string
		want     bool
	}{
		{"2026-10-01", "2026-10-01", true},
		{"2026-10-01", "2026-10-31", true},  // 31 วัน
		{"2026-10-01", "2026-11-01", false}, // 32 วัน
		{"2026-10-14", "2026-10-01", false},
	}
	for _, tt := range tests {
		if got := DateRangeOK(day(tt.from), day(tt.to)); got != tt.want {
			t.Errorf("DateRangeOK(%s, %s) = %v · want %v", tt.from, tt.to, got, tt.want)
		}
	}
}

func TestFailedNote(t *testing.T) { // DW-36
	long := strings.Repeat("ก", 200)
	tests := []struct {
		note, reason, want string
	}{
		{"", "ยอดเงินของคุณไม่พอ", "ล้มเหลว: ยอดเงินของคุณไม่พอ"},
		{"เติมรอบแรก", "ยอดเงินของคุณไม่พอ", "เติมรอบแรก · ล้มเหลว: ยอดเงินของคุณไม่พอ"},
		{long, "ยอดเงินของคุณไม่พอ", strings.Repeat("ก", 200-len([]rune("ล้มเหลว: ยอดเงินของคุณไม่พอ"))-3) + " · ล้มเหลว: ยอดเงินของคุณไม่พอ"},
	}
	for _, tt := range tests {
		got := FailedNote(tt.note, tt.reason)
		if got != tt.want || utf8.RuneCountInString(got) > NoteMaxRunes {
			t.Errorf("FailedNote(%d runes, %q) = %q (%d runes) · want %q", utf8.RuneCountInString(tt.note), tt.reason, got, utf8.RuneCountInString(got), tt.want)
		}
	}
}
