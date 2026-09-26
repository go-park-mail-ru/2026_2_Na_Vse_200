package auth

import "testing"

func TestNewSessionIDUnique(t *testing.T) {
	if first, second := NewSessionID(), NewSessionID(); first == second {
		t.Errorf("два вызова вернули один идентификатор: %s", first)
	}
}
