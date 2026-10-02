// Package apperr คือ error ที่มี business code ติดมาด้วย
//
// วิธีใช้:
//
//	// ประกาศครั้งเดียวในไฟล์ของ module (pkg/apperr/{module}_errors.go)
//	var ErrInsufficientAgentBalance = apperr.New(220026, "ยอดเงินเอเย่นต์ไม่พอ", "Insufficient agent balance")
//
//	// service คืน error ตรงๆ
//	if bal < amount { return apperr.ErrInsufficientAgentBalance }
//
//	// ต้องการแนบสาเหตุไว้ดูใน log (client ไม่เห็น)
//	return apperr.ErrInsufficientAgentBalance.Wrap(err)
//
//	// controller ไม่ต้องรู้จัก error รายตัว — response.Error(c, err) แปลงให้เอง
//
// ห้ามแยก error ด้วยการเทียบข้อความ (strings.Contains) — ใช้ errors.Is(err, apperr.ErrX) เท่านั้น
package apperr

import (
	"errors"
	"fmt"
	"net/http"
	"sort"
	"sync"
)

type Error struct {
	Code       int
	MsgTH      string
	MsgEN      string
	HTTPStatus int
	cause      error
}

func (e *Error) Error() string {
	if e.cause != nil {
		return fmt.Sprintf("[%d] %s: %v", e.Code, e.MsgEN, e.cause)
	}
	return fmt.Sprintf("[%d] %s", e.Code, e.MsgEN)
}

func (e *Error) Unwrap() error { return e.cause }

// Is ทำให้ errors.Is(wrapped, ErrX) เป็นจริง แม้ ErrX จะถูก Wrap ไปแล้ว — เทียบกันที่ Code
func (e *Error) Is(target error) bool {
	var t *Error
	if errors.As(target, &t) {
		return t.Code == e.Code
	}
	return false
}

// Wrap คืนสำเนาของ error เดิมพร้อมสาเหตุ (ตัวแปร global ไม่ถูกแก้)
func (e *Error) Wrap(cause error) *Error {
	cp := *e
	cp.cause = cause
	return &cp
}

var (
	mu       sync.RWMutex
	registry = map[int]*Error{}
)

// New ประกาศ error code ใหม่ — code ซ้ำ = panic ตอน boot (กันสอง module แย่ง code เดียวกัน)
// HTTP status เริ่มต้นคือ 200 ตาม contract เดิม (client อ่านผลจาก field code)
func New(code int, msgTH, msgEN string) *Error {
	return NewWithStatus(http.StatusOK, code, msgTH, msgEN)
}

// NewWithStatus เหมือน New แต่กำหนด HTTP status เอง (ใช้กับ error ระดับ infra เช่น 500/503)
func NewWithStatus(status, code int, msgTH, msgEN string) *Error {
	mu.Lock()
	defer mu.Unlock()
	if _, dup := registry[code]; dup {
		panic(fmt.Sprintf("apperr: duplicate code %d", code))
	}
	e := &Error{Code: code, MsgTH: msgTH, MsgEN: msgEN, HTTPStatus: status}
	registry[code] = e
	return e
}

// From ดึง *Error ออกจาก err — ถ้าไม่ใช่ error ที่ประกาศไว้ คืน ErrInternal (พร้อมแนบ err เดิมไว้ log)
func From(err error) *Error {
	var e *Error
	if errors.As(err, &e) {
		return e
	}
	return ErrInternal.Wrap(err)
}

// All คืน error ทั้งหมดเรียงตาม code — ใช้ gen เอกสาร response code
func All() []*Error {
	mu.RLock()
	defer mu.RUnlock()
	out := make([]*Error, 0, len(registry))
	for _, e := range registry {
		out = append(out, e)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Code < out[j].Code })
	return out
}
