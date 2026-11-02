package home

import (
	"testing"
	"time"
)

func TestOptionalString(t *testing.T) {
	if got := optionalString(""); got != nil {
		t.Errorf("для пустой строки вернулось %q, ожидался nil", *got)
	}

	const url = "https://cdn.example.com/covers/1001.jpg"
	got := optionalString(url)
	if got == nil {
		t.Fatal("для непустой строки вернулся nil")
	}

	if *got != url {
		t.Errorf("значение = %q, ожидалось %q", *got, url)
	}
}

func TestFormatReleaseDate(t *testing.T) {
	if got := formatReleaseDate(nil); got != nil {
		t.Errorf("для неизвестной даты вернулось %q, ожидался nil", *got)
	}

	// Смещение проверяет преобразование даты релиза в UTC.
	moscow := time.FixedZone("MSK", 3*60*60)
	date := time.Date(2024, 8, 1, 1, 30, 0, 0, moscow)

	got := formatReleaseDate(&date)
	if got == nil {
		t.Fatal("для известной даты вернулся nil")
	}

	if *got != "2024-07-31" {
		t.Errorf("дата = %q, ожидалась %q", *got, "2024-07-31")
	}
}
