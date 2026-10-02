// Package migrations เก็บไฟล์ goose migration (.sql) ทั้งหมด
//
// กฎ:
//   - สร้างไฟล์ใหม่ด้วย `make migration name=<module>_<what>` เท่านั้น (ได้ชื่อ timestamp อัตโนมัติ)
//   - ไฟล์ที่ merge เข้า dev แล้ว ห้ามแก้ — ต้องการเปลี่ยน schema ให้สร้างไฟล์ใหม่
//   - ทุกไฟล์ต้องมีทั้ง -- +goose Up และ -- +goose Down
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS
