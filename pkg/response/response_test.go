package response

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWriteJSON(t *testing.T) {
	w := httptest.NewRecorder()

	WriteJSON(w, http.StatusCreated, map[string]string{"email": "andrey@example.com"})

	if w.Code != http.StatusCreated {
		t.Errorf("WriteJSON: статус = %d, ожидался %d", w.Code, http.StatusCreated)
	}

	gotType := w.Header().Get("Content-Type")
	wantType := "application/json; charset=utf-8"
	if gotType != wantType {
		t.Errorf("WriteJSON: Content-Type = %q, ожидался %q", gotType, wantType)
	}

	// Тело разбираем, а не сравниваем строкой: порядок полей и пробелы не гарантированы.
	var body map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("WriteJSON: тело не разобралось как JSON: %v, тело: %s", err, w.Body.String())
	}
	if body["email"] != "andrey@example.com" {
		t.Errorf("WriteJSON: email = %q, ожидался %q", body["email"], "andrey@example.com")
	}
}

func TestWriteError(t *testing.T) {
	w := httptest.NewRecorder()

	WriteError(w, http.StatusConflict, "email_taken", "Пользователь с таким email уже существует")

	if w.Code != http.StatusConflict {
		t.Errorf("WriteError: статус = %d, ожидался %d", w.Code, http.StatusConflict)
	}

	got := decode(t, w)
	if got.Error.Code != "email_taken" {
		t.Errorf("WriteError: code = %q, ожидался %q", got.Error.Code, "email_taken")
	}
	if got.Error.Message == "" {
		t.Error("WriteError: message пустой, клиенту нечего показать пользователю")
	}
	if got.Error.Fields != nil {
		t.Errorf("WriteError: fields = %v, ожидалось отсутствие ключа для ошибки не про поля", got.Error.Fields)
	}
}

func TestWriteFieldsError(t *testing.T) {
	w := httptest.NewRecorder()

	WriteFieldsError(w, http.StatusBadRequest, "validation_failed", "Проверьте поля", map[string]string{
		"email":    "Некорректный email",
		"password": "Пароль должен быть не короче 8 символов",
	})

	if w.Code != http.StatusBadRequest {
		t.Errorf("WriteFieldsError: статус = %d, ожидался %d", w.Code, http.StatusBadRequest)
	}

	got := decode(t, w)
	if got.Error.Code != "validation_failed" {
		t.Errorf("WriteFieldsError: code = %q, ожидался %q", got.Error.Code, "validation_failed")
	}
	if len(got.Error.Fields) != 2 {
		t.Errorf("WriteFieldsError: полей = %d, ожидалось 2: %v", len(got.Error.Fields), got.Error.Fields)
	}
	if got.Error.Fields["email"] == "" {
		t.Error("WriteFieldsError: для поля email нет текста ошибки")
	}
}

// decode разбирает тело ответа как ошибку единого формата API.
func decode(t *testing.T, w *httptest.ResponseRecorder) ErrorResponse {
	t.Helper()

	var got ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("тело не разобралось как ошибка API: %v, тело: %s", err, w.Body.String())
	}
	return got
}
