package postgres

import (
	"reflect"
	"testing"

	"github.com/go-park-mail-ru/2026_2_Na_Vse_200/internal/models"
)

func TestTrackStatusDefaultsToDraft(t *testing.T) {
	if got := trackStatus(""); got != "draft" {
		t.Fatalf("trackStatus empty = %q, want draft", got)
	}
	if got := trackStatus("published"); got != "published" {
		t.Fatalf("trackStatus published = %q, want published", got)
	}
}

func TestNormalizePlaylistItemsMatchesStoredPositions(t *testing.T) {
	input := []models.PlaylistItem{
		{TrackID: "first"},
		{TrackID: "second", Position: 4},
		{TrackID: "third", Position: -1},
	}

	got, err := normalizePlaylistItems(input)
	if err != nil {
		t.Fatalf("normalizePlaylistItems: %v", err)
	}
	want := []models.PlaylistItem{
		{TrackID: "first", Position: 1},
		{TrackID: "second", Position: 4},
		{TrackID: "third", Position: 3},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("normalized items = %+v, want %+v", got, want)
	}
	if input[0].Position != 0 || input[2].Position != -1 {
		t.Fatalf("normalization changed caller's slice: %+v", input)
	}
}
