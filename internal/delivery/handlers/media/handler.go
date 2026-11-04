package media

import (
	"errors"
	"io"
	"log"
	"net/http"
	"os"
	"uuid"

	"2026_2_Na_Vse_200/internal/delivery/response"
	"2026_2_Na_Vse_200/internal/models"
	"2026_2_Na_Vse_200/internal/usecase/ports"
)

type MediaBlobStore interface {
	Open(key string) (*os.File, error)
}

type MediaHandler struct {
	media ports.MediaRepository
	blobs MediaBlobStore
}

func NewMediaHandler(media ports.MediaRepository, blobs MediaBlobStore) *MediaHandler {
	return &MediaHandler{media: media, blobs: blobs}
}

func (h *MediaHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		response.WriteError(w, http.StatusNotFound, models.CodeNotFound, models.MsgNotFound)
		return
	}

	file, err := h.media.GetByID(r.Context(), models.MediaFileID(id.String()))
	if errors.Is(err, models.ErrMediaFileNotFound) {
		response.WriteError(w, http.StatusNotFound, models.CodeNotFound, models.MsgNotFound)
		return
	}
	if err != nil {
		log.Printf("чтение медиафайла: %v", err)
		response.WriteError(w, http.StatusInternalServerError, models.CodeInternal, models.MsgInternal)
		return
	}

	blob, err := h.blobs.Open(file.StorageKey)
	if err != nil {
		if os.IsNotExist(err) {
			response.WriteError(w, http.StatusNotFound, models.CodeNotFound, models.MsgNotFound)
			return
		}
		log.Printf("открытие медиафайла: %v", err)
		response.WriteError(w, http.StatusInternalServerError, models.CodeInternal, models.MsgInternal)
		return
	}
	defer blob.Close()

	w.Header().Set("Content-Type", file.MIMEType)
	w.Header().Set("Cache-Control", "public, max-age=86400")
	w.WriteHeader(http.StatusOK)
	if _, err := io.Copy(w, blob); err != nil {
		log.Printf("отправка медиафайла: %v", err)
	}
}
