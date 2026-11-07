package security

import "html"

// EscapeText экранирует пользовательский текст для безопасного показа как обычный текст,
// а не HTML. Санитайзить HTML здесь нельзя: формы принимают только текст.
func EscapeText(value string) string {
	return html.EscapeString(value)
}
