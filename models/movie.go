package models

import "encoding/json"

type Movie struct {
	ID               uint      `json:"id"`
	ImdbCode         string    `json:"imdb_code"`
	Title            string    `json:"title"`
	TitleEnglish     string    `json:"title_english,omitempty"`
	TitleLong        string    `json:"title_long"`
	Slug             string    `json:"slug"`
	Year             uint      `json:"year"`
	Rating           float32   `json:"rating"`
	Runtime          uint      `json:"runtime"`
	Genres           []string  `json:"genres"`
	Summary          string    `json:"summary"`
	DescriptionFull  string    `json:"description_full"`
	Synopsis         string    `json:"synopsis"`
	YtTrailerCode    string    `json:"yt_trailer_code"`
	Language         string    `json:"language"`
	BackgroundImage  string    `json:"background_image"`
	SmallCoverImage  string    `json:"small_cover_image"`
	MediumCoverImage string    `json:"medium_cover_image"`
	LargeCoverImage  string    `json:"large_cover_image"`
	Torrents         []Torrent `json:"torrents"`
	DateUploaded     string    `json:"date_uploaded"`
	DateUploadedUnix int64     `json:"date_uploaded_unix"`
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
