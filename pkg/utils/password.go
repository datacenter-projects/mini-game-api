package utils

import (
	"sync"

	"golang.org/x/crypto/bcrypt"
)

// passwordCost ตั้งจาก env PASSWORD_COST ผ่าน SetPasswordCost ตอน boot (prod = 12, CI/test = bcrypt.MinCost)
var passwordCost = 12

// SetPasswordCost เรียกครั้งเดียวตอน boot ก่อนรับ request — ไม่ปลอดภัยถ้าเรียกพร้อมกับ HashPassword/CheckPasswordDummy
func SetPasswordCost(cost int) {
	passwordCost = cost
	dummyOnce = sync.Once{} // dummyHash ต้องใช้ cost เดียวกับ hash จริง ไม่งั้นเวลาตอบต่างกัน
}

func HashPassword(plain string) (string, error) {
	h, err := bcrypt.GenerateFromPassword([]byte(plain), passwordCost)
	return string(h), err
}

// CheckPassword คืน true เมื่อรหัสตรงกับ hash
func CheckPassword(hash, plain string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain)) == nil
}

// dummyHash ใช้เทียบตอนไม่พบ user ให้เวลาตอบเท่ากับกรณีรหัสผิด (กันเดา username จากเวลา)
// สร้างตอนใช้ครั้งแรก เพื่อให้ได้ cost ที่ตั้งจาก config แล้ว
var (
	dummyOnce sync.Once
	dummyHash []byte
)

func CheckPasswordDummy(plain string) {
	dummyOnce.Do(func() {
		dummyHash, _ = bcrypt.GenerateFromPassword([]byte("dummy-password-for-timing"), passwordCost)
	})
	_ = bcrypt.CompareHashAndPassword(dummyHash, []byte(plain))
}
