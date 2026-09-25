package auth

import (
	"strings"
	"testing"
)

// Во всех тестах берём минимальную цену: проверяем поведение, а не скорость
// вычислений, а с ценой по умолчанию каждый хеш считался бы десятки миллисекунд.
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

// bcrypt генерирует соль сам, поэтому один и тот же пароль даёт разные хеши.
// Иначе по совпадающим хешам было бы видно, у кого одинаковые пароли.
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

// Формат bcrypt: $2a$<цена>$<соль и хеш>, всего 60 символов.
// Соль лежит внутри строки, отдельная колонка в БД под неё не нужна.
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

// Цена записана внутри хеша, поэтому пароль, захешированный с другой ценой,
// проверяется без дополнительных настроек. Это важно: в сидах БД лежат
// хеши с ценой 12, а сервис считает новые с ценой по умолчанию.
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

// Пароль длиннее 72 байт bcrypt не принимает — ровно поэтому в правилах
// валидации стоит такая же верхняя граница.
func TestHashTooLongPassword(t *testing.T) {
	if _, err := testHasher().Hash(strings.Repeat("a", 73)); err == nil {
		t.Error("Hash принял пароль длиннее 72 байт, ожидалась ошибка")
	}
}

// Испорченный хеш из базы не должен ронять сервер: это просто «пароль не подошёл».
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
