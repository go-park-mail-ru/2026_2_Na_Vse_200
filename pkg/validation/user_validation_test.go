package validation

import (
	"strings"
	"testing"
)

func TestSignupValid(t *testing.T) {
	result := Signup(SignupInput{
		Email:       "  Andrey@Example.COM ",
		Password:    "muzyka2026",
		DisplayName: "  Андрей  ",
	})

	if !result.Valid() {
		t.Fatalf("данные должны быть валидны, получены ошибки: %v", result.Fields)
	}

	if result.Email != "andrey@example.com" {
		t.Errorf("Email = %q, ожидался %q", result.Email, "andrey@example.com")
	}

	if result.DisplayName != "Андрей" {
		t.Errorf("DisplayName = %q, ожидался %q", result.DisplayName, "Андрей")
	}

	if result.Password != "muzyka2026" {
		t.Errorf("Password = %q, пароль не должен меняться", result.Password)
	}
}

func TestSignupEmail(t *testing.T) {
	tests := []struct {
		name  string
		email string
		valid bool
	}{
		{name: "обычный", email: "andrey@example.com", valid: true},
		{name: "с плюсом", email: "andrey+music@example.com", valid: true},
		{name: "поддомен", email: "andrey@mail.example.com", valid: true},
		{name: "пустой", email: "", valid: false},
		{name: "без собаки", email: "andrey.example.com", valid: false},
		{name: "две собаки", email: "andrey@@example.com", valid: false},
		{name: "пустая часть до собаки", email: "@example.com", valid: false},
		{name: "пустой домен", email: "andrey@", valid: false},
		{name: "домен без точки", email: "andrey@example", valid: false},
		{name: "точка в конце домена", email: "andrey@example.", valid: false},
		{name: "точка в начале домена", email: "andrey@.com", valid: false},
		{name: "пробел внутри", email: "and rey@example.com", valid: false},
		{name: "слишком длинный", email: strings.Repeat("a", 250) + "@example.com", valid: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Signup(SignupInput{
				Email:       tt.email,
				Password:    "muzyka2026",
				DisplayName: "Андрей",
			})

			_, hasError := result.Fields["email"]
			if tt.valid && hasError {
				t.Errorf("email %q признан невалидным: %s", tt.email, result.Fields["email"])
			}

			if !tt.valid && !hasError {
				t.Errorf("email %q признан валидным, а не должен", tt.email)
			}
		})
	}
}

func TestSignupPassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
		valid    bool
	}{
		{name: "буквы и цифры", password: "muzyka2026", valid: true},
		{name: "ровно 8 символов", password: "muzyka26", valid: true},
		{name: "с символами", password: "muzyka-2026!", valid: true},
		{name: "пустой", password: "", valid: false},
		{name: "короткий", password: "muzyk26", valid: false},
		{name: "без цифр", password: "muzykamuzyka", valid: false},
		{name: "без букв", password: "202620262026", valid: false},
		{name: "длиннее 72 байт", password: strings.Repeat("a1", 40), valid: false},
		// Шесть кириллических букв — это 11 байт: по длине в байтах пароль
		// прошёл бы, хотя символов меньше восьми.
		{name: "шесть кириллических символов", password: "парол1", valid: false},
		{name: "восемь кириллических символов", password: "пароль12", valid: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Signup(SignupInput{
				Email:       "andrey@example.com",
				Password:    tt.password,
				DisplayName: "Андрей",
			})

			_, hasError := result.Fields["password"]
			if tt.valid && hasError {
				t.Errorf("пароль признан невалидным: %s", result.Fields["password"])
			}

			if !tt.valid && !hasError {
				t.Error("пароль признан валидным, а не должен")
			}
		})
	}
}

func TestSignupDisplayName(t *testing.T) {
	tests := []struct {
		name        string
		displayName string
		valid       bool
	}{
		{name: "кириллица", displayName: "Андрей", valid: true},
		{name: "латиница", displayName: "Andrey", valid: true},
		{name: "два символа кириллицей", displayName: "Ян", valid: true},
		{name: "ровно 50 символов", displayName: strings.Repeat("я", 50), valid: true},
		{name: "пустое", displayName: "", valid: false},
		{name: "только пробелы", displayName: "   ", valid: false},
		{name: "один символ", displayName: "Я", valid: false},
		{name: "51 символ", displayName: strings.Repeat("я", 51), valid: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Signup(SignupInput{
				Email:       "andrey@example.com",
				Password:    "muzyka2026",
				DisplayName: tt.displayName,
			})

			_, hasError := result.Fields["display_name"]
			if tt.valid && hasError {
				t.Errorf("имя %q признано невалидным: %s", tt.displayName, result.Fields["display_name"])
			}

			if !tt.valid && !hasError {
				t.Errorf("имя %q признано валидным, а не должно", tt.displayName)
			}
		})
	}
}

// Все ошибки приходят разом, а не по одной.
func TestSignupReportsAllErrorsAtOnce(t *testing.T) {
	result := Signup(SignupInput{
		Email:       "нет-собаки",
		Password:    "123",
		DisplayName: "",
	})

	if result.Valid() {
		t.Fatal("данные должны быть невалидны")
	}

	if len(result.Fields) != 3 {
		t.Errorf("полей с ошибками = %d, ожидалось 3: %v", len(result.Fields), result.Fields)
	}
	for _, field := range []string{"email", "password", "display_name"} {
		if result.Fields[field] == "" {
			t.Errorf("нет текста ошибки для поля %q", field)
		}
	}
}

func TestLoginValid(t *testing.T) {
	got := Login(LoginInput{Email: "  Andrey@Example.com ", Password: "muzyka2026"})

	if !got.Valid() {
		t.Fatalf("данные не прошли проверку: %v", got.Fields)
	}
	if got.Email != "andrey@example.com" {
		t.Errorf("email = %q, ожидался %q", got.Email, "andrey@example.com")
	}
}

func TestLoginEmptyFields(t *testing.T) {
	got := Login(LoginInput{Email: "   ", Password: ""})

	for _, field := range []string{"email", "password"} {
		if got.Fields[field] == "" {
			t.Errorf("нет ошибки по полю %q, получено: %v", field, got.Fields)
		}
	}
}

// На входе пароль проверяется только на заполненность: правила могли
// ужесточить уже после того, как аккаунт завели.
func TestLoginIgnoresPasswordRules(t *testing.T) {
	if got := Login(LoginInput{Email: "andrey@example.com", Password: "123"}); !got.Valid() {
		t.Errorf("короткий пароль отклонён на входе: %v", got.Fields)
	}
}
