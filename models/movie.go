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
	SmallCoverImage         string    `json:"small_cover_image"`
	MediumCoverImage        string    `json:"medium_cover_image"`
	LargeCoverImage         string    `json:"large_cover_image"`
	Torrents                []Torrent `json:"torrents"`
	Cast                    []Cast    `json:"cast,omitempty"`
	LikeCount               uint      `json:"like_count,omitempty"`
	DownloadCount           uint      `json:"download_count,omitempty"`
	DateUploaded            string    `json:"date_uploaded"`
	DateUploadedUnix        int64     `json:"date_uploaded_unix"`
	// Ratings from external sources
	ImdbRating      *float32 `json:"imdb_rating,omitempty"`
	RottenTomatoes  *int     `json:"rotten_tomatoes,omitempty"`
	Metacritic      *int     `json:"metacritic,omitempty"`
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
