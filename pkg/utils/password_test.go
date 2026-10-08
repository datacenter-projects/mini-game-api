package utils

import (
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func TestSetPasswordCost(t *testing.T) {
	original := passwordCost
	t.Cleanup(func() { SetPasswordCost(original) })

	SetPasswordCost(bcrypt.MinCost)
	hash, err := HashPassword("secret-123")
	if err != nil {
		t.Fatal(err)
	}
	if cost, _ := bcrypt.Cost([]byte(hash)); cost != bcrypt.MinCost {
		t.Fatalf("hash cost = %d, want %d", cost, bcrypt.MinCost)
	}
	if !CheckPassword(hash, "secret-123") || CheckPassword(hash, "wrong") {
		t.Fatal("CheckPassword mismatch")
	}

	CheckPasswordDummy("x")
	if cost, _ := bcrypt.Cost(dummyHash); cost != bcrypt.MinCost {
		t.Fatalf("dummyHash cost = %d, want %d", cost, bcrypt.MinCost)
	}
}
