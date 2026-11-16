package catalog

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"2026_2_Na_Vse_200/internal/models"
)

type catalogStub struct {
	calls []string
	fail  string
	err   error
	ctx   context.Context
	t     *testing.T
}

func (s *catalogStub) call(ctx context.Context, name string, limit int) error {
	s.t.Helper()
	if ctx != s.ctx || limit != 20 {
		s.t.Fatal("context or limit changed")
	}
	s.calls = append(s.calls, name)
	if s.fail == name {
		return s.err
	}
	return nil
}
func (s *catalogStub) HomeTracks(ctx context.Context, n int) ([]models.Track, error) {
	return []models.Track{{ID: "track"}}, s.call(ctx, "tracks", n)
}
func (s *catalogStub) HomeArtists(ctx context.Context, n int) ([]models.Artist, error) {
	return []models.Artist{{ID: "artist"}}, s.call(ctx, "artists", n)
}
func (s *catalogStub) HomeAlbums(ctx context.Context, n int) ([]models.Album, error) {
	return []models.Album{{ID: "album"}}, s.call(ctx, "albums", n)
}
func TestHome(t *testing.T) {
	for _, tc := range []struct {
		fail  string
		calls []string
	}{
		{"tracks", []string{"tracks"}},
		{"artists", []string{"tracks", "artists"}},
		{"albums", []string{"tracks", "artists", "albums"}},
		{"", []string{"tracks", "artists", "albums"}},
	} {
		t.Run("fail_"+tc.fail, func(t *testing.T) {
			ctx := context.WithValue(context.Background(), struct{}{}, "request")
			failure := errors.New("storage down")
			repo := &catalogStub{fail: tc.fail, err: failure, ctx: ctx, t: t}
			result, err := New(repo).Home(ctx)
			if !reflect.DeepEqual(repo.calls, tc.calls) {
				t.Fatalf("calls: %v", repo.calls)
			}
			if tc.fail != "" {
				if !errors.Is(err, failure) || len(result.Tracks) != 0 {
					t.Fatalf("partial result or wrong error: %+v %v", result, err)
				}
			} else if err != nil || result.Tracks[0].ID != "track" || result.Artists[0].ID != "artist" || result.Albums[0].ID != "album" {
				t.Fatalf("result: %+v %v", result, err)
			}
		})
	}
}
