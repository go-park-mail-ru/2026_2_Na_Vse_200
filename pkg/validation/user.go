// Package validation проверяет данные, пришедшие от клиента.
// Правила описаны в docs/api.md, раздел 5, и повторяются на фронтенде.
package validation

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

const (
	emailMinLen = 3
	emailMaxLen = 254

	// Верхняя граница пароля равна пределу bcrypt: всё после 72 байт он игнорирует.
	passwordMinLen = 8
	passwordMaxLen = 72

	displayNameMinLen = 2
	displayNameMaxLen = 50
)

// SignupInput — данные формы регистрации до проверки.
type SignupInput struct {
	Email       string
	Password    string
	DisplayName string
}

// SignupResult — результат проверки. Email и DisplayName нормализованы,
// Fields содержит все найденные ошибки разом.
type SignupResult struct {
	Email       string
	DisplayName string
	Password    string
	Fields      map[string]string
}

// Valid сообщает, прошли ли данные проверку.
func (r SignupResult) Valid() bool {
	return len(r.Fields) == 0
}

// Signup проверяет и нормализует данные регистрации.
func Signup(in SignupInput) SignupResult {
	result := SignupResult{
		Email:       NormalizeEmail(in.Email),
		DisplayName: strings.TrimSpace(in.DisplayName),
		Password:    in.Password, // пробелы могут быть частью пароля
		Fields:      make(map[string]string),
	}

	if msg := checkEmail(result.Email); msg != "" {
		result.Fields["email"] = msg
	}
	if msg := checkPassword(result.Password); msg != "" {
		result.Fields["password"] = msg
	}
	if msg := checkDisplayName(result.DisplayName); msg != "" {
		result.Fields["display_name"] = msg
	}

	return result
}

// NormalizeEmail обрезает пробелы и приводит адрес к нижнему регистру,
// чтобы Andrey@mail.ru и andrey@mail.ru считались одним аккаунтом.
func NormalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

// checkEmail отсекает явный мусор. Полная проверка по RFC 5322 бессмысленна:
// адрес всё равно подтверждается письмом.
func checkEmail(email string) string {
	const msg = "Некорректный email"

	if email == "" {
		return "Укажите email"
	}
	if len(email) < emailMinLen || len(email) > emailMaxLen {
		return msg
	}

	local, domain, found := strings.Cut(email, "@")
	if !found || local == "" || domain == "" {
		return msg
	}
	if strings.Contains(domain, "@") {
		return msg
	}
	if strings.ContainsAny(email, " \t\n\r") {
		return msg
	}

	// Точка в домене обязательна и не может стоять с краю.
	dot := strings.Index(domain, ".")
	if dot <= 0 || dot == len(domain)-1 {
		return msg
	}

	return ""
}

func checkPassword(password string) string {
	const msg = "Пароль должен быть не короче 8 символов и содержать букву и цифру"

	if password == "" {
		return "Укажите пароль"
	}
	// Длина в байтах: предел bcrypt тоже в байтах.
	if len(password) < passwordMinLen || len(password) > passwordMaxLen {
		return msg
	}

	var hasLetter, hasDigit bool
	for _, r := range password {
		switch {
		case unicode.IsLetter(r):
			hasLetter = true
		case unicode.IsDigit(r):
			hasDigit = true
		}
	}
	if !hasLetter || !hasDigit {
		return msg
	}

	return ""
}

func checkDisplayName(name string) string {
	const msg = "Имя должно содержать от 2 до 50 символов"

	if name == "" {
		return "Укажите имя"
	}

	// Считаем символы, а не байты: кириллица занимает по два байта.
	length := utf8.RuneCountInString(name)
	if length < displayNameMinLen || length > displayNameMaxLen {
		return msg
	}

	return ""
}
