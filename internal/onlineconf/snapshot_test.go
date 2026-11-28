package onlineconf

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCacheRefreshReplacesOnlyValidSnapshot(t *testing.T) {
	const userID = "11111111-1111-4111-8111-111111111111"
	responses := make(chan string, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/internal/v1/snapshot" {
			t.Errorf("unexpected snapshot request: %s %s", r.Method, r.URL)
		}
		_, _ = io.WriteString(w, <-responses)
	}))
	defer server.Close()
	client, err := NewClient(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	cache := &Cache{}
	defaults := PopularTracks{Limit: 6, Enabled: true}
	if got := cache.Snapshot().PopularTracks(userID); got != defaults {
		t.Fatalf("before first snapshot: %+v", got)
	}

	responses <- `{"entries":[
		{"scope":"catalog","key":"popular_tracks_limit","kind":"config","value":3},
		{"scope":"catalog","key":"popular_tracks_enabled","kind":"flag","enabled":true,
		 "rollout_percent":0,"force_on":["` + userID + `"],"force_off":[]}
	]}`
	if err := cache.refresh(context.Background(), client); err != nil {
		t.Fatal(err)
	}
	working := cache.Snapshot()
	if got := working.PopularTracks(userID); got != (PopularTracks{Limit: 3, Enabled: true}) {
		t.Fatalf("included user: %+v", got)
	}
	if got := working.PopularTracks(""); got != (PopularTracks{Limit: 3, Enabled: false}) {
		t.Fatalf("guest: %+v", got)
	}

	responses <- `{"entries":[{"scope":"catalog","key":"popular_tracks_limit","kind":"config","value":21}]}`
	if err := cache.refresh(context.Background(), client); err == nil || cache.Snapshot() != working {
		t.Fatal("invalid snapshot replaced the last good snapshot")
	}
	responses <- `{"entries":[]}`
	if err := cache.refresh(context.Background(), client); err != nil {
		t.Fatal(err)
	}
	if got := cache.Snapshot().PopularTracks(userID); got != defaults {
		t.Fatalf("after removing entries: %+v", got)
	}
}

func TestPopularTracksFlagRules(t *testing.T) {
	const included = "11111111-1111-4111-8111-111111111111"
	const excluded = "22222222-2222-4222-8222-222222222222"
	const other = "33333333-3333-4333-8333-333333333333"
	rule := flag{
		enabled:  true,
		percent:  100,
		forceOn:  map[string]struct{}{included: {}, excluded: {}},
		forceOff: map[string]struct{}{excluded: {}},
	}
	if rule.allows(excluded) || !rule.allows(other) || !rule.allows("") {
		t.Fatal("100% rollout, guest or force-off precedence is incorrect")
	}
	rule.percent = 0
	if !rule.allows(included) || rule.allows(other) || rule.allows("") {
		t.Fatal("0% rollout or force-on precedence is incorrect")
	}
	rule.enabled = false
	if rule.allows(included) {
		t.Fatal("force-on overrode the global off switch")
	}
	rule.enabled = true
	rule.percent = 50
	want := rule.allows(other)
	for range 20 {
		if got := rule.allows(other); got != want {
			t.Fatal("the same user changed buckets")
		}
	}
}
