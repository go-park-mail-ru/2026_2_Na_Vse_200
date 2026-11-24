package home

type homeResponse struct {
	Tracks  []TrackResponse  `json:"tracks"`
	Artists []ArtistResponse `json:"artists"`
	Albums  []AlbumResponse  `json:"albums"`
}

// TrackResponse represents a catalog track in the API.
type TrackResponse struct {
	ID         string              `json:"id"`
	Title      string              `json:"title"`
	DurationMS int                 `json:"duration_ms"`
	CoverURL   *string             `json:"cover_url"`
	Artists    []artistRefResponse `json:"artists"`
}

// ArtistResponse represents a catalog artist in the API.
type ArtistResponse struct {
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	ImageURL *string `json:"image_url"`
}

// AlbumResponse represents a catalog album in the API.
type AlbumResponse struct {
	ID          string              `json:"id"`
	Title       string              `json:"title"`
	ReleaseDate *string             `json:"release_date"`
	CoverURL    *string             `json:"cover_url"`
	Artists     []artistRefResponse `json:"artists"`
}

type artistRefResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}
