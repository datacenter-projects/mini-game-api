// Package depositwithdraw คือกฎของการฝาก / ถอนเงินให้ลูกตรง — spec: docs/modules/deposit_withdraw.md
package depositwithdraw

import (
	"math"
	"time"
	"unicode/utf8"

	"app/app/models"
)

const (
	MaxAmount    = 1_000_000_000 // เพดานต่อรายการ (DW-10 · lead DQ6)
	NoteMaxRunes = 200           // DW-15
	LogsMaxDays  = 31            // DW-47 ช่วงวันของ DW3 รวมทั้ง 2 วัน

	// SuperadminSeedAmount — ยอดตั้งต้นของ Superadmin ต่อสกุล (lead 2026-10-15 · migration 20261015100000 · script create_superadmin)
	SuperadminSeedAmount = 100_000_000

	// RoleMember — role ของ Member ในรายชื่อ DW1 และ request DW2 (ไม่มีใน models.AgentRole)
	RoleMember = "MEMBER"
)

// round4 — ปัด 4 ตำแหน่ง (ตรงกับ utils.Round4 · core ห้าม import pkg/utils เพราะดึง fiber มาด้วย)
func round4(v float64) float64 { return math.Round(v*10000) / 10000 }

// OwnerOfRole — role ใน request → ตารางของบัญชีปลายทาง (DW-02) · ADMIN / SUPERADMIN / ค่าอื่น = false
func OwnerOfRole(role string) (models.BalanceOwnerType, bool) {
	switch role {
	case string(models.AgentRoleCompany), string(models.AgentRoleShareholder), string(models.AgentRoleAgent):
		return models.BalanceOwnerAgent, true
	case RoleMember:
		return models.BalanceOwnerMember, true
	}
	return "", false
}

// CanDeposit — ฝากได้เฉพาะลูกที่ ACTIVE · ถอนได้ทุกสถานะ (DW-05 · lead DQ1)
func CanDeposit(status models.AgentStatus) bool { return status == models.AgentStatusActive }

// Violation — ผลตรวจยอดของ Settle
type Violation int

const (
	OK                 Violation = iota
	ParentInsufficient           // DW-12 → 404304
	TargetInsufficient           // DW-13 → 404306
)

// Balances — ยอดก่อน / หลังของทั้ง 2 ฝั่ง
type Balances struct {
	ParentBefore, ParentAfter float64
	TargetBefore, TargetAfter float64
}

// Settle — ยอดหลังทำรายการ (DW-12 – DW-14) · ปัด 4 ตำแหน่งหลังบวก / ลบ แล้วจึงเทียบกับ 0 (กฎข้อ 9)
// แม่ทุกประเภทรวม Superadmin มียอดจริง (lead 2026-10-15) — ฝากหักแม่ · ถอนเพิ่มแม่
func Settle(t models.DWTransactionType, parentBefore, targetBefore, amount float64) (Balances, Violation) {
	b := Balances{ParentBefore: parentBefore, TargetBefore: targetBefore}
	if t == models.DWDeposit {
		b.ParentAfter = round4(parentBefore - amount)
		if b.ParentAfter < 0 {
			return b, ParentInsufficient
		}
		b.TargetAfter = round4(targetBefore + amount)
		return b, OK
	}
	b.TargetAfter = round4(targetBefore - amount)
	if b.TargetAfter < 0 {
		return b, TargetInsufficient
	}
	b.ParentAfter = round4(parentBefore + amount)
	return b, OK
}

// LedgerReasons — reason ของ balance_ledger ฝั่งแม่ / ฝั่งลูก (DW-33)
func LedgerReasons(t models.DWTransactionType) (parent, target models.LedgerReason) {
	if t == models.DWDeposit {
		return models.LedgerDepositOut, models.LedgerDepositIn
	}
	return models.LedgerWithdrawIn, models.LedgerWithdrawOut
}

// SignedAmounts — amount ใน ledger ของฝั่งแม่ / ลูก (+ เข้า · − ออก)
func SignedAmounts(t models.DWTransactionType, amount float64) (parent, target float64) {
	if t == models.DWDeposit {
		return -amount, amount
	}
	return amount, -amount
}

// FailedNote — note ของแถว FAILED (DW-36): note ที่คนกดพิมพ์ + " · ล้มเหลว: " + เหตุผล · ไม่มี note = "ล้มเหลว: " + เหตุผล
// ยาวเกิน NoteMaxRunes ตัด note ที่พิมพ์ก่อน (เหตุผลต้องอยู่ครบ) — คอลัมน์ VARCHAR(200)
func FailedNote(note, reason string) string {
	suffix := "ล้มเหลว: " + reason
	if note == "" {
		return truncateRunes(suffix, NoteMaxRunes)
	}
	const sep = " · "
	room := NoteMaxRunes - utf8.RuneCountInString(suffix) - utf8.RuneCountInString(sep)
	if room <= 0 {
		return truncateRunes(suffix, NoteMaxRunes)
	}
	return truncateRunes(note, room) + sep + suffix
}

func truncateRunes(s string, n int) string {
	if utf8.RuneCountInString(s) <= n {
		return s
	}
	return string([]rune(s)[:n])
}

// DateRangeOK — ช่วงวันของ DW3 (DW-47): from ≤ to และรวมทั้ง 2 วันไม่เกิน LogsMaxDays วัน · from / to = เที่ยงคืนของวันนั้น
func DateRangeOK(from, to time.Time) bool {
	return !to.Before(from) && to.Sub(from) < LogsMaxDays*24*time.Hour
}
