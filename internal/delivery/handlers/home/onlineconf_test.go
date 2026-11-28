package home

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"2026_2_Na_Vse_200/internal/models"
	"2026_2_Na_Vse_200/internal/onlineconf"
	"2026_2_Na_Vse_200/internal/usecase/catalog"

	"github.com/rs/zerolog"
)

type catalogFixture struct {
	result catalog.Home
}

func (c catalogFixture) Home(context.Context) (catalog.Home, error) {
	return c.result, nil
}

func TestHomeAppliesOnlineConfWithoutChangingQueue(t *testing.T) {
	const allowedUser = "11111111-1111-4111-8111-111111111111"
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `{"entries":[
			{"scope":"catalog","key":"popular_tracks_limit","kind":"config","value":3},
			{"scope":"catalog","key":"popular_tracks_enabled","kind":"flag","enabled":true,
			 "rollout_percent":0,"force_on":["`+allowedUser+`"],"force_off":[]}
		]}`)
	}))
	defer upstream.Close()
	client, err := onlineconf.NewClient(upstream.URL)
	if err != nil {
		t.Fatal(err)
	}
	cache := &onlineconf.Cache{}
	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Second)
	defer cancel()
	done := make(chan struct{})
	go func() {
		defer close(done)
		cache.Run(ctx, client, zerolog.Nop())
	}()
	defer func() {
		cancel()
		<-done
	}()

	catalogData := catalog.Home{
		Tracks:  []models.Track{{ID: "a"}, {ID: "b"}, {ID: "c"}, {ID: "d"}},
		Artists: []models.Artist{{ID: "artist"}},
		Albums:  []models.Album{{ID: "album"}},
	}
	handler := New(catalogFixture{result: catalogData}, cache, "session_id", func(_ context.Context, id models.SessionID) (models.Session, error) {
		switch id {
		case "allowed":
			return models.Session{UserID: allowedUser}, nil
		case "other":
			return models.Session{UserID: "22222222-2222-4222-8222-222222222222"}, nil
		default:
			return models.Session{}, models.ErrSessionExpired
		}
	})
	for cache.Snapshot() == nil {
		select {
		case <-ctx.Done():
			t.Fatal("initial snapshot was not loaded")
		case <-time.After(5 * time.Millisecond):
		}
	}

	for _, tc := range []struct {
		session string
		enabled bool
		popular int
	}{
		{"allowed", true, 3},
		{"other", false, 0},
		{"expired", false, 0},
		{"", false, 0},
	} {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/home", nil)
		req.Header.Set("X-OnlineConf-Actor", allowedUser)
		if tc.session != "" {
			req.AddCookie(&http.Cookie{Name: "session_id", Value: tc.session})
		}
		w := httptest.NewRecorder()
		handler.Home(w, req)
		var body homeResponse
		if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		if w.Code != http.StatusOK || w.Header().Get("Cache-Control") != "private, no-store" ||
			body.OnlineConf != (onlineconf.PopularTracks{Limit: 3, Enabled: tc.enabled}) ||
			body.PopularTracks == nil || len(body.PopularTracks) != tc.popular ||
			len(body.Tracks) != 4 || len(body.Artists) != 1 || len(body.Albums) != 1 {
			t.Errorf("session %q: status=%d body=%s", tc.session, w.Code, w.Body.String())
		}
	}
}
