package security

import "testing"

func TestEscapeText(t *testing.T) {
	in := `<script>alert("xss")</script>`
	got := EscapeText(in)
	want := `&lt;script&gt;alert(&#34;xss&#34;)&lt;/script&gt;`
	if got != want {
		t.Fatalf("EscapeText() = %q, want %q", got, want)
	}
	if EscapeText("Кино") != "Кино" {
		t.Fatal("обычный текст не должен меняться")
	}
}
