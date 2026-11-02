package middleware

import (
	"net/http"

	"2026_2_Na_Vse_200/internal/delivery/response"
	"2026_2_Na_Vse_200/internal/models"
)

// WithJSONErrors заменяет текстовые ответы 404 и 405 на JSON.
func WithJSONErrors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(&errorInterceptor{ResponseWriter: w}, r)
	})
}

// errorInterceptor заменяет тело ответа, сохраняя заголовки ServeMux.
type errorInterceptor struct {
	http.ResponseWriter
	replaced bool
}

func (w *errorInterceptor) WriteHeader(code int) {
	switch code {
	case http.StatusNotFound:
		w.replaced = true
		response.WriteError(w.ResponseWriter, code, models.CodeNotFound, models.MsgNotFound)
	case http.StatusMethodNotAllowed:
		w.replaced = true
		response.WriteError(w.ResponseWriter, code, models.CodeMethodNotAllowed, models.MsgMethodNotAllowed)
	default:
		w.ResponseWriter.WriteHeader(code)
	}
}

// Write отбрасывает исходное тело после подмены ответа и сообщает об успешной записи.
func (w *errorInterceptor) Write(b []byte) (int, error) {
	if w.replaced {
		return len(b), nil
	}
	return w.ResponseWriter.Write(b)
}
