package agentmanagement

import "math"

// กฎค่าหุ้นส่วน (MGMT-16 – MGMT-25) · ทุกค่าเป็น % แบบ float64 ปัด 4 ตำแหน่ง (กฎข้อ 9 — แก้ 2026-10-09)

const (
	FullPT         = 100.0 // Superadmin ได้รับ (MGMT-22)
	PTStep         = 0.5   // MGMT-18
	MaxCommission  = 1.0   // MGMT-18
	CommissionStep = 0.1   // MGMT-18
)

// round4 — ปัด 4 ตำแหน่ง (ตรงกับ utils.Round4 · core ห้าม import pkg/utils เพราะดึง fiber มาด้วย)
func round4(v float64) float64 { return math.Round(v*10000) / 10000 }

// isStep — v ลงตาม step ไหม · เทียบเป็นจำนวนเต็มของหน่วย 0.0001 กันเศษ float
func isStep(v, step float64) bool {
	return math.Mod(math.Round(v*10000), math.Round(step*10000)) == 0
}

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
func IsPTStep(v float64) bool { return v >= 0 && v <= FullPT && isStep(v, PTStep) }

// IsCommissionStep — 0 ถึง 1% ทีละ 0.1%
func IsCommissionStep(v float64) bool {
	return v >= 0 && v <= MaxCommission && isStep(v, CommissionStep)
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
	PTFromParent float64
	Force        float64
	Remain       float64
	Commission   float64
}

// ValidateChildPT — ตรวจค่าที่ผู้สร้างตั้งให้ลูก (MGMT-18, MGMT-19, MGMT-25)
// creatorReceived = ค่าที่ผู้สร้างได้รับในกลุ่มนี้ · creatorIsSeamlessMaster = ผู้สร้างเป็น Company Seamless Master
func ValidateChildPT(v ChildPT, creatorReceived float64, creatorIsSeamlessMaster bool) PTViolation {
	return CheckChildPT(v, creatorReceived, creatorIsSeamlessMaster).Violation
}

// PTIssue — ผลตรวจค่าหุ้นส่วนพร้อม field ที่ผิดและค่าที่ตั้งได้ (ใช้สร้างข้อความ error)
type PTIssue struct {
	Violation PTViolation
	Field     string  // pt_from_parent · force · remain_quota · commission_percent
	Limit     float64 // ค่าสูงสุดที่ตั้งได้ · PTSeamlessMasterLock = ค่าที่ต้องเป็น
}

// CheckChildPT — ตรวจค่าที่ผู้สร้างตั้งให้ลูก ไล่ตามลำดับ field ใน body (pt_from_parent → force → remain_quota → commission_percent)
func CheckChildPT(v ChildPT, creatorReceived float64, creatorIsSeamlessMaster bool) PTIssue {
	if !IsPTStep(v.PTFromParent) {
		return PTIssue{PTInvalidStep, "pt_from_parent", FullPT}
	}
	if creatorIsSeamlessMaster && v.PTFromParent != creatorReceived {
		return PTIssue{PTSeamlessMasterLock, "pt_from_parent", creatorReceived}
	}
	if v.PTFromParent > creatorReceived {
		return PTIssue{PTExceedsReceived, "pt_from_parent", creatorReceived}
	}
	for _, f := range []struct {
		name string
		v    float64
	}{{"force", v.Force}, {"remain_quota", v.Remain}} {
		if !IsPTStep(f.v) {
			return PTIssue{PTInvalidStep, f.name, v.PTFromParent}
		}
		if creatorIsSeamlessMaster && f.v != 0 {
			return PTIssue{PTSeamlessMasterLock, f.name, 0}
		}
		if f.v > v.PTFromParent {
			return PTIssue{PTForceRemainExceeded, f.name, v.PTFromParent}
		}
	}
	if v.Commission < 0 || !isStep(v.Commission, CommissionStep) {
		return PTIssue{PTInvalidStep, "commission_percent", MaxCommission}
	}
	if v.Commission > MaxCommission {
		return PTIssue{PTCommissionExceeded, "commission_percent", MaxCommission}
	}
	return PTIssue{Violation: PTOK}
}

// ValidateMemberCommission — Member มีแค่ Commission (MGMT-21)
func ValidateMemberCommission(v float64) PTViolation {
	if v < 0 || !isStep(v, CommissionStep) {
		return PTInvalidStep
	}
	if v > MaxCommission {
		return PTCommissionExceeded
	}
	return PTOK
}

// ValidateMemberPT — ค่าที่ผู้สร้างถือสู้กับ Member คนนี้ (MGMT-21 แก้ 2026-10-09) · 0 ถึงค่าที่ผู้สร้างได้รับ ทีละ 0.5%
func ValidateMemberPT(pt, creatorReceived float64) PTViolation {
	if !IsPTStep(pt) {
		return PTInvalidStep
	}
	if pt > creatorReceived {
		return PTExceedsReceived
	}
	return PTOK
}

// MemberRemain — ส่วนที่ผู้สร้างไม่ได้ถือสู้กับ Member คนนี้ = ค่าที่ผู้สร้างได้รับ − pt (MGMT-21 แก้ 2026-10-09)
func MemberRemain(creatorReceived, pt float64) float64 { return round4(creatorReceived - pt) }

// MinPTFromParent — ค่าต่ำสุดที่ผู้สร้างตั้งให้ลูกได้ (MGMT-24 · R1 แก้ 2026-10-09)
// = ค่าที่มากที่สุดระหว่าง ค่าที่ลูกให้ลูกของมันแต่ละคน และ pt ที่ลูกถือสู้กับ Member แต่ละคนที่ลูกสร้าง · ไม่มีเลย = 0
func MinPTFromParent(grandchildrenPTFromParent, memberPTs []float64) float64 {
	m := 0.0
	for _, v := range append(append([]float64{}, grandchildrenPTFromParent...), memberPTs...) {
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
