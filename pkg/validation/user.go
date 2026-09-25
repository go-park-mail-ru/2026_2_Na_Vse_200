// Package validation проверяет входные данные пользователя.
//
// Функции здесь ничего не знают про HTTP: на вход приходят строки, на выходе —
// карта «поле → текст ошибки». Благодаря этому правила покрываются тестами
// без поднятия сервера, а обработчику остаётся только отдать результат клиенту.
//
// Пакет лежит в pkg рядом с другими утилитами: проверки не привязаны
// ни к одной ручке и переиспользуются везде, где приходят данные от клиента.
//
// Правила совпадают с docs/api.md, раздел 5, и должны совпадать с проверками
// на фронтенде. Клиентская валидация — удобство для пользователя, серверная —
// обязательна: запрос может прийти и мимо формы.
package validation

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// Границы длин из контракта.
const (
	emailMinLen = 3
	emailMaxLen = 254

	// Верхняя граница пароля в 72 символа выбрана не случайно: bcrypt
	// игнорирует всё после 72 байт. Если команда перейдёт на него,
	// поведение не изменится.
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

// SignupResult — то, что получилось после проверки и нормализации.
type SignupResult struct {
	// Email приведён к нижнему регистру и очищен от пробелов.
	Email string
	// DisplayName очищен от пробелов по краям.
	DisplayName string
	// Password не меняется: пробелы внутри и по краям могут быть частью пароля.
	Password string
	// Fields пуст, если данные в порядке. Иначе содержит все ошибки сразу,
	// чтобы форма подсветила проблемные поля за один проход.
	Fields map[string]string
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
		Password:    in.Password,
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

// NormalizeEmail приводит адрес к каноничному виду: без пробелов по краям
// и в нижнем регистре. Иначе Andrey@mail.ru и andrey@mail.ru стали бы
// двумя разными аккаунтами.
func NormalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

// checkEmail проверяет адрес по правилам контракта.
//
// Полная проверка синтаксиса email по RFC 5322 практически бесполезна:
// она пропускает несуществующие адреса и отсекает редкие валидные.
// Реально адрес подтверждается письмом, поэтому здесь только защита
// от очевидного мусора.
func checkEmail(email string) string {
	const msg = "Некорректный email"

	if email == "" {
		return "Укажите email"
	}
	if len(email) < emailMinLen || len(email) > emailMaxLen {
		return msg
	}

	local, domain, found := strings.Cut(email, "@")
	if !found {
		return msg
	}
	// Cut режет по первой собаке, поэтому вторую ищем в остатке.
	if strings.Contains(domain, "@") {
		return msg
	}
	if local == "" || domain == "" {
		return msg
	}
	if strings.ContainsAny(email, " \t\n\r") {
		return msg
	}

	// В домене нужна точка, и она не может стоять с краю: mail.ru годится,
	// mail. или .ru — нет.
	dot := strings.Index(domain, ".")
	if dot <= 0 || dot == len(domain)-1 {
		return msg
	}

	return ""
}

// checkPassword требует длину и хотя бы одну букву с цифрой.
func checkPassword(password string) string {
	const msg = "Пароль должен быть не короче 8 символов и содержать букву и цифру"

	if password == "" {
		return "Укажите пароль"
	}
	// Считаем в байтах: ограничение bcrypt тоже в байтах.
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

// checkDisplayName проверяет отображаемое имя.
func checkDisplayName(name string) string {
	const msg = "Имя должно содержать от 2 до 50 символов"

	if name == "" {
		return "Укажите имя"
	}

	// Считаем символы, а не байты: в UTF-8 кириллическая буква занимает
	// два байта, и по len("Ян") имя из двух букв не прошло бы проверку.
	length := utf8.RuneCountInString(name)
	if length < displayNameMinLen || length > displayNameMaxLen {
		return msg
	}

	return ""
}
