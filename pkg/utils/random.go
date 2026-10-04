package utils

import (
	"crypto/rand"
	"math/big"
)

// RandomString สุ่มสตริงยาว n จาก alphabet ด้วย crypto/rand (กระจายเท่ากันทุกตัว)
func RandomString(alphabet string, n int) (string, error) {
	max := big.NewInt(int64(len(alphabet)))
	out := make([]byte, n)
	for i := range out {
		idx, err := rand.Int(rand.Reader, max)
		if err != nil {
			return "", err
		}
		out[i] = alphabet[idx.Int64()]
	}
	return string(out), nil
}
