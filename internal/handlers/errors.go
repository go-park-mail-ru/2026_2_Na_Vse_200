package handlers

import (
	"log"
	"net/http"

	"github.com/go-park-mail-ru/2026_2_Na_Vse_200/internal/apimessage"
	"github.com/go-park-mail-ru/2026_2_Na_Vse_200/pkg/response"
)

// Причины отказа для лога. Клиенту они не уходят: он видит общий текст,
// иначе по ответу можно было бы изучать устройство сервиса.
const (
	_errHashPassword    = "ошибка хеширования пароля"
	_errCreateUser      = "ошибка создания пользователя"
	_errAutoSignIn      = "ошибка автовхода после регистрации"
	_errFindUser        = "ошибка поиска пользователя"
	_errVerifyPassword  = "ошибка проверки пароля"
	_errSignIn          = "ошибка входа пользователя"
	_errReadSession     = "ошибка чтения сессии"
	_errDeleteSession   = "ошибка удаления сессии"
	_errNoSessionCookie = "запрос без cookie сессии"
	_errUnknownEmail    = "вход по незарегистрированному email"
	_errWrongPassword   = "вход с неверным паролем"
	_errSessionRejected = "сессия неизвестна или истекла"
	_errUserGone        = "сессия ссылается на удалённый аккаунт"
)

// logCause записывает причину отказа. Ошибка может отсутствовать: не за каждым
// отказом стоит сбой — клиент, например, просто не прислал cookie.
func logCause(cause string, err error) {
	if err != nil {
		log.Printf("%s: %v", cause, err)
		return
	}
	log.Print(cause)
}

// writeInternalError логирует причину и отдаёт клиенту общий текст без деталей.
func writeInternalError(w http.ResponseWriter, cause string, err error) {
	logCause(cause, err)
	response.WriteError(w, http.StatusInternalServerError,
		apimessage.CodeInternal, apimessage.MsgInternal)
}

// writeUnauthorized отвечает одинаково на отсутствующую, неизвестную
// и истёкшую сессию: разница подсказывала бы, что идентификатор угадан верно.
func writeUnauthorized(w http.ResponseWriter, cause string, err error) {
	logCause(cause, err)
	response.WriteError(w, http.StatusUnauthorized,
		apimessage.CodeUnauthorized, apimessage.MsgUnauthorized)
}

// writeInvalidCredentials отвечает одинаково на неизвестный email и на неверный
// пароль: по разнице ответов перебирают зарегистрированные адреса.
func writeInvalidCredentials(w http.ResponseWriter, cause string) {
	logCause(cause, nil)
	response.WriteError(w, http.StatusUnauthorized,
		apimessage.CodeInvalidCredentials, apimessage.MsgInvalidCredentials)
}
