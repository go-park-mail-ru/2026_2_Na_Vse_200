package security

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"errors"
)

const (
	// CSRFCookieName — cookie с CSRF-токеном (без HttpOnly: фронт читает и кладёт в заголовок).
	CSRFCookieName = "csrf_token"
	// CSRFHeaderName — заголовок с копией токена (double-submit cookie).
	CSRFHeaderName = "X-CSRF-Token"
	csrfTokenBytes = 32
)

var (
	ErrCSRFMissing = errors.New("csrf token missing")
	ErrCSRFInvalid = errors.New("csrf token invalid")
)

// NewCSRFToken возвращает криптостойкий токен для CSRF.
func NewCSRFToken() (string, error) {
	buf := make([]byte, csrfTokenBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

// CheckCSRF сравнивает токен из cookie и заголовка (double-submit).
func CheckCSRF(cookieToken, headerToken string) error {
	if cookieToken == "" || headerToken == "" {
		return ErrCSRFMissing
	}
	if subtle.ConstantTimeCompare([]byte(cookieToken), []byte(headerToken)) != 1 {
		return ErrCSRFInvalid
	}
	return nil
}
