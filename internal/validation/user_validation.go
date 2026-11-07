package validation

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

const (
	_emailMinLen = 3
	_emailMaxLen = 254

	// Минимальная длина считается в символах, предел bcrypt — в байтах.
	_passwordMinRunes = 8
	_passwordMaxBytes = 72

	_displayNameMinLen = 2
	_displayNameMaxLen = 50
)

// SignupInput содержит данные регистрации.
type SignupInput struct {
	Email       string
	Password    string
	DisplayName string
}

// SignupResult содержит нормализованные данные и ошибки полей.
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

// Signup нормализует данные регистрации и проверяет все поля за один вызов.
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

// ProfileUpdateInput содержит данные изменения профиля.
type ProfileUpdateInput struct {
	Email       string
	DisplayName string
}

// ProfileUpdateResult содержит нормализованные данные и ошибки полей.
type ProfileUpdateResult struct {
	Email       string
	DisplayName string
	Fields      map[string]string
}

func ProfileUpdate(in ProfileUpdateInput) ProfileUpdateResult {
	result := ProfileUpdateResult{
		Email:       NormalizeEmail(in.Email),
		DisplayName: strings.TrimSpace(in.DisplayName),
		Fields:      make(map[string]string),
	}
	if err := checkEmail(result.Email); err != nil {
		result.Fields["email"] = err.Error()
	}
	if err := checkDisplayName(result.DisplayName); err != nil {
		result.Fields["display_name"] = err.Error()
	}
	return result
}

// LoginInput содержит данные входа.
type LoginInput struct {
	Email    string
	Password string
}

// LoginResult содержит нормализованные данные входа и ошибки полей.
type LoginResult struct {
	Email    string
	Password string
	Fields   map[string]string
}

// Valid сообщает, прошли ли данные проверку.
func (r LoginResult) Valid() bool {
	return len(r.Fields) == 0
}

// Login нормализует email и проверяет заполненность полей.
// Требования к составу пароля применяются только при регистрации.
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

// NormalizeEmail удаляет пробелы по краям и переводит email в нижний регистр.
func NormalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

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

	dot := strings.Index(domain, ".")
	if dot <= 0 || dot == len(domain)-1 {
		return ErrEmailInvalid
	}

	return nil
}

func checkPassword(password string) error {
	if password == "" {
		return ErrPasswordRequired
	}

	if utf8.RuneCountInString(password) < _passwordMinRunes {
		return ErrPasswordShort
	}

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

func checkDisplayName(name string) error {
	if name == "" {
		return ErrDisplayNameRequired
	}

	length := utf8.RuneCountInString(name)
	if length < _displayNameMinLen || length > _displayNameMaxLen {
		return ErrDisplayNameLength
	}

	return nil
}
