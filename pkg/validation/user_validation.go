// Package validation проверяет данные, пришедшие от клиента.
package validation

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

const (
	emailMinLen = 3
	emailMaxLen = 254

	// Минимум считается в символах, максимум — в байтах: столько принимает bcrypt.
	passwordMinRunes = 8
	passwordMaxBytes = 72

	displayNameMinLen = 2
	displayNameMaxLen = 50
)

// SignupInput — данные формы регистрации от клиента.
type SignupInput struct {
	Email       string
	Password    string
	DisplayName string
}

// SignupResult — результат проверки: нормализованные поля и ошибки по каждому
// непрошедшему полю. Пустой Fields означает, что данные в порядке.
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

// Signup проверяет и нормализует данные регистрации in.
// Проверяются все поля сразу, а не до первой ошибки.
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

// LoginInput — данные формы входа от клиента.
type LoginInput struct {
	Email    string
	Password string
}

// LoginResult — результат проверки: нормализованный email и ошибки по полям.
type LoginResult struct {
	Email    string
	Password string
	Fields   map[string]string
}

// Valid сообщает, прошли ли данные проверку.
func (r LoginResult) Valid() bool {
	return len(r.Fields) == 0
}

// Login проверяет, что поля входа заполнены, и нормализует email.
// Правила длины и состава пароля здесь не применяются: аккаунт мог быть заведён
// до их ужесточения, да и отказ по ним подсказывал бы требования к паролю.
func Login(in LoginInput) LoginResult {
	result := LoginResult{
		Email:    NormalizeEmail(in.Email),
		Password: in.Password,
		Fields:   make(map[string]string),
	}

	if result.Email == "" {
		result.Fields["email"] = "Укажите email"
	}
	if result.Password == "" {
		result.Fields["password"] = "Укажите пароль"
	}

	return result
}

// NormalizeEmail приводит адрес к виду, в котором он хранится и ищется:
// без пробелов по краям и в нижнем регистре.
func NormalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

// checkEmail возвращает текст ошибки или пустую строку, если адрес годится.
// Проверка по RFC 5322 не делается: адрес подтверждается письмом.
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

// checkPassword возвращает текст ошибки или пустую строку, если пароль годится.
func checkPassword(password string) string {
	if password == "" {
		return "Укажите пароль"
	}
	// Символы, а не байты: иначе пароль из шести кириллических букв
	// прошёл бы как достаточно длинный.
	if utf8.RuneCountInString(password) < passwordMinRunes {
		return fmt.Sprintf("Пароль должен быть не короче %d символов", passwordMinRunes)
	}
	// Предел bcrypt задан в байтах, поэтому в символах он разный:
	// называть число пользователю было бы враньём.
	if len(password) > passwordMaxBytes {
		return "Пароль слишком длинный"
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
		return "Пароль должен содержать хотя бы одну букву и одну цифру"
	}

	return ""
}

// checkDisplayName возвращает текст ошибки или пустую строку, если имя годится.
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
