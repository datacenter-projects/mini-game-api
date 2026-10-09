package agentmanagement

// กฎค่าหุ้นส่วน (MGMT-16 – MGMT-25) · ทุกค่าเป็น bp (1% = 100 bp)

const (
	FullPTBP         = 10000 // 100% — Superadmin ได้รับ (MGMT-22)
	PTStepBP         = 50    // 0.5% (MGMT-18)
	MaxCommissionBP  = 100   // 1% (MGMT-18)
	CommissionStepBP = 10    // 0.1% (MGMT-18)
)

// PTGroup — กลุ่ม PT ใน API (MGMT-16)
type PTGroup string

const PTGroupMinigame PTGroup = "minigame"

// GroupGame — เกมหนึ่งในกลุ่ม
type GroupGame struct {
	Category string
	GameCode string
}

// groupGames — กลุ่ม → หมวด → เกม (MGMT-16) · เพิ่มเกม / กลุ่มที่นี่โดยไม่เปลี่ยน API / ตาราง
var groupGames = map[PTGroup][]GroupGame{
	PTGroupMinigame: {
		{"minigame", "coin_toss"},
		{"minigame", "rock_paper_scissors"},
		{"minigame", "scratch_card"},
	},
}

// Groups — กลุ่มทั้งหมดเรียงตามชื่อ
func Groups() []PTGroup { return []PTGroup{PTGroupMinigame} }

// GamesOf — เกมในกลุ่ม · กลุ่มที่ไม่มี = nil
func GamesOf(g PTGroup) []GroupGame { return groupGames[g] }

// GroupOfGame — กลุ่มของรหัสเกม · ไม่พบ = ok false
func GroupOfGame(gameCode string) (PTGroup, bool) {
	for g, games := range groupGames {
		for _, gg := range games {
			if gg.GameCode == gameCode {
				return g, true
			}
		}
	}
	return "", false
}

// IsPTStep — 0 ถึง max ทีละ 0.5%
func IsPTStep(bp int) bool { return bp >= 0 && bp <= FullPTBP && bp%PTStepBP == 0 }

// IsCommissionStep — 0 ถึง 1% ทีละ 0.1%
func IsCommissionStep(bp int) bool {
	return bp >= 0 && bp <= MaxCommissionBP && bp%CommissionStepBP == 0
}

// PTViolation — ผลการตรวจค่าหุ้นส่วน · service แปลงเป็น apperr
type PTViolation int

const (
	PTOK                  PTViolation = iota
	PTInvalidStep                     // 422 — ไม่ลงตาม step / เกินช่วง (MGMT-18)
	PTExceedsReceived                 // 402305 (MGMT-18, MGMT-22)
	PTSeamlessMasterLock              // 402307 (MGMT-19)
	PTForceRemainExceeded             // 402308 (MGMT-25)
	PTCommissionExceeded              // 402309 (MGMT-18)
)

// ChildPT — ค่าที่ผู้สร้างตั้งให้ลูก 1 กลุ่ม (MGMT-16)
type ChildPT struct {
	PTFromParentBP int
	ForceBP        int
	RemainBP       int
	CommissionBP   int
}

// ValidateChildPT — ตรวจค่าที่ผู้สร้างตั้งให้ลูก (MGMT-18, MGMT-19, MGMT-25)
// creatorReceivedBP = ค่าที่ผู้สร้างได้รับในกลุ่มนี้ · creatorIsSeamlessMaster = ผู้สร้างเป็น Company Seamless Master
func ValidateChildPT(v ChildPT, creatorReceivedBP int, creatorIsSeamlessMaster bool) PTViolation {
	return CheckChildPT(v, creatorReceivedBP, creatorIsSeamlessMaster).Violation
}

// PTIssue — ผลตรวจค่าหุ้นส่วนพร้อม field ที่ผิดและค่าที่ตั้งได้ (ใช้สร้างข้อความ error)
type PTIssue struct {
	Violation PTViolation
	Field     string // pt_from_parent · force · remain_quota · commission_percent
	LimitBP   int    // ค่าสูงสุดที่ตั้งได้ · PTSeamlessMasterLock = ค่าที่ต้องเป็น
}

// CheckChildPT — ตรวจค่าที่ผู้สร้างตั้งให้ลูก ไล่ตามลำดับ field ใน body (pt_from_parent → force → remain_quota → commission_percent)
func CheckChildPT(v ChildPT, creatorReceivedBP int, creatorIsSeamlessMaster bool) PTIssue {
	if !IsPTStep(v.PTFromParentBP) {
		return PTIssue{PTInvalidStep, "pt_from_parent", FullPTBP}
	}
	if creatorIsSeamlessMaster && v.PTFromParentBP != creatorReceivedBP {
		return PTIssue{PTSeamlessMasterLock, "pt_from_parent", creatorReceivedBP}
	}
	if v.PTFromParentBP > creatorReceivedBP {
		return PTIssue{PTExceedsReceived, "pt_from_parent", creatorReceivedBP}
	}
	for _, f := range []struct {
		name string
		bp   int
	}{{"force", v.ForceBP}, {"remain_quota", v.RemainBP}} {
		if !IsPTStep(f.bp) {
			return PTIssue{PTInvalidStep, f.name, v.PTFromParentBP}
		}
		if creatorIsSeamlessMaster && f.bp != 0 {
			return PTIssue{PTSeamlessMasterLock, f.name, 0}
		}
		if f.bp > v.PTFromParentBP {
			return PTIssue{PTForceRemainExceeded, f.name, v.PTFromParentBP}
		}
	}
	if v.CommissionBP < 0 || v.CommissionBP%CommissionStepBP != 0 {
		return PTIssue{PTInvalidStep, "commission_percent", MaxCommissionBP}
	}
	if v.CommissionBP > MaxCommissionBP {
		return PTIssue{PTCommissionExceeded, "commission_percent", MaxCommissionBP}
	}
	return PTIssue{Violation: PTOK}
}

// ValidateMemberCommission — Member มีแค่ Commission (MGMT-21)
func ValidateMemberCommission(bp int) PTViolation {
	if bp < 0 || bp%CommissionStepBP != 0 {
		return PTInvalidStep
	}
	if bp > MaxCommissionBP {
		return PTCommissionExceeded
	}
	return PTOK
}

// ValidateOwnPT — ค่าถือ pt ที่บัญชีตั้งเอง (MGMT-22) · Company Seamless Master ล็อกที่ 0 (MGMT-19)
func ValidateOwnPT(ptBP, receivedBP int, isSeamlessMaster bool) PTViolation {
	if !IsPTStep(ptBP) {
		return PTInvalidStep
	}
	if isSeamlessMaster && ptBP != 0 {
		return PTSeamlessMasterLock
	}
	if ptBP > receivedBP {
		return PTExceedsReceived
	}
	return PTOK
}

// InitialOwnPT — บัญชีใหม่เริ่มถือทั้งหมดที่ได้รับ (MGMT-22) · Company Seamless Master = 0 (MGMT-19)
func InitialOwnPT(receivedBP int, isSeamlessMaster bool) int {
	if isSeamlessMaster {
		return 0
	}
	return receivedBP
}

// ValidateMemberPT — ค่าที่ผู้สร้างถือสู้กับ Member คนนี้ (MGMT-21 แก้ 2026-10-09) · 0 ถึงค่าที่ผู้สร้างได้รับ ทีละ 0.5%
func ValidateMemberPT(ptBP, creatorReceivedBP int) PTViolation {
	if !IsPTStep(ptBP) {
		return PTInvalidStep
	}
	if ptBP > creatorReceivedBP {
		return PTExceedsReceived
	}
	return PTOK
}

// MemberRemain — ส่วนที่ผู้สร้างไม่ได้ถือสู้กับ Member คนนี้ = ค่าที่ผู้สร้างได้รับ − pt (MGMT-21 แก้ 2026-10-09)
func MemberRemain(creatorReceivedBP, ptBP int) int { return creatorReceivedBP - ptBP }

// MinPTFromParent — ค่าต่ำสุดที่ผู้สร้างตั้งให้ลูกได้ (MGMT-24)
// = ค่าที่มากที่สุดระหว่าง pt ของลูก และค่าที่ลูกให้ลูกของมันแต่ละคน
func MinPTFromParent(childOwnPTBP int, grandchildrenPTFromParentBP []int) int {
	m := childOwnPTBP
	for _, v := range grandchildrenPTFromParentBP {
		if v > m {
			m = v
		}
	}
	return m
}

// IsGameOpen — เกมเล่นได้เมื่อ status_game ของเกมนั้นเป็น true ทุกชั้นตั้งแต่บัญชีนี้ถึงหัวสาย (MGMT-20)
// status ใน pt ไม่เกี่ยว — false = ไม่รับ PT แต่เกมยังเปิด
func IsGameOpen(statusGameChain []bool) bool {
	for _, on := range statusGameChain {
		if !on {
			return false
		}
	}
	return true
}
