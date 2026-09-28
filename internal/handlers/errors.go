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
	errHashPassword    = "ошибка хеширования пароля"
	errCreateUser      = "ошибка создания пользователя"
	errAutoSignIn      = "ошибка автовхода после регистрации"
	errFindUser        = "ошибка поиска пользователя"
	errVerifyPassword  = "ошибка проверки пароля"
	errSignIn          = "ошибка входа пользователя"
	errReadSession     = "ошибка чтения сессии"
	errDeleteSession   = "ошибка удаления сессии"
	errNoSessionCookie = "запрос без cookie сессии"
	errUnknownEmail    = "вход по незарегистрированному email"
	errWrongPassword   = "вход с неверным паролем"
	errSessionRejected = "сессия неизвестна или истекла"
	errUserGone        = "сессия ссылается на удалённый аккаунт"
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
