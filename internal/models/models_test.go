package models

import (
	"testing"
	"time"
)

func TestSessionIsExpired(t *testing.T) {
	expires := time.Date(2026, 10, 4, 12, 0, 0, 0, time.UTC)
	session := Session{ExpiresAt: expires}

	tests := []struct {
		name string
		now  time.Time
		want bool
	}{
		{name: "до срока", now: expires.Add(-time.Minute), want: false},
		{name: "ровно в срок", now: expires, want: true},
		{name: "после срока", now: expires.Add(time.Minute), want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := session.IsExpired(tt.now); got != tt.want {
				t.Errorf("IsExpired = %v, ожидалось %v", got, tt.want)
			}
		})
	}
}
