package utils

import "golang.org/x/crypto/bcrypt"

const passwordCost = 12

func HashPassword(plain string) (string, error) {
	h, err := bcrypt.GenerateFromPassword([]byte(plain), passwordCost)
	return string(h), err
}

// CheckPassword คืน true เมื่อรหัสตรงกับ hash
func CheckPassword(hash, plain string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain)) == nil
}

// dummyHash ใช้เทียบตอนไม่พบ user ให้เวลาตอบเท่ากับกรณีรหัสผิด (กันเดา username จากเวลา)
var dummyHash, _ = bcrypt.GenerateFromPassword([]byte("dummy-password-for-timing"), passwordCost)

func CheckPasswordDummy(plain string) {
	_ = bcrypt.CompareHashAndPassword(dummyHash, []byte(plain))
}
