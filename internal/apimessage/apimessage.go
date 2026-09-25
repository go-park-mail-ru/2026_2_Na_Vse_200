// Package apimessage хранит коды ошибок и тексты для пользователя.
// Коды — часть контракта API, менять их нельзя без обновления docs/api.md.
package apimessage

// Коды, по которым клиент различает ситуации.
const (
	CodeInvalidJSON        = "invalid_json"
	CodeValidationFailed   = "validation_failed"
	CodeEmailTaken         = "email_taken"
	CodeInvalidCredentials = "invalid_credentials"
	CodeUnauthorized       = "unauthorized"
	CodeNotFound           = "not_found"
	CodeMethodNotAllowed   = "method_not_allowed"
	CodeInternal           = "internal_error"
)

// Тексты, которые показываются пользователю как есть.
const (
	MsgInvalidJSON      = "Некорректный JSON в теле запроса"
	MsgValidationFailed = "Проверьте правильность заполнения полей"
	MsgEmailTaken       = "Пользователь с таким email уже существует"
	// Один текст на неизвестный email и на неверный пароль: иначе по разнице
	// ответов перебирают зарегистрированные адреса.
	MsgInvalidCredentials = "Неверный email или пароль"
	MsgUnauthorized       = "Требуется вход"
	MsgNotFound           = "Адрес не найден"
	MsgMethodNotAllowed   = "Метод не поддерживается этим адресом"
	MsgInternal           = "Внутренняя ошибка сервера"
)
