package handlers

type homeResponse struct {
	Tracks  []trackResponse  `json:"tracks"`
	Artists []artistResponse `json:"artists"`
	Albums  []albumResponse  `json:"albums"`
}

const _sessionCookieName = "session_id"

type healthResponse struct {
	Status string
}
type userResponse struct {
	ID          string  `json:"id"`
	Email       string  `json:"email"`
	DisplayName string  `json:"display_name"`
	AvatarURL   *string `json:"avatar_url"`
}

type trackResponse struct {
	ID         string              `json:"id"`
	Title      string              `json:"title"`
	DurationMS int                 `json:"duration_ms"`
	CoverURL   *string             `json:"cover_url"`
	Artists    []artistRefResponse `json:"artists"`
}

type artistResponse struct {
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	ImageURL *string `json:"image_url"`
}

type albumResponse struct {
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
