package onlineconf

import (
	"context"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"net/http"
	"regexp"
	"strings"
	"sync/atomic"
	"time"

	"github.com/rs/zerolog"
)

const (
	catalogScope  = "catalog"
	trackLimitKey = "popular_tracks_limit"
	trackFlagKey  = "popular_tracks_enabled"
	pollInterval  = 10 * time.Second
)

var (
	entryName = regexp.MustCompile(`^[a-z][a-z0-9_-]*$`)
	userUUID  = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
)

func ValidName(name string) bool { return entryName.MatchString(name) }

func ValidUserID(id string) bool { return userUUID.MatchString(id) }

type Cache struct {
	current atomic.Pointer[Snapshot]
}

// Snapshot возвращает одну версию настроек на весь запрос.
func (c *Cache) Snapshot() *Snapshot {
	return c.current.Load()
}

// Run сразу запрашивает снимок, затем обновляет его до отмены контекста.
func (c *Cache) Run(ctx context.Context, client *Client, logger zerolog.Logger) {
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for ctx.Err() == nil {
		if err := c.refresh(ctx, client); err != nil && ctx.Err() == nil {
			logger.Warn().Err(err).Str("operation", "onlineconf.refresh").Msg("не удалось обновить настройки")
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (c *Cache) refresh(ctx context.Context, client *Client) error {
	status, body, err := client.request(ctx, http.MethodGet, "/internal/v1/snapshot", "", nil, "")
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("OnlineConf: снимок вернул HTTP %d", status)
	}
	snapshot, err := parseSnapshot(body)
	if err != nil {
		return err
	}
	c.current.Store(snapshot)
	return nil
}

type Snapshot struct {
	trackLimit int
	trackFlag  *flag
}

type flag struct {
	enabled  bool
	percent  int
	forceOn  map[string]struct{}
	forceOff map[string]struct{}
}

type PopularTracks struct {
	Limit   int  `json:"popular_tracks_limit"`
	Enabled bool `json:"popular_tracks_enabled"`
}

func (s *Snapshot) NeedsUser() bool {
	return s != nil && s.trackFlag != nil && s.trackFlag.enabled
}

func (s *Snapshot) PopularTracks(userID string) PopularTracks {
	if s == nil {
		return PopularTracks{Limit: 6, Enabled: true}
	}
	return PopularTracks{Limit: s.trackLimit, Enabled: s.trackFlag.allows(userID)}
}

func (f *flag) allows(userID string) bool {
	switch {
	case f == nil:
		return true
	case !f.enabled:
		return false
	case userID == "":
		return f.percent == 100
	}
	userID = strings.ToLower(userID)
	if _, excluded := f.forceOff[userID]; excluded {
		return false
	}
	if _, included := f.forceOn[userID]; included {
		return true
	}
	hash := fnv.New64a()
	_, _ = hash.Write([]byte(catalogScope + "/" + trackFlagKey + "/" + userID))
	return int(hash.Sum64()%100) < f.percent
}

type snapshotEntry struct {
	Scope    string          `json:"scope"`
	Key      string          `json:"key"`
	Kind     string          `json:"kind"`
	Value    json.RawMessage `json:"value"`
	Enabled  *bool           `json:"enabled"`
	Percent  *int            `json:"rollout_percent"`
	ForceOn  []string        `json:"force_on"`
	ForceOff []string        `json:"force_off"`
}

func parseSnapshot(body []byte) (*Snapshot, error) {
	var data struct {
		Entries []snapshotEntry `json:"entries"`
	}
	if json.Unmarshal(body, &data) != nil || data.Entries == nil {
		return nil, fmt.Errorf("OnlineConf: неверный формат снимка")
	}
	snapshot := &Snapshot{trackLimit: 6}
	seen := make(map[string]struct{}, len(data.Entries))
	for _, entry := range data.Entries {
		if !ValidName(entry.Scope) || !ValidName(entry.Key) {
			return nil, fmt.Errorf("OnlineConf: неверный scope или key в снимке")
		}
		id := entry.Scope + "/" + entry.Key
		if _, exists := seen[id]; exists {
			return nil, fmt.Errorf("OnlineConf: повтор ключа в снимке")
		}
		seen[id] = struct{}{}

		switch entry.Kind {
		case "config":
			if len(entry.Value) == 0 {
				return nil, fmt.Errorf("OnlineConf: отсутствует значение конфига")
			}
			if id == catalogScope+"/"+trackLimitKey {
				var limit int
				if json.Unmarshal(entry.Value, &limit) != nil || limit < 1 || limit > 20 {
					return nil, fmt.Errorf("OnlineConf: неверный лимит популярных треков")
				}
				snapshot.trackLimit = limit
			}
		case "flag":
			parsed, err := entry.parseFlag()
			if err != nil {
				return nil, err
			}
			if id == catalogScope+"/"+trackFlagKey {
				snapshot.trackFlag = parsed
			}
		default:
			return nil, fmt.Errorf("OnlineConf: неизвестный вид записи")
		}
	}
	return snapshot, nil
}

func (e snapshotEntry) parseFlag() (*flag, error) {
	if e.Enabled == nil || e.Percent == nil || *e.Percent < 0 || *e.Percent > 100 || e.ForceOn == nil || e.ForceOff == nil {
		return nil, fmt.Errorf("OnlineConf: неверный формат флага")
	}
	forceOn, err := parseUsers(e.ForceOn)
	if err != nil {
		return nil, err
	}
	forceOff, err := parseUsers(e.ForceOff)
	if err != nil {
		return nil, err
	}
	return &flag{enabled: *e.Enabled, percent: *e.Percent, forceOn: forceOn, forceOff: forceOff}, nil
}

func parseUsers(ids []string) (map[string]struct{}, error) {
	users := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		if !ValidUserID(id) {
			return nil, fmt.Errorf("OnlineConf: неверный UUID пользователя")
		}
		users[strings.ToLower(id)] = struct{}{}
	}
	return users, nil
}
