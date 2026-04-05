package models

import "time"

type Movie struct {
	ID                      uint        `json:"id" gorm:"primaryKey"`
	URL                     string      `json:"url,omitempty" gorm:"column:url"`
	ImdbCode                string      `json:"imdb_code" gorm:"uniqueIndex;not null"`
	Title                   string      `json:"title" gorm:"not null"`
	TitleEnglish            string      `json:"title_english,omitempty"`
	TitleLong               string      `json:"title_long"`
	Slug                    string      `json:"slug"`
	Year                    uint        `json:"year"`
	Rating                  float32     `json:"rating" gorm:"default:0"`
	Runtime                 uint        `json:"runtime" gorm:"default:0"`
	Genres                  StringSlice `json:"genres" gorm:"column:genres;type:text"`
	Summary                 string      `json:"summary"`
	DescriptionFull         string      `json:"description_full"`
	Synopsis                string      `json:"synopsis"`
	YtTrailerCode           string      `json:"yt_trailer_code" gorm:"column:yt_trailer_code"`
	Language                string      `json:"language" gorm:"default:'en'"`
	MpaRating               string      `json:"mpa_rating,omitempty"`
	BackgroundImage         string      `json:"background_image"`
	BackgroundImageOriginal string      `json:"background_image_original,omitempty"`
	SmallCoverImage         string      `json:"small_cover_image"`
	MediumCoverImage        string      `json:"medium_cover_image"`
	LargeCoverImage         string      `json:"large_cover_image"`
	DateUploaded            string      `json:"date_uploaded"`
	DateUploadedUnix        int64       `json:"date_uploaded_unix"`
	CreatedAt               time.Time   `json:"created_at,omitempty" gorm:"autoCreateTime"`

	// Relationships
	Torrents []Torrent `json:"torrents" gorm:"foreignKey:MovieID;constraint:OnDelete:CASCADE"`

	// Rich data (JSON-serialized in DB)
	Cast       CastSlice   `json:"cast,omitempty" gorm:"column:cast_json;type:text"`
	Writers    StringSlice `json:"writers,omitempty" gorm:"column:writers;type:text"`
	AllImages  StringSlice `json:"all_images,omitempty" gorm:"column:all_images;type:text"`

	// Not stored in DB
	BackgroundImages []string `json:"background_images,omitempty" gorm:"-"`

	// Counters
	LikeCount     uint `json:"like_count,omitempty" gorm:"default:0"`
	DownloadCount uint `json:"download_count,omitempty" gorm:"default:0"`

	// Content metadata
	ContentType string `json:"content_type,omitempty" gorm:"default:'movie'"`
	Provider    string `json:"provider,omitempty"`

	// External ratings
	ImdbRating       *float32 `json:"imdb_rating,omitempty"`
	ImdbVotes        string   `json:"imdb_votes,omitempty"`
	RottenTomatoes   *int     `json:"rotten_tomatoes,omitempty"`
	Metacritic       *int     `json:"metacritic,omitempty"`
	RatingsUpdatedAt string   `json:"ratings_updated_at,omitempty"`

	// Franchise/collection
	Franchise string `json:"franchise,omitempty"`
	State     string `json:"state,omitempty" gorm:"default:'ok'"`

	// Coming soon
	Status      string `json:"status,omitempty" gorm:"default:'available'"`
	ReleaseDate string `json:"release_date,omitempty"`

	// Rich data from IMDB
	Director       string `json:"director,omitempty"`
	Budget         string `json:"budget,omitempty"`
	BoxOfficeGross string `json:"box_office_gross,omitempty"`
	Country        string `json:"country,omitempty"`
	Awards         string `json:"awards,omitempty"`
}

type Cast struct {
	Name          string `json:"name"`
	CharacterName string `json:"character_name"`
	URLSmallImage string `json:"url_small_image,omitempty"`
	ImdbCode      string `json:"imdb_code"`
}

type LocalRating struct {
	ImdbRating     *float32 `json:"imdb_rating,omitempty"`
	RottenTomatoes *int     `json:"rotten_tomatoes,omitempty"`
	Metacritic     *int     `json:"metacritic,omitempty"`
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
