package security

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"errors"
)

const (
	CSRFCookieName = "csrf_token"
	CSRFHeaderName = "X-CSRF-Token"
	csrfTokenBytes = 32
)

var (
	ErrCSRFMissing = errors.New("csrf token missing")
	ErrCSRFInvalid = errors.New("csrf token invalid")
)

func NewCSRFToken() (string, error) {
	buf := make([]byte, csrfTokenBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func CheckCSRF(cookieToken, headerToken string) error {
	if cookieToken == "" || headerToken == "" {
		return ErrCSRFMissing
	}
	if subtle.ConstantTimeCompare([]byte(cookieToken), []byte(headerToken)) != 1 {
		return ErrCSRFInvalid
	}
	return nil
}
