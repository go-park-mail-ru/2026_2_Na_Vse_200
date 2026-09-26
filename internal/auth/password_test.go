package auth

import (
	"errors"
	"strings"
	"testing"
)

func TestHashAndVerify(t *testing.T) {
	h := NewBcryptHasherWithCost(MinCost)
	const pass = "muzyka2026"

	hash, err := h.Hash(pass)
	if err != nil {
		t.Fatalf("Hash: %v", err)
	}

	ok, err := h.Verify(pass, hash)
	if err != nil {
		t.Fatalf("Verify верный пароль: %v", err)
	}

	if !ok {
		t.Error("верный пароль не прошёл проверку")
	}

	ok, err = h.Verify("muzyka2027", hash)
	if err != nil {
		t.Fatalf("Verify неверный пароль: %v", err)
	}

	if ok {
		t.Error("неверный пароль прошёл проверку")
	}
}

// cost зашит в сам хеш, поэтому проверять можно любым хешером.
func TestVerifyOtherCost(t *testing.T) {
	const pass = "muzyka2026"

	hash, err := NewBcryptHasherWithCost(6).Hash(pass)
	if err != nil {
		t.Fatalf("Hash: %v", err)
	}

	ok, err := NewBcryptHasherWithCost(MinCost).Verify(pass, hash)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}

	if !ok {
		t.Error("хеш с cost=6 не прошёл проверку хешером с cost=MinCost")
	}
}

func TestHashTooLong(t *testing.T) {
	h := NewBcryptHasherWithCost(MinCost)
	long := strings.Repeat("a", MaxPasswordBytes+1)

	_, err := h.Hash(long)
	if err == nil {
		t.Fatal("Hash принял пароль длиннее предела")
	}

	if !errors.Is(err, ErrPasswordTooLong) {
		t.Errorf("Hash вернул %v, ожидалась ErrPasswordTooLong", err)
	}
}

// Битый хеш в хранилище - это ошибка
func TestVerifyBrokenHash(t *testing.T) {
	h := NewBcryptHasherWithCost(MinCost)

	cases := []string{
		"",
		"простотекст",
		"$2a$04$abc",
		"sha256$c29sdA$aGFzaA",
	}

	for _, hash := range cases {
		ok, err := h.Verify("muzyka2026", hash)
		if ok {
			t.Errorf("Verify принял битый хеш %q", hash)
		}

		if err == nil {
			t.Errorf("Verify не вернул ошибку для %q", hash)
		}
	}
}
