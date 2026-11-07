package artist

import (
	"context"
	"fmt"
	"log/slog"

	"2026_2_Na_Vse_200/internal/logger"
	"2026_2_Na_Vse_200/internal/models"
	"2026_2_Na_Vse_200/internal/usecase/ports"
)

const pageLimit = 50

type ArtistPage struct {
	Artist models.Artist
	Tracks []models.Track
	Albums []models.Album
}

type ArtistUsecase struct {
	artists ports.ArtistRepository
	log     *slog.Logger
}

func NewArtistUsecase(artists ports.ArtistRepository, log *slog.Logger) *ArtistUsecase {
	if log == nil {
		log = logger.Discard()
	}
	return &ArtistUsecase{artists: artists, log: log}
}

func (u *ArtistUsecase) Page(ctx context.Context, id models.ArtistID) (ArtistPage, error) {
	return u.PageAt(ctx, id, pageLimit, 0)
}

func (u *ArtistUsecase) PageAt(ctx context.Context, id models.ArtistID, limit, offset int) (ArtistPage, error) {
	if limit < 1 || limit > 100 || offset < 0 {
		return ArtistPage{}, fmt.Errorf("invalid pagination")
	}
	log := logger.FromContext(ctx, u.log)
	log.Info("artist.Page: старт", slog.String("artist_id", string(id)))

	artist, err := u.artists.GetByID(ctx, id)
	if err != nil {
		log.Error("artist.Page: чтение исполнителя", slog.Any("error", err), slog.String("artist_id", string(id)))
		return ArtistPage{}, fmt.Errorf("read artist: %w", err)
	}
	log.Info("artist.Page: исполнитель получен", slog.String("artist_id", string(artist.ID)))

	tracks, err := u.artists.Tracks(ctx, id, limit, offset)
	if err != nil {
		log.Error("artist.Page: чтение треков", slog.Any("error", err), slog.String("artist_id", string(id)))
		return ArtistPage{}, fmt.Errorf("read artist tracks: %w", err)
	}
	log.Info("artist.Page: треки получены", slog.String("artist_id", string(id)), slog.Int("count", len(tracks)))

	albums, err := u.artists.Albums(ctx, id, limit, offset)
	if err != nil {
		log.Error("artist.Page: чтение альбомов", slog.Any("error", err), slog.String("artist_id", string(id)))
		return ArtistPage{}, fmt.Errorf("read artist albums: %w", err)
	}
	log.Info("artist.Page: альбомы получены", slog.String("artist_id", string(id)), slog.Int("count", len(albums)))

	return ArtistPage{Artist: artist, Tracks: tracks, Albums: albums}, nil
}
