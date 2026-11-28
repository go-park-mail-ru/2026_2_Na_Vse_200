package onlineconfadmin

import (
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path"
	"strings"

	"2026_2_Na_Vse_200/internal/delivery/middleware"
	"2026_2_Na_Vse_200/internal/delivery/response"
	"2026_2_Na_Vse_200/internal/models"
	"2026_2_Na_Vse_200/internal/onlineconf"

	"github.com/rs/zerolog"
)

const APIPrefix = "/api/v1/admin/onlineconf/"

const maxEntryBytes = 64 << 10

type Handler struct {
	client *onlineconf.Client
	dist   string
	admins map[string]struct{}
}

func New(client *onlineconf.Client, dist, adminIDs string) (*Handler, error) {
	admins := make(map[string]struct{})
	if strings.TrimSpace(adminIDs) != "" {
		for _, rawID := range strings.Split(adminIDs, ",") {
			id := strings.TrimSpace(rawID)
			if !onlineconf.ValidUserID(id) {
				return nil, fmt.Errorf("ONLINECONF_ADMIN_IDS должен содержать UUID через запятую")
			}
			admins[strings.ToLower(id)] = struct{}{}
		}
	}
	return &Handler{client: client, dist: dist, admins: admins}, nil
}

func (h *Handler) Register(mux *http.ServeMux, authenticate middleware.Middleware) {
	protect := func(next http.Handler) http.Handler {
		protected := authenticate(h.requireAdmin(next))
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Cache-Control", "private, no-store")
			protected.ServeHTTP(w, r)
		})
	}
	mux.Handle("GET /admin", protect(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/admin/", http.StatusTemporaryRedirect)
	})))
	mux.Handle("GET /admin/", protect(http.HandlerFunc(h.serveSPA)))
	mux.Handle("GET "+APIPrefix+"entries", protect(http.HandlerFunc(h.list)))
	mux.Handle("PUT "+APIPrefix+"entries/{scope}/{key}", protect(middleware.WithCSRF(http.HandlerFunc(h.save))))
}

func (h *Handler) requireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session, ok := middleware.SessionFromContext(r.Context())
		if !ok {
			response.WriteError(r.Context(), w, http.StatusUnauthorized, models.CodeUnauthorized, models.MsgUnauthorized)
			return
		}
		if _, allowed := h.admins[strings.ToLower(string(session.UserID))]; !allowed {
			response.WriteError(r.Context(), w, http.StatusForbidden, models.CodeForbidden, models.MsgForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (h *Handler) serveSPA(w http.ResponseWriter, r *http.Request) {
	if h.dist == "" {
		writeUnavailable(w, r)
		return
	}
	root, err := os.OpenRoot(h.dist)
	if err != nil {
		writeUnavailable(w, r)
		return
	}
	defer root.Close()

	dist := root.FS()
	name := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/admin/"), "/")
	if name == "" {
		name = "index.html"
	}
	if !fs.ValidPath(name) || strings.Contains(name, `\`) {
		http.NotFound(w, r)
		return
	}
	info, err := fs.Stat(dist, name)
	if err == nil && info.Mode().IsRegular() {
		http.ServeFileFS(w, r, dist, name)
		return
	}
	if strings.HasPrefix(name, "assets/") || path.Ext(name) != "" || (err != nil && !os.IsNotExist(err)) {
		http.NotFound(w, r)
		return
	}
	http.ServeFileFS(w, r, dist, "index.html")
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	if h.client == nil {
		writeUnavailable(w, r)
		return
	}
	status, result, err := h.client.ListEntries(r.Context(), r.URL.Query())
	writeResult(w, r, status, result, err)
}

func (h *Handler) save(w http.ResponseWriter, r *http.Request) {
	scope, key := r.PathValue("scope"), r.PathValue("key")
	if !onlineconf.ValidName(scope) || !onlineconf.ValidName(key) {
		response.WriteError(r.Context(), w, http.StatusBadRequest, "validation_failed", "Некорректный scope или ключ")
		return
	}
	if h.client == nil {
		writeUnavailable(w, r)
		return
	}
	payload, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxEntryBytes))
	if err != nil {
		response.WriteError(r.Context(), w, http.StatusBadRequest, "validation_failed", "Не удалось прочитать тело запроса или превышен его размер")
		return
	}
	session, _ := middleware.SessionFromContext(r.Context())
	status, result, err := h.client.SaveEntry(r.Context(), scope, key, bytes.NewReader(payload), string(session.UserID))
	writeResult(w, r, status, result, err)
}

func writeResult(w http.ResponseWriter, r *http.Request, status int, body []byte, err error) {
	if err != nil {
		zerolog.Ctx(r.Context()).Warn().Err(err).Str("operation", "onlineconf.admin").Msg("ошибка запроса к OnlineConf")
		writeUnavailable(w, r)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(status)
	_, _ = w.Write(body)
}

func writeUnavailable(w http.ResponseWriter, r *http.Request) {
	response.WriteError(r.Context(), w, http.StatusServiceUnavailable, "onlineconf_unavailable", "OnlineConf временно недоступен")
}
