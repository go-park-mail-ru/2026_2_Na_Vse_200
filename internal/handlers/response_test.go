package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWriteJSON(t *testing.T) {
	w := httptest.NewRecorder()

	writeJSON(w, http.StatusCreated, map[string]string{"email": "andrey@example.com"})

	if w.Code != http.StatusCreated {
		t.Errorf("writeJSON: статус = %d, ожидался %d", w.Code, http.StatusCreated)
	}

	gotType := w.Header().Get("Content-Type")
	wantType := "application/json; charset=utf-8"
	if gotType != wantType {
		t.Errorf("writeJSON: Content-Type = %q, ожидался %q", gotType, wantType)
	}

	// Тело разбираем, а не сравниваем строкой: порядок полей и пробелы не гарантированы.
	var body map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("writeJSON: тело не разобралось как JSON: %v, тело: %s", err, w.Body.String())
	}
	if body["email"] != "andrey@example.com" {
		t.Errorf("writeJSON: email = %q, ожидался %q", body["email"], "andrey@example.com")
	}
}

func TestWriteError(t *testing.T) {
	w := httptest.NewRecorder()

	writeError(w, http.StatusConflict, CodeEmailTaken, "Пользователь с таким email уже существует")

	if w.Code != http.StatusConflict {
		t.Errorf("writeError: статус = %d, ожидался %d", w.Code, http.StatusConflict)
	}

	got := decodeError(t, w)
	if got.Error.Code != CodeEmailTaken {
		t.Errorf("writeError: code = %q, ожидался %q", got.Error.Code, CodeEmailTaken)
	}
	if got.Error.Message == "" {
		t.Error("writeError: message пустой, клиенту нечего показать пользователю")
	}
	if got.Error.Fields != nil {
		t.Errorf("writeError: fields = %v, ожидалось отсутствие ключа для ошибки не про поля", got.Error.Fields)
	}
}

func TestWriteValidationError(t *testing.T) {
	w := httptest.NewRecorder()

	writeValidationError(w, map[string]string{
		"email":    "Некорректный email",
		"password": "Пароль должен быть не короче 8 символов",
	})

	if w.Code != http.StatusBadRequest {
		t.Errorf("writeValidationError: статус = %d, ожидался %d", w.Code, http.StatusBadRequest)
	}

	got := decodeError(t, w)
	if got.Error.Code != CodeValidationFailed {
		t.Errorf("writeValidationError: code = %q, ожидался %q", got.Error.Code, CodeValidationFailed)
	}
	// Все невалидные поля должны приходить сразу, чтобы форма подсветилась за один раз.
	if len(got.Error.Fields) != 2 {
		t.Errorf("writeValidationError: полей = %d, ожидалось 2, получено: %v", len(got.Error.Fields), got.Error.Fields)
	}
	if got.Error.Fields["email"] == "" {
		t.Error("writeValidationError: для поля email нет текста ошибки")
	}
}

// decodeError разбирает тело ответа как ошибку единого формата API.
func decodeError(t *testing.T, w *httptest.ResponseRecorder) errorResponse {
	t.Helper()

	var got errorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("тело не разобралось как ошибка API: %v, тело: %s", err, w.Body.String())
	}
	return got
}
