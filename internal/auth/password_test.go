package auth

import (
	"strings"
	"testing"
)

// Минимальная цена: тесты проверяют поведение, а не скорость вычислений.
func testHasher() BcryptHasher {
	return NewBcryptHasherWithCost(MinCost)
}

func TestHashAndVerify(t *testing.T) {
	hasher := testHasher()
	const password = "muzyka2026"

	hash, err := hasher.Hash(password)
	if err != nil {
		t.Fatalf("Hash: неожиданная ошибка: %v", err)
	}

	if !hasher.Verify(password, hash) {
		t.Error("Verify вернул false для правильного пароля")
	}
	if hasher.Verify("muzyka2027", hash) {
		t.Error("Verify вернул true для неправильного пароля")
	}
}

// В хеше не должно быть исходного пароля даже частично.
func TestHashDoesNotContainPassword(t *testing.T) {
	hasher := testHasher()
	const password = "muzyka2026"

	hash, err := hasher.Hash(password)
	if err != nil {
		t.Fatalf("Hash: неожиданная ошибка: %v", err)
	}

	if strings.Contains(hash, password) {
		t.Errorf("пароль виден в хеше: %s", hash)
	}
}

// Соль генерируется на каждый вызов, поэтому хеши одного пароля различаются.
func TestHashIsSaltedDifferently(t *testing.T) {
	hasher := testHasher()
	const password = "muzyka2026"

	first, err := hasher.Hash(password)
	if err != nil {
		t.Fatalf("Hash: неожиданная ошибка: %v", err)
	}
	second, err := hasher.Hash(password)
	if err != nil {
		t.Fatalf("Hash: неожиданная ошибка: %v", err)
	}

	if first == second {
		t.Error("два хеша одного пароля совпали — соль не случайная")
	}
	if !hasher.Verify(password, first) || !hasher.Verify(password, second) {
		t.Error("Verify не принял пароль для одного из хешей")
	}
}

// Формат bcrypt: $2a$<цена>$<соль и хеш>, 60 символов.
func TestHashFormat(t *testing.T) {
	hash, err := testHasher().Hash("muzyka2026")
	if err != nil {
		t.Fatalf("Hash: неожиданная ошибка: %v", err)
	}

	if !strings.HasPrefix(hash, "$2a$") {
		t.Errorf("хеш не похож на bcrypt: %s", hash)
	}
	if len(hash) != 60 {
		t.Errorf("длина хеша = %d, ожидалось 60: %s", len(hash), hash)
	}
}

// Цена записана в самом хеше, поэтому сиды с ценой 12 проверяются тем же кодом.
func TestVerifyAcceptsOtherCost(t *testing.T) {
	const password = "muzyka2026"

	hash, err := NewBcryptHasherWithCost(6).Hash(password)
	if err != nil {
		t.Fatalf("Hash: неожиданная ошибка: %v", err)
	}

	if !NewBcryptHasherWithCost(MinCost).Verify(password, hash) {
		t.Error("хеш с другой ценой не прошёл проверку")
	}
}

// Пароли длиннее 72 байт bcrypt не принимает.
func TestHashTooLongPassword(t *testing.T) {
	if _, err := testHasher().Hash(strings.Repeat("a", 73)); err == nil {
		t.Error("Hash принял пароль длиннее 72 байт, ожидалась ошибка")
	}
}

// Испорченный хеш означает «пароль не подошёл», а не панику.
func TestVerifyBrokenHash(t *testing.T) {
	hasher := testHasher()

	tests := []struct {
		name string
		hash string
	}{
		{name: "пустая строка", hash: ""},
		{name: "не хеш", hash: "простотекст"},
		{name: "обрезанный хеш", hash: "$2a$04$abc"},
		{name: "чужой формат", hash: "sha256$c29sdA$aGFzaA"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if hasher.Verify("muzyka2026", tt.hash) {
				t.Errorf("Verify вернул true для непригодного хеша %q", tt.hash)
			}
		})
	}
}
