package apperr

import "net/http"

// asset (module id 05) — spec: docs/modules/asset.md หัวข้อ 9
// สิทธิ์เมนูใช้ code ของ module ②: ErrNoMenuPermission (402303) · ErrDownlineNotFound (402402) ไม่ใช้แล้ว — BO ดูได้แค่ของตัวเอง (lead 2026-10-16)
// ตัด 405302, 405303 และ 405404 (banner เลื่อน) — เลขที่ตัดไม่ใช้ซ้ำ
var (
	// ErrAssetImageSize ใช้คู่กับ WithMessage เพื่อบอกขนาดที่ช่องนี้รับ (AST-17)
	ErrAssetImageSize    = New(405101, "ขนาดภาพไม่ตรงกับที่ช่องนี้รับ", "Image size is not accepted by this slot")
	ErrAssetFileType     = New(405102, "ชนิดไฟล์ไม่รองรับ (รับ png / jpeg / gif / webp)", "Unsupported file type (png, jpeg, gif and webp only)")
	ErrAssetFileTooLarge = New(405103, "ไฟล์ใหญ่เกิน 5 MB", "File is larger than 5 MB")

	ErrAssetNoPermission = New(405301, "ไม่มีสิทธิ์แก้หน้าตาของบัญชีนี้", "You cannot change the look of this account")
	ErrAssetSlotRetired  = New(405304, "ช่องนี้เลิกใช้แล้ว", "This slot is retired")

	ErrAssetSlotNotFound   = New(405401, "ไม่พบเกมหรือช่อง asset", "Game or asset slot not found")
	ErrAssetConfigNotFound = New(405402, "ไม่พบ asset", "Asset not found") // ตอบเหมือนกันทุกกรณี กันไล่เดา id (AST-23, AST-31)
	ErrAssetInUse          = New(405403, "asset นี้ยังถูกใช้งานอยู่ เลือกรูปอื่นหรือยกเลิกการเลือก (reset) ก่อนลบ", "This asset is still in use. Pick another one or reset the selection before deleting")

	ErrStorageUnavailable  = NewWithStatus(http.StatusServiceUnavailable, 505801, "ที่เก็บไฟล์ยังไม่พร้อม", "File storage is not available")
	ErrStorageUploadFailed = NewWithStatus(http.StatusBadGateway, 505802, "อัปโหลดไปที่เก็บไฟล์ไม่สำเร็จ", "Uploading to file storage failed")
)
