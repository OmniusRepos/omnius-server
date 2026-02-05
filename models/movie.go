package models

import "encoding/json"

type Movie struct {
	ID                      uint      `json:"id"`
	URL                     string    `json:"url,omitempty"`
	ImdbCode                string    `json:"imdb_code"`
	Title                   string    `json:"title"`
	TitleEnglish            string    `json:"title_english,omitempty"`
	TitleLong               string    `json:"title_long"`
	Slug                    string    `json:"slug"`
	Year                    uint      `json:"year"`
	Rating                  float32   `json:"rating"`
	Runtime                 uint      `json:"runtime"`
	Genres                  []string  `json:"genres"`
	Summary                 string    `json:"summary"`
	DescriptionFull         string    `json:"description_full"`
	Synopsis                string    `json:"synopsis"`
	YtTrailerCode           string    `json:"yt_trailer_code"`
	Language                string    `json:"language"`
	MpaRating               string    `json:"mpa_rating,omitempty"`
	BackgroundImage         string    `json:"background_image"`
	BackgroundImageOriginal string    `json:"background_image_original,omitempty"`
	BackgroundImages        []string  `json:"background_images,omitempty"` // Multiple images for Ken Burns effect
	SmallCoverImage         string    `json:"small_cover_image"`
	MediumCoverImage        string    `json:"medium_cover_image"`
	LargeCoverImage         string    `json:"large_cover_image"`
	Torrents                []Torrent `json:"torrents"`
	Cast                    []Cast    `json:"cast,omitempty"`
	LikeCount               uint      `json:"like_count,omitempty"`
	DownloadCount           uint      `json:"download_count,omitempty"`
	DateUploaded            string    `json:"date_uploaded"`
	DateUploadedUnix        int64     `json:"date_uploaded_unix"`
	// Content metadata
	ContentType     string   `json:"content_type,omitempty"` // movie, etc.
	Provider        string   `json:"provider,omitempty"`     // yts, etc.
	CreatedAt       string   `json:"created_at,omitempty"`
	// Ratings from external sources
	ImdbRating       *float32 `json:"imdb_rating,omitempty"`
	ImdbVotes        string   `json:"imdb_votes,omitempty"`
	RottenTomatoes   *int     `json:"rotten_tomatoes,omitempty"`
	Metacritic       *int     `json:"metacritic,omitempty"`
	RatingsUpdatedAt string   `json:"ratings_updated_at,omitempty"`
	// Franchise/collection info
	Franchise       string   `json:"franchise,omitempty"`
	State           string   `json:"state,omitempty"`
	// Coming soon status
	Status          string   `json:"status,omitempty"`       // "available" or "coming_soon"
	ReleaseDate     string   `json:"release_date,omitempty"` // YYYY-MM-DD format
	// Rich data from IMDB
	Director        string   `json:"director,omitempty"`
	Writers         []string `json:"writers,omitempty"`
	Budget          string   `json:"budget,omitempty"`
	BoxOfficeGross  string   `json:"box_office_gross,omitempty"`
	Country         string   `json:"country,omitempty"`
	Awards          string   `json:"awards,omitempty"`
	AllImages       []string `json:"all_images,omitempty"`
}

type Cast struct {
	Name          string `json:"name"`
	CharacterName string `json:"character_name"`
	URLSmallImage string `json:"url_small_image,omitempty"`
	ImdbCode      string `json:"imdb_code"`
}

// LocalRating represents cached ratings for a movie
type LocalRating struct {
	ImdbRating     *float32 `json:"imdb_rating,omitempty"`
	RottenTomatoes *int     `json:"rotten_tomatoes,omitempty"`
	Metacritic     *int     `json:"metacritic,omitempty"`
}

// GenresJSON returns genres as JSON string for database storage
func (m *Movie) GenresJSON() string {
	data, _ := json.Marshal(m.Genres)
	return string(data)
}

// ParseGenres parses JSON genres string from database
func (m *Movie) ParseGenres(genresJSON string) {
	if genresJSON != "" {
		json.Unmarshal([]byte(genresJSON), &m.Genres)
	}
}

// WritersJSON returns writers as JSON string for database storage
func (m *Movie) WritersJSON() string {
	data, _ := json.Marshal(m.Writers)
	return string(data)
}

// ParseWriters parses JSON writers string from database
func (m *Movie) ParseWriters(writersJSON string) {
	if writersJSON != "" {
		json.Unmarshal([]byte(writersJSON), &m.Writers)
	}
}

// CastJSON returns cast as JSON string for database storage
func (m *Movie) CastJSON() string {
	data, _ := json.Marshal(m.Cast)
	return string(data)
}

// ParseCast parses JSON cast string from database
func (m *Movie) ParseCast(castJSON string) {
	if castJSON != "" {
		json.Unmarshal([]byte(castJSON), &m.Cast)
	}
}

// AllImagesJSON returns all images as JSON string for database storage
func (m *Movie) AllImagesJSON() string {
	data, _ := json.Marshal(m.AllImages)
	return string(data)
}

// ParseAllImages parses JSON all images string from database
func (m *Movie) ParseAllImages(imagesJSON string) {
	if imagesJSON != "" {
		json.Unmarshal([]byte(imagesJSON), &m.AllImages)
	}
}

type MovieListData struct {
	MovieCount int     `json:"movie_count"`
	Limit      int     `json:"limit"`
	PageNumber int     `json:"page_number"`
	Movies     []Movie `json:"movies"`
}

type MovieDetailsData struct {
	Movie Movie `json:"movie"`
}

type MovieSuggestionsData struct {
	MovieCount int     `json:"movie_count"`
	Movies     []Movie `json:"movies"`
}
