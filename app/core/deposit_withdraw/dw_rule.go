// Package depositwithdraw คือกฎของการฝาก / ถอนเงินให้ลูกตรง — spec: docs/modules/deposit_withdraw.md
package depositwithdraw

import (
	"math"
	"time"

	"app/app/models"
)

const (
	MaxAmount    = 1_000_000_000 // เพดานต่อรายการ (DW-10 · lead DQ6)
	NoteMaxRunes = 200           // DW-15
	LogsMaxDays  = 31            // DW-47 ช่วงวันของ DW3 รวมทั้ง 2 วัน

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

// Balances — ยอดก่อน / หลังของทั้ง 2 ฝั่ง · แม่ไม่จำกัด (Superadmin) = ฝั่งแม่เป็น 0 ทั้งคู่
type Balances struct {
	ParentBefore, ParentAfter float64
	TargetBefore, TargetAfter float64
}

// Settle — ยอดหลังทำรายการ (DW-12 – DW-14) · ปัด 4 ตำแหน่งหลังบวก / ลบ แล้วจึงเทียบกับ 0 (กฎข้อ 9)
// unlimited = แม่เป็น Superadmin: ฝากไม่หักแม่ · ถอนไม่เพิ่มแม่
func Settle(t models.DWTransactionType, unlimited bool, parentBefore, targetBefore, amount float64) (Balances, Violation) {
	b := Balances{TargetBefore: targetBefore}
	if !unlimited {
		b.ParentBefore = parentBefore
	}
	switch t {
	case models.DWDeposit:
		if !unlimited {
			b.ParentAfter = round4(parentBefore - amount)
			if b.ParentAfter < 0 {
				return b, ParentInsufficient
			}
		}
		b.TargetAfter = round4(targetBefore + amount)
	default: // DWWithdraw
		b.TargetAfter = round4(targetBefore - amount)
		if b.TargetAfter < 0 {
			return b, TargetInsufficient
		}
		if !unlimited {
			b.ParentAfter = round4(parentBefore + amount)
		}
	}
	return b, OK
}

// LedgerReasons — reason ของ balance_ledger ฝั่งแม่ / ฝั่งลูก (DW-33 · lead DQ5) · withParent = false เมื่อแม่เป็น Superadmin
func LedgerReasons(t models.DWTransactionType, unlimited bool) (parent, target models.LedgerReason, withParent bool) {
	if t == models.DWDeposit {
		if unlimited {
			return "", models.LedgerDepositFromSuperadmin, false
		}
		return models.LedgerDepositOut, models.LedgerDepositIn, true
	}
	if unlimited {
		return "", models.LedgerWithdrawToSuperadmin, false
	}
	return models.LedgerWithdrawIn, models.LedgerWithdrawOut, true
}

// SignedAmounts — amount ใน ledger ของฝั่งแม่ / ลูก (+ เข้า · − ออก)
func SignedAmounts(t models.DWTransactionType, amount float64) (parent, target float64) {
	if t == models.DWDeposit {
		return -amount, amount
	}
	return amount, -amount
}

// DateRangeOK — ช่วงวันของ DW3 (DW-47): from ≤ to และรวมทั้ง 2 วันไม่เกิน LogsMaxDays วัน · from / to = เที่ยงคืนของวันนั้น
func DateRangeOK(from, to time.Time) bool {
	return !to.Before(from) && to.Sub(from) < LogsMaxDays*24*time.Hour
}
