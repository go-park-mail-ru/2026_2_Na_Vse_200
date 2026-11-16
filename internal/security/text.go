package security

import "html"

func EscapeText(value string) string {
	return html.EscapeString(value)
}
