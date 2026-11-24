package media

import (
	"2026_2_Na_Vse_200/internal/delivery/response"
	"2026_2_Na_Vse_200/internal/models"
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"uuid"
)

type Repository interface {
	GetByID(context.Context, models.MediaFileID) (models.MediaFile, error)
}
type Tracks interface {
	GetAudio(context.Context, models.TrackID) (models.MediaFile, error)
}
type Blobs interface {
	Open(string) (*os.File, error)
}
type MediaHandler struct {
	media  Repository
	tracks Tracks
	blobs  Blobs
}

func NewMediaHandler(media Repository, blobs Blobs, tracks ...Tracks) *MediaHandler {
	h := &MediaHandler{media: media, blobs: blobs}
	if len(tracks) > 0 {
		h.tracks = tracks[0]
	}
	return h
}
func (h *MediaHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		response.WriteError(w, http.StatusNotFound, models.CodeNotFound, models.MsgNotFound)
		return
	}
	file, err := h.media.GetByID(r.Context(), models.MediaFileID(id.String()))
	h.serve(w, r, file, err)
}
func (h *MediaHandler) Audio(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil || h.tracks == nil {
		response.WriteError(w, http.StatusNotFound, models.CodeNotFound, models.MsgNotFound)
		return
	}
	file, err := h.tracks.GetAudio(r.Context(), models.TrackID(id.String()))
	h.serve(w, r, file, err)
}
func (h *MediaHandler) serve(w http.ResponseWriter, r *http.Request, file models.MediaFile, err error) {
	if errors.Is(err, models.ErrMediaFileNotFound) || errors.Is(err, models.ErrTrackNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		log.Printf("media metadata: %v", err)
		response.WriteError(w, 500, models.CodeInternal, models.MsgInternal)
		return
	}
	body, err := h.blobs.Open(file.StorageKey)
	if os.IsNotExist(err) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		log.Printf("open media: %v", err)
		response.WriteError(w, 500, models.CodeInternal, models.MsgInternal)
		return
	}
	defer body.Close()
	info, err := body.Stat()
	if err != nil || !info.Mode().IsRegular() {
		response.WriteError(w, 500, models.CodeInternal, models.MsgInternal)
		return
	}
	w.Header().Set("Content-Type", file.MIMEType)
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	http.ServeContent(w, r, info.Name(), info.ModTime(), body)
}
