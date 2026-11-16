package security

import "testing"

func TestNewCSRFToken(t *testing.T) {
	a, err := NewCSRFToken()
	if err != nil || len(a) != csrfTokenBytes*2 {
		t.Fatalf("token = %q err = %v", a, err)
	}
	b, err := NewCSRFToken()
	if err != nil || a == b {
		t.Fatalf("токены должны отличаться: %q %q err=%v", a, b, err)
	}
}

func TestCheckCSRF(t *testing.T) {
	if err := CheckCSRF("aaa", "aaa"); err != nil {
		t.Fatalf("равные токены: %v", err)
	}
	if err := CheckCSRF("", "aaa"); err != ErrCSRFMissing {
		t.Fatalf("пустой cookie: %v", err)
	}
	if err := CheckCSRF("aaa", "bbb"); err != ErrCSRFInvalid {
		t.Fatalf("разные токены: %v", err)
	}
}
