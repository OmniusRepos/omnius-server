package models

type Series struct {
	ID              uint        `json:"id" gorm:"primaryKey"`
	ImdbCode        string      `json:"imdb_code" gorm:"uniqueIndex"`
	TvdbID          *uint       `json:"tvdb_id,omitempty"`
	Title           string      `json:"title" gorm:"not null"`
	TitleSlug       string      `json:"title_slug"`
	Year            uint        `json:"year"`
	EndYear         *uint       `json:"end_year,omitempty"`
	Rating          float32     `json:"rating" gorm:"default:0"`
	Runtime         uint        `json:"runtime" gorm:"default:0"`
	Genres          StringSlice `json:"genres" gorm:"column:genres;type:text"`
	Summary         string      `json:"summary"`
	Status          string      `json:"status" gorm:"default:'ongoing'"`
	Network         string      `json:"network,omitempty"`
	PosterImage     string      `json:"poster_image"`
	BackgroundImage string      `json:"background_image"`
	TotalSeasons    uint        `json:"total_seasons" gorm:"default:0"`
	TotalEpisodes   uint        `json:"total_episodes" gorm:"default:0"`
	DateAdded       string      `json:"date_added"`
	DateAddedUnix   int64       `json:"date_added_unix"`
	ImdbRating      *float32    `json:"imdb_rating,omitempty"`
	RottenTomatoes  *int        `json:"rotten_tomatoes,omitempty"`
	Franchise       string      `json:"franchise,omitempty"`

	// Relationships
	Seasons  []Season  `json:"seasons,omitempty" gorm:"foreignKey:SeriesID;constraint:OnDelete:CASCADE"`
	Episodes []Episode `json:"-" gorm:"foreignKey:SeriesID;constraint:OnDelete:CASCADE"`
}

type Season struct {
	ID           uint   `json:"id" gorm:"primaryKey"`
	SeriesID     uint   `json:"series_id" gorm:"index;not null"`
	SeasonNumber uint   `json:"season_number"`
	EpisodeCount uint   `json:"episode_count" gorm:"default:0"`
	AirDate      string `json:"air_date,omitempty"`
	PosterImage  string `json:"poster_image,omitempty"`

	// Relationships
	Episodes []Episode `json:"episodes,omitempty" gorm:"foreignKey:SeriesID,SeasonNumber;references:SeriesID,SeasonNumber"`
}

func (Season) TableName() string { return "seasons" }

type Episode struct {
	ID            uint   `json:"id" gorm:"primaryKey"`
	SeriesID      uint   `json:"series_id" gorm:"index;not null"`
	SeasonNumber  uint   `json:"season_number" gorm:"column:season_number"`
	EpisodeNumber uint   `json:"episode_number" gorm:"column:episode_number"`
	Title         string `json:"title"`
	Summary       string `json:"summary,omitempty"`
	AirDate       string `json:"air_date,omitempty"`
	Runtime       *uint  `json:"runtime,omitempty"`
	StillImage    string `json:"still_image,omitempty"`
	ImdbCode      string `json:"imdb_code,omitempty"`

	// Relationships
	Torrents []EpisodeTorrent `json:"torrents,omitempty" gorm:"foreignKey:EpisodeID;constraint:OnDelete:CASCADE"`
}

func (Episode) TableName() string { return "episodes" }

type EpisodeTorrent struct {
	ID               uint   `json:"id" gorm:"primaryKey"`
	EpisodeID        uint   `json:"episode_id" gorm:"index;not null"`
	SeriesID         uint   `json:"series_id"`
	SeasonNumber     uint   `json:"season_number"`
	EpisodeNumber    uint   `json:"episode_number"`
	Hash             string `json:"hash" gorm:"not null"`
	Quality          string `json:"quality"`
	VideoCodec       string `json:"video_codec,omitempty"`
	Seeds            uint   `json:"seeds" gorm:"default:0"`
	Peers            uint   `json:"peers" gorm:"default:0"`
	Size             string `json:"size"`
	SizeBytes        uint64 `json:"size_bytes"`
	FileIndex        int    `json:"file_index" gorm:"default:-1"`
	ReleaseGroup     string `json:"release_group,omitempty"`
	DateUploaded     string `json:"date_uploaded"`
	DateUploadedUnix int64  `json:"date_uploaded_unix"`
}

func (EpisodeTorrent) TableName() string { return "episode_torrents" }

type SeasonPack struct {
	ID        uint   `json:"id" gorm:"primaryKey"`
	SeriesID  uint   `json:"series_id" gorm:"index;not null"`
	Season    uint   `json:"season" gorm:"column:season_number"`
	Hash      string `json:"hash" gorm:"not null"`
	Quality   string `json:"quality"`
	Seeds     uint   `json:"seeds" gorm:"default:0"`
	Peers     uint   `json:"peers" gorm:"default:0"`
	Size      string `json:"size"`
	SizeBytes uint64 `json:"size_bytes"`
	Source    string `json:"source"`
}

func (SeasonPack) TableName() string { return "season_packs" }

// API response wrappers

type SeriesListData struct {
	SeriesCount int      `json:"series_count"`
	Limit       int      `json:"limit"`
	PageNumber  int      `json:"page_number"`
	Series      []Series `json:"series"`
}

type SeriesDetailsData struct {
	Series Series `json:"series"`
}
