// Package ptcalculation คือกฎการคิด PT สุดท้ายต่อ Member (Force → Remain → คืน Superadmin) — spec: docs/modules/pt_calculation.md
// pure function ล้วน (กฎข้อ 6) · warning ส่งกลับให้ service log เอง
package ptcalculation

import (
	"errors"
	"fmt"
	"math"
)

// FullPT — PT ทั้งหมด = 100% (Superadmin ได้รับ — MGMT-22)
const FullPT = 100.0

// ErrInvalidChain — ข้อมูลในสายผิดรูป คิดต่อไม่ได้ (PTC-08 / PTC-09) · service แปลงเป็น 508601 + log รายละเอียดจาก error
var ErrInvalidChain = errors.New("pt chain invalid")

// round4 — ปัด 4 ตำแหน่ง (ตรงกับ utils.Round4 · core ห้าม import pkg/utils เพราะดึง fiber มาด้วย) — PTC-07
func round4(v float64) float64 { return math.Round(v*10000) / 10000 }

// Layer — ชั้นฝั่ง agent หนึ่งชั้นในสาย · index 0 = Superadmin ลงไปถึงผู้สร้าง Member (L0 … Lk)
// ค่า Force / Remain / AcceptsPT / IsCSM ของ L0 ไม่ใช้
type Layer struct {
	Received  float64 // P[i] = agent_game_settings.pt_from_parent · L0 ถือว่า 100 เสมอ
	Force     float64 // ผู้สร้างของชั้นนี้ตั้งให้ · 0 = ไม่ตั้ง
	Remain    float64 // ผู้สร้างของชั้นนี้ตั้งให้ · 0 = ไม่ตั้ง
	AcceptsPT bool    // agent_game_settings.status — false = ถือ 0 + ไม่โดน Force ของตัวเอง (PTC-02A)
	IsCSM     bool    // Company Seamless Master — ข้าม Force เสมอ (PTC-04A)
}

// Input — สายของ Member หนึ่งคนในเกมหนึ่ง
type Input struct {
	Layers   []Layer // L0 … Lk (อย่างน้อย 2 ชั้น — Superadmin สร้าง Member ไม่ได้ MGMT-02)
	MemberPT float64 // user_member_game_settings.pt = Lk ถือสู้กับ Member คนนี้
}

// WarningKind — เรื่องที่คิดต่อได้แต่ควร log ไว้ตรวจข้อมูล
type WarningKind int

const (
	WarnCSMForce     WarningKind = iota + 1 // แถวของ CSM มี force ≠ 0 — ข้ามไป (PTC-04A)
	WarnCappedByLeft                        // ส่วนที่ขาดเกิน left — ถือเพิ่มได้แค่ left (PTC-10)
)

// Warning — ชั้นที่เกิดเรื่อง + ค่าที่ต้องการ / ค่าที่ได้จริง
type Warning struct {
	Kind  WarningKind
	Layer int
	Want  float64
	Got   float64
}

// Result — PT สุดท้ายของ L0 … Lk (รวม = 100) + warning
type Result struct {
	PT       []float64
	Warnings []Warning
}

// Calculate — คิด PT สุดท้ายของทุกชั้น (PTC-02 – PTC-08)
//
//	hold[i] = P[i] − P[i+1] (i < k) · hold[k] = MemberPT · ไม่รับ PT = 0 · left = 100 − Σ hold
//	ไล่ i = k … 1: Force (Li ถือเพิ่ม) แล้ว Remain (L(i−1) ถือเพิ่ม) · เหลือคืน L0
func Calculate(in Input) (Result, error) {
	if err := validate(in); err != nil {
		return Result{}, err
	}
	n := len(in.Layers)
	k := n - 1
	received := func(i int) float64 {
		if i == 0 {
			return FullPT
		}
		return in.Layers[i].Received
	}

	eff := make([]float64, n)
	left := FullPT
	for i := 0; i < n; i++ {
		h := in.MemberPT
		if i < k {
			h = round4(received(i) - received(i+1))
		}
		if i > 0 && !in.Layers[i].AcceptsPT { // PTC-02A
			h = 0
		}
		eff[i] = h
		left = round4(left - h)
	}

	var warnings []Warning
	take := func(i int, target float64) float64 { // ส่วนที่ขาดจาก target เทียบกับ Σ eff[i..k] · ไม่เกิน left (PTC-10)
		s := 0.0
		for j := i; j <= k; j++ {
			s += eff[j]
		}
		s = round4(s)
		if s >= target {
			return 0
		}
		want := round4(target - s)
		if want > left {
			warnings = append(warnings, Warning{Kind: WarnCappedByLeft, Layer: i, Want: want, Got: left})
			return left
		}
		return want
	}

	for i := k; i >= 1; i-- {
		l := in.Layers[i]
		switch {
		case l.IsCSM: // PTC-04A
			if l.Force != 0 {
				warnings = append(warnings, Warning{Kind: WarnCSMForce, Layer: i, Want: l.Force})
			}
		case l.AcceptsPT && l.Force > 0: // PTC-04 · ไม่รับ PT = ไม่โดน Force ของตัวเอง (PTC-02A)
			add := take(i, l.Force)
			eff[i] = round4(eff[i] + add)
			left = round4(left - add)
		}
		if l.Remain > 0 { // PTC-05 — ผู้สร้างถือเพิ่ม (ผู้สร้างไม่รับ PT ก็ยังทำงาน — PTC-02A)
			add := take(i, l.Remain)
			eff[i-1] = round4(eff[i-1] + add)
			left = round4(left - add)
		}
	}
	eff[0] = round4(eff[0] + left) // PTC-06

	sum := 0.0
	for i, v := range eff { // PTC-08
		if v < 0 {
			return Result{}, fmt.Errorf("%w: layer %d final pt %v < 0", ErrInvalidChain, i, v)
		}
		sum += v
	}
	if round4(sum) != FullPT {
		return Result{}, fmt.Errorf("%w: final pt sum %v != 100", ErrInvalidChain, round4(sum))
	}
	return Result{PT: eff, Warnings: warnings}, nil
}

// validate — PTC-09: สายอย่างน้อย 2 ชั้น · ค่าที่ได้รับไม่เกินผู้สร้าง · pt ของ Member ไม่เกินที่ผู้สร้างได้รับ · ค่า % อยู่ใน 0 – 100
func validate(in Input) error {
	n := len(in.Layers)
	if n < 2 {
		return fmt.Errorf("%w: chain has %d layers", ErrInvalidChain, n)
	}
	prev := FullPT
	for i := 1; i < n; i++ {
		l := in.Layers[i]
		for _, v := range []float64{l.Received, l.Force, l.Remain} {
			if v < 0 || v > FullPT {
				return fmt.Errorf("%w: layer %d value %v out of 0–100", ErrInvalidChain, i, v)
			}
		}
		if l.Received > prev {
			return fmt.Errorf("%w: layer %d received %v > parent %v", ErrInvalidChain, i, l.Received, prev)
		}
		prev = l.Received
	}
	if in.MemberPT < 0 || in.MemberPT > prev {
		return fmt.Errorf("%w: member pt %v out of 0–%v", ErrInvalidChain, in.MemberPT, prev)
	}
	return nil
}
