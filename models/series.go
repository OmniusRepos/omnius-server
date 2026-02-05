package models

import "encoding/json"

type Series struct {
	ID              uint      `json:"id"`
	ImdbCode        string    `json:"imdb_code"`
	TvdbID          *uint     `json:"tvdb_id,omitempty"`
	Title           string    `json:"title"`
	TitleSlug       string    `json:"title_slug"`
	Year            uint      `json:"year"`
	EndYear         *uint     `json:"end_year,omitempty"`
	Rating          float32   `json:"rating"`
	Runtime         uint      `json:"runtime"`
	Genres          []string  `json:"genres"`
	Summary         string    `json:"summary"`
	Status          string    `json:"status"` // Continuing, Ended
	Network         string    `json:"network,omitempty"`
	PosterImage     string    `json:"poster_image"`
	BackgroundImage string    `json:"background_image"`
	TotalSeasons    uint      `json:"total_seasons"`
	TotalEpisodes   uint      `json:"total_episodes"`
	Seasons         []Season  `json:"seasons,omitempty"`
	DateAdded       string    `json:"date_added"`
	DateAddedUnix   int64     `json:"date_added_unix"`
	ImdbRating      *float32  `json:"imdb_rating,omitempty"`
	RottenTomatoes  *int      `json:"rotten_tomatoes,omitempty"`
	Franchise       string    `json:"franchise,omitempty"`
}

// GenresJSON returns genres as JSON string for database storage
func (s *Series) GenresJSON() string {
	data, _ := json.Marshal(s.Genres)
	return string(data)
}

// ParseGenres parses JSON genres string from database
func (s *Series) ParseGenres(genresJSON string) {
	if genresJSON != "" {
		json.Unmarshal([]byte(genresJSON), &s.Genres)
	}
}

type Season struct {
	ID           uint      `json:"id"`
	SeriesID     uint      `json:"series_id"`
	SeasonNumber uint      `json:"season_number"`
	EpisodeCount uint      `json:"episode_count"`
	AirDate      string    `json:"air_date,omitempty"`
	PosterImage  string    `json:"poster_image,omitempty"`
	Episodes     []Episode `json:"episodes,omitempty"`
}

type Episode struct {
	ID            uint             `json:"id"`
	SeriesID      uint             `json:"series_id"`
	SeasonNumber  uint             `json:"season_number"`
	EpisodeNumber uint             `json:"episode_number"`
	Title         string           `json:"title"`
	Summary       string           `json:"summary,omitempty"`
	AirDate       string           `json:"air_date,omitempty"`
	Runtime       *uint            `json:"runtime,omitempty"`
	StillImage    string           `json:"still_image,omitempty"`
	Torrents      []EpisodeTorrent `json:"torrents,omitempty"`
}

type EpisodeTorrent struct {
	ID             uint   `json:"id"`
	EpisodeID      uint   `json:"episode_id"`
	SeriesID       uint   `json:"series_id"`
	SeasonNumber   uint   `json:"season_number"`
	EpisodeNumber  uint   `json:"episode_number"`
	Hash           string `json:"hash"`
	Quality        string `json:"quality"`
	VideoCodec     string `json:"video_codec,omitempty"`
	Seeds          uint   `json:"seeds"`
	Peers          uint   `json:"peers"`
	Size           string `json:"size"`
	SizeBytes      uint64 `json:"size_bytes"`
	ReleaseGroup   string `json:"release_group,omitempty"`
	DateUploaded   string `json:"date_uploaded"`
	DateUploadedUnix int64 `json:"date_uploaded_unix"`
}

type SeasonPack struct {
	ID        uint   `json:"-"`
	SeriesID  uint   `json:"series_id"`
	Season    uint   `json:"season"`
	Hash      string `json:"hash"`
	Quality   string `json:"quality"`
	Seeds     uint   `json:"seeds"`
	Peers     uint   `json:"peers"`
	Size      string `json:"size"`
	SizeBytes uint64 `json:"size_bytes"`
	Source    string `json:"source"`
}

type SeriesListData struct {
	SeriesCount int      `json:"series_count"`
	Limit       int      `json:"limit"`
	PageNumber  int      `json:"page_number"`
	Series      []Series `json:"series"`
}

type SeriesDetailsData struct {
	Series Series `json:"series"`
}

// TorrentStats represents real-time torrent statistics
type TorrentStats struct {
	Hash  string `json:"hash"`
	Seeds uint   `json:"seeds"`
	Peers uint   `json:"peers"`
	Name  string `json:"name,omitempty"`
	Found bool   `json:"found"`
}
