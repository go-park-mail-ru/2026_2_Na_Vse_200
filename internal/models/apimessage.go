package models

// Коды ошибок входят в контракт API.
const (
	CodeInvalidJSON        = "invalid_json"
	CodeValidationFailed   = "validation_failed"
	CodeEmailTaken         = "email_taken"
	CodeInvalidCredentials = "invalid_credentials"
	CodeUnauthorized       = "unauthorized"
	CodeForbidden          = "forbidden"
	CodeNotFound           = "not_found"
	CodeMethodNotAllowed   = "method_not_allowed"
	CodeInternal           = "internal_error"
)

const (
	MsgInvalidJSON        = "Некорректный JSON в теле запроса"
	MsgValidationFailed   = "Проверьте правильность заполнения полей"
	MsgEmailTaken         = "Пользователь с таким email уже существует"
	MsgInvalidCredentials = "Неверный email или пароль"
	MsgUnauthorized       = "Требуется вход"
	MsgForbidden          = "Недостаточно прав"
	MsgNotFound           = "Адрес не найден"
	MsgMethodNotAllowed   = "Метод не поддерживается этим адресом"
	MsgInternal           = "Внутренняя ошибка сервера"
)
