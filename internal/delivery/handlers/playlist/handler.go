package playlist

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"mime"
	"net/http"
	"os"
	"strings"
	"unicode/utf8"
	"uuid"

	"2026_2_Na_Vse_200/internal/delivery/handlers/home"
	"2026_2_Na_Vse_200/internal/delivery/middleware"
	"2026_2_Na_Vse_200/internal/delivery/response"
	"2026_2_Na_Vse_200/internal/models"
	"github.com/rs/zerolog"
)

const (
	_coverLimit = 2 << 20
	_bodyLimit  = _coverLimit + (1 << 20)
)

// Repository persists playlists and resolves them for their owner.
type Repository interface {
	List(context.Context, models.UserID) ([]models.PlaylistPage, error)
	Get(context.Context, models.UserID, string) (models.PlaylistPage, error)
	Save(context.Context, models.UserID, string, models.PlaylistInput) (models.PlaylistPage, error)
	Import(context.Context, models.UserID, string, models.PlaylistInput, []string) (models.PlaylistImportResult, error)
	ChangeTracks(context.Context, models.UserID, string, models.PlaylistTrackChange) (models.PlaylistPage, error)
	Delete(context.Context, models.UserID, string) error
	Cover(context.Context, models.UserID, string, string) (models.MediaFile, error)
}

// Blobs opens playlist covers from storage.
type Blobs interface {
	Open(string) (*os.File, error)
}

// Handler serves the authenticated playlist API.
type Handler struct {
	repo  Repository
	blobs Blobs
}

// New creates a playlist handler.
func New(repo Repository, blobs Blobs) *Handler { return &Handler{repo: repo, blobs: blobs} }

type card struct {
	ID         models.PlaylistID `json:"id"`
	Name       string            `json:"name"`
	CoverURL   *string           `json:"cover_url"`
	TrackCount int               `json:"track_count"`
}

type detail struct {
	card

	Description string               `json:"description"`
	Tracks      []home.TrackResponse `json:"tracks"`
}

func newCard(p models.PlaylistPage) card {
	item := card{ID: p.ID, Name: p.Title, TrackCount: p.TrackCount}
	if p.CoverFileID != nil {
		url := "/api/v1/playlists/" + string(p.ID) + "/cover/" + string(*p.CoverFileID)
		item.CoverURL = &url
	}
	return item
}

func newDetail(p models.PlaylistPage) detail {
	return detail{
		card:        newCard(p),
		Description: p.Description,
		Tracks:      home.NewTrackResponses(p.Tracks),
	}
}

// Register mounts playlist routes behind the supplied authentication middleware.
func (h *Handler) Register(mux *http.ServeMux, auth middleware.Middleware) {
	register := func(pattern string, action func(http.ResponseWriter, *http.Request, models.UserID)) {
		mux.Handle(pattern, auth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			session, ok := middleware.SessionFromContext(r.Context())
			if !ok {
				response.WriteError(r.Context(), w, http.StatusUnauthorized, models.CodeUnauthorized, models.MsgUnauthorized)
				return
			}
			w.Header().Set("Cache-Control", "private, no-store")
			action(w, r, session.UserID)
		})))
	}
	register("GET /api/v1/playlists", h.list)
	register("GET /api/v1/playlists/{id}", h.get)
	register("POST /api/v1/playlists", h.save)
	register("PUT /api/v1/playlists/{id}", h.save)
	register("DELETE /api/v1/playlists/{id}", h.delete)
	register("POST /api/v1/playlists/import", h.importPlaylist)
	register("POST /api/v1/playlists/{id}/tracks", h.addTracks)
	register("DELETE /api/v1/playlists/{id}/tracks/{trackId}", h.removeTrack)
	register("GET /api/v1/playlists/{id}/cover/{fileId}", h.cover)
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request, owner models.UserID) {
	playlists, err := h.repo.List(r.Context(), owner)
	if err != nil {
		writeError(r.Context(), w, err)
		return
	}
	cards := make([]card, 0, len(playlists))
	for _, playlist := range playlists {
		cards = append(cards, newCard(playlist))
	}
	response.WriteJSON(r.Context(), w, http.StatusOK, struct {
		Playlists []card `json:"playlists"`
	}{Playlists: cards})
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request, owner models.UserID) {
	playlist, err := h.repo.Get(r.Context(), owner, r.PathValue("id"))
	if err != nil {
		writeError(r.Context(), w, err)
		return
	}
	response.WriteJSON(r.Context(), w, http.StatusOK, newDetail(playlist))
}

func (h *Handler) save(w http.ResponseWriter, r *http.Request, owner models.UserID) {
	form, err := parseForm(w, r)
	if err != nil {
		writeError(r.Context(), w, err)
		return
	}
	id := r.PathValue("id")
	playlist, err := h.repo.Save(r.Context(), owner, id, form.input)
	if err != nil {
		writeError(r.Context(), w, err)
		return
	}
	status := http.StatusOK
	if id == "" {
		status = http.StatusCreated
	}
	response.WriteJSON(r.Context(), w, status, newDetail(playlist))
}

func (h *Handler) importPlaylist(w http.ResponseWriter, r *http.Request, owner models.UserID) {
	form, err := parseForm(w, r)
	if err != nil {
		writeError(r.Context(), w, err)
		return
	}
	if !validLegacyID(form.legacyID) || form.trackIDs == nil {
		writeError(r.Context(), w, requestError{message: "Некорректные данные для импорта плейлиста."})
		return
	}
	if form.sourceAccountID != "" && form.sourceAccountID != string(owner) {
		response.WriteError(r.Context(), w, http.StatusForbidden, models.CodeForbidden, "Сессия аккаунта изменилась. Повторите вход.")
		return
	}
	result, err := h.repo.Import(r.Context(), owner, form.legacyID, form.input, form.trackIDs)
	if err != nil {
		writeError(r.Context(), w, err)
		return
	}
	status := http.StatusOK
	if result.Created {
		status = http.StatusCreated
	}
	response.WriteJSON(r.Context(), w, status, struct {
		Playlist detail   `json:"playlist"`
		Skipped  []string `json:"skipped_track_ids"`
	}{Playlist: newDetail(result.Playlist), Skipped: result.SkippedTrackIDs})
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request, owner models.UserID) {
	if err := h.repo.Delete(r.Context(), owner, r.PathValue("id")); err != nil {
		writeError(r.Context(), w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) addTracks(w http.ResponseWriter, r *http.Request, owner models.UserID) {
	var input struct {
		IDs []string `json:"track_ids"`
	}
	if err := decodeJSON(http.MaxBytesReader(w, r.Body, 1<<20), &input); err != nil {
		writeError(r.Context(), w, err)
		return
	}
	if input.IDs == nil {
		writeError(r.Context(), w, requestError{message: "Передайте список track_ids."})
		return
	}
	h.changeTracks(w, r, owner, models.PlaylistTrackChange{Add: input.IDs})
}

func (h *Handler) removeTrack(w http.ResponseWriter, r *http.Request, owner models.UserID) {
	id, err := uuid.Parse(r.PathValue("trackId"))
	if err != nil {
		writeError(r.Context(), w, models.ErrTrackNotFound)
		return
	}
	h.changeTracks(w, r, owner, models.PlaylistTrackChange{Remove: models.TrackID(id.String())})
}

func (h *Handler) changeTracks(
	w http.ResponseWriter,
	r *http.Request,
	owner models.UserID,
	change models.PlaylistTrackChange,
) {
	page, err := h.repo.ChangeTracks(r.Context(), owner, r.PathValue("id"), change)
	if err != nil {
		writeError(r.Context(), w, err)
		return
	}
	response.WriteJSON(r.Context(), w, http.StatusOK, newDetail(page))
}

func (h *Handler) cover(w http.ResponseWriter, r *http.Request, owner models.UserID) {
	file, err := h.repo.Cover(r.Context(), owner, r.PathValue("id"), r.PathValue("fileId"))
	if err != nil {
		writeError(r.Context(), w, err)
		return
	}
	body, err := h.blobs.Open(file.StorageKey)
	if err != nil {
		writeError(r.Context(), w, err)
		return
	}
	defer body.Close()
	info, err := body.Stat()
	if err != nil {
		writeError(r.Context(), w, err)
		return
	}
	w.Header().Set("Content-Type", file.MIMEType)
	w.Header().Set("X-Content-Type-Options", "nosniff")
	http.ServeContent(w, r, info.Name(), info.ModTime(), body)
}

func validLegacyID(id string) bool {
	return id != "" && id != "." && id != ".." && len(id) <= 128 && !strings.ContainsAny(id, "/\\\x00")
}

type playlistForm struct {
	sourceAccountID string
	input           models.PlaylistInput
	legacyID        string
	trackIDs        []string
}

func parseForm(w http.ResponseWriter, r *http.Request) (playlistForm, error) {
	r.Body = http.MaxBytesReader(w, r.Body, _bodyLimit)
	kind, _, _ := mime.ParseMediaType(r.Header.Get("Content-Type"))
	var (
		form playlistForm
		err  error
	)
	if kind == "multipart/form-data" {
		form, err = parseMultipartForm(r)
	} else {
		form, err = parseJSONForm(r.Body)
	}
	if err != nil {
		return form, err
	}
	form.input.Name = strings.TrimSpace(form.input.Name)
	form.input.Description = strings.TrimSpace(form.input.Description)
	if form.input.Name == "" || utf8.RuneCountInString(form.input.Name) > 100 {
		return form, requestError{message: "Введите название от 1 до 100 символов."}
	}
	if utf8.RuneCountInString(form.input.Description) > 500 {
		return form, requestError{message: "Описание должно быть не длиннее 500 символов."}
	}
	if strings.ContainsRune(form.input.Name, 0) || strings.ContainsRune(form.input.Description, 0) {
		return form, requestError{message: "Некорректные символы в полях плейлиста."}
	}
	return form, nil
}

func parseMultipartForm(r *http.Request) (playlistForm, error) {
	var form playlistForm
	if err := r.ParseMultipartForm(_bodyLimit); err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			return form, err
		}
		return form, requestError{message: "Не удалось прочитать поля плейлиста."}
	}
	defer r.MultipartForm.RemoveAll()

	form.input.Name = r.FormValue("name")
	form.input.Description = r.FormValue("description")
	form.legacyID = r.FormValue("legacy_id")
	form.sourceAccountID = r.FormValue("source_account_id")
	if raw := r.FormValue("track_ids"); raw != "" {
		if err := json.Unmarshal([]byte(raw), &form.trackIDs); err != nil {
			return form, requestError{message: "Некорректный список track_ids."}
		}
	}
	if len(r.MultipartForm.File["cover"]) == 0 {
		return form, nil
	}
	data, mimeType, err := readCover(r)
	if err != nil {
		return form, err
	}
	form.input.Cover = data
	form.input.CoverMIME = mimeType
	return form, nil
}

func parseJSONForm(body io.Reader) (playlistForm, error) {
	var input struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := decodeJSON(body, &input); err != nil {
		return playlistForm{}, err
	}
	return playlistForm{
		input: models.PlaylistInput{
			Name:        input.Name,
			Description: input.Description,
		},
	}, nil
}

func readCover(r *http.Request) ([]byte, string, error) {
	file, _, err := r.FormFile("cover")
	if err != nil {
		return nil, "", err
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, _coverLimit+1))
	if err != nil {
		return nil, "", err
	}
	if len(data) > _coverLimit {
		return nil, "", &http.MaxBytesError{Limit: _coverLimit}
	}
	_, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil || (format != "jpeg" && format != "png") {
		return nil, "", requestError{message: "Выберите файл JPG или PNG."}
	}
	return data, "image/" + format, nil
}

func decodeJSON(body io.Reader, target any) error {
	decoder := json.NewDecoder(body)
	decoder.DisallowUnknownFields()
	err := decoder.Decode(target)
	if err == nil {
		err = decoder.Decode(new(any))
		if errors.Is(err, io.EOF) {
			return nil
		}
	}
	var tooLarge *http.MaxBytesError
	if errors.As(err, &tooLarge) {
		return err
	}
	return requestError{message: models.MsgInvalidJSON}
}

type requestError struct{ message string }

func (e requestError) Error() string { return e.message }

func writeError(ctx context.Context, w http.ResponseWriter, err error) {
	var (
		invalid  requestError
		tooLarge *http.MaxBytesError
	)
	switch {
	case errors.Is(err, models.ErrPlaylistNotFound), errors.Is(err, models.ErrTrackNotFound),
		errors.Is(err, models.ErrMediaFileNotFound), errors.Is(err, os.ErrNotExist):
		response.WriteError(ctx, w, http.StatusNotFound, models.CodeNotFound, models.MsgNotFound)
	case errors.Is(err, models.ErrPlaylistImportDeleted):
		response.WriteError(ctx, w, http.StatusGone, "playlist_import_deleted", "Импортированный плейлист удалён.")
	case errors.As(err, &tooLarge):
		response.WriteError(ctx, w, http.StatusRequestEntityTooLarge, models.CodeValidationFailed,
			"Файл или запрос превышает допустимый размер.",
		)
	case errors.As(err, &invalid):
		response.WriteError(ctx, w, http.StatusBadRequest, models.CodeValidationFailed, invalid.message)
	default:
		zerolog.Ctx(ctx).Error().Err(err).Str("operation", "playlist.request").Msg("playlist request")
		response.WriteError(ctx, w, http.StatusInternalServerError, models.CodeInternal, models.MsgInternal)
	}
}
