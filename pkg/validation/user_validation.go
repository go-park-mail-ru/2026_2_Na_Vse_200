// Package validation проверяет данные, пришедшие от клиента.
package validation

import (
	"errors"
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

const (
	_emailMinLen = 3
	_emailMaxLen = 254

	// Минимум считается в символах, максимум — в байтах: столько принимает bcrypt.
	_passwordMinRunes = 8
	_passwordMaxBytes = 72

	_displayNameMinLen = 2
	_displayNameMaxLen = 50
)

var (
	ErrEmailRequired = errors.New("Укажите email")
	ErrEmailInvalid  = errors.New("Некорректный email")

	ErrPasswordRequired = errors.New("Укажите пароль")
	ErrPasswordShort    = fmt.Errorf("Пароль должен быть не короче %d символов", _passwordMinRunes)
	ErrPasswordLong     = errors.New("Пароль слишком длинный")
	ErrPasswordSimple   = errors.New("Пароль должен содержать хотя бы одну букву и одну цифру")

	ErrDisplayNameRequired = errors.New("Укажите имя")
	ErrDisplayNameLength   = fmt.Errorf("Имя должно содержать от %d до %d символов",
		_displayNameMinLen, _displayNameMaxLen)
)

// SignupInput — данные формы регистрации от клиента.
type SignupInput struct {
	Email       string
	Password    string
	DisplayName string
}

// SignupResult — результат проверки: нормализованные поля и причина отказа
// по каждому непрошедшему полю. Пустой Fields означает, что данные в порядке.
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
		Password:    strings.TrimSpace(in.Password),
		Fields:      make(map[string]string),
	}

	if err := checkEmail(result.Email); err != nil {
		result.Fields["email"] = err.Error()
	}

	if err := checkPassword(result.Password); err != nil {
		result.Fields["password"] = err.Error()
	}

	if err := checkDisplayName(result.DisplayName); err != nil {
		result.Fields["display_name"] = err.Error()
	}

	return result
}

// LoginInput — данные формы входа от клиента.
type LoginInput struct {
	Email    string
	Password string
}

// LoginResult — результат проверки: нормализованный email и причины отказа по полям.
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
		Password: strings.TrimSpace(in.Password),
		Fields:   make(map[string]string),
	}

	if result.Email == "" {
		result.Fields["email"] = ErrEmailRequired.Error()
	}

	if result.Password == "" {
		result.Fields["password"] = ErrPasswordRequired.Error()
	}

	return result
}

// NormalizeEmail приводит адрес к виду, в котором он хранится и ищется:
// без пробелов по краям и в нижнем регистре.
func NormalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

// checkEmail сообщает, годится ли адрес.
// Проверка по RFC 5322 не делается: адрес подтверждается письмом.
func checkEmail(email string) error {
	if email == "" {
		return ErrEmailRequired
	}

	if len(email) < _emailMinLen || len(email) > _emailMaxLen {
		return ErrEmailInvalid
	}

	local, domain, found := strings.Cut(email, "@")
	if !found || local == "" || domain == "" {
		return ErrEmailInvalid
	}

	if strings.Contains(domain, "@") {
		return ErrEmailInvalid
	}

	if strings.ContainsAny(email, " \t\n\r") {
		return ErrEmailInvalid
	}

	// Точка в домене обязательна и не может стоять с краю.
	dot := strings.Index(domain, ".")
	if dot <= 0 || dot == len(domain)-1 {
		return ErrEmailInvalid
	}

	return nil
}

// checkPassword сообщает, годится ли пароль.
func checkPassword(password string) error {
	if password == "" {
		return ErrPasswordRequired
	}

	// Символы, а не байты: иначе пароль из шести кириллических букв
	// прошёл бы как достаточно длинный.
	if utf8.RuneCountInString(password) < _passwordMinRunes {
		return ErrPasswordShort
	}

	// Предел bcrypt задан в байтах, поэтому в символах он разный:
	// называть число пользователю было бы враньём.
	if len(password) > _passwordMaxBytes {
		return ErrPasswordLong
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
		return ErrPasswordSimple
	}

	return nil
}

// checkDisplayName сообщает, годится ли имя.
func checkDisplayName(name string) error {
	if name == "" {
		return ErrDisplayNameRequired
	}

	// Считаем символы, а не байты: кириллица занимает по два байта.
	length := utf8.RuneCountInString(name)
	if length < _displayNameMinLen || length > _displayNameMaxLen {
		return ErrDisplayNameLength
	}

	return nil
}
