package apperr

import "net/http"

// Code สำเร็จ — ไม่ใช่ error แต่ประกาศไว้ที่นี่ให้ทุก code อยู่ที่เดียว
const CodeSuccess = 200

// Common errors ใช้ได้ทุก module
// error ของแต่ละ module ให้แยกไฟล์ {module}_errors.go และจองช่วง code ใน docs/ERROR_CODES.md ก่อน
var (
	ErrBadRequest   = New(400, "คำขอไม่ถูกต้อง", "Bad request")
	ErrUnauthorized = New(401, "ไม่ได้รับอนุญาต", "Unauthorized")
	ErrForbidden    = New(403, "ไม่มีสิทธิ์เข้าถึง", "Forbidden")
	ErrNotFound     = New(404, "ไม่พบข้อมูล", "Not found")
	ErrConflict     = New(409, "ข้อมูลซ้ำ", "Conflict")
	ErrValidation   = New(422, "ข้อมูลไม่ถูกต้อง", "Validation failed")
	ErrTooMany      = NewWithStatus(http.StatusTooManyRequests, 429, "เรียกใช้งานบ่อยเกินไป", "Too many requests")
	ErrInternal     = NewWithStatus(http.StatusInternalServerError, 500, "เกิดข้อผิดพลาดภายในระบบ", "Internal server error")
	ErrUnavailable  = NewWithStatus(http.StatusServiceUnavailable, 503, "ระบบยังไม่พร้อมให้บริการ", "Service unavailable")

	// ErrRouteNotFound คือ path ที่ไม่มีอยู่จริง (ต่างจาก ErrNotFound ที่หมายถึงหาข้อมูลไม่เจอ)
	ErrRouteNotFound = NewWithStatus(http.StatusNotFound, 404001, "ไม่พบ API ที่เรียก", "Route not found")
)
