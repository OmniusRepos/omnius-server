package models

type Series struct {
	ID               uint      `json:"id"`
	ImdbCode         string    `json:"imdb_code"`
	Title            string    `json:"title"`
	TitleSlug        string    `json:"title_slug"`
	Year             uint      `json:"year"`
	Rating           float32   `json:"rating"`
	Genres           []string  `json:"genres"`
	Summary          string    `json:"summary"`
	PosterImage      string    `json:"poster_image"`
	BackgroundImage  string    `json:"background_image"`
	TotalSeasons     uint      `json:"total_seasons"`
	Status           string    `json:"status"` // ongoing, ended
	DateAdded        string    `json:"date_added"`
	DateAddedUnix    int64     `json:"date_added_unix"`
}

type Episode struct {
	ID            uint            `json:"id"`
	SeriesID      uint            `json:"series_id"`
	Season        uint            `json:"season"`
	Episode       uint            `json:"episode"`
	Title         string          `json:"title"`
	Overview      string          `json:"overview"`
	AirDate       string          `json:"air_date"`
	ImdbCode      string          `json:"imdb_code"`
	Torrents      []EpisodeTorrent `json:"torrents"`
}

type EpisodeTorrent struct {
	ID           uint   `json:"-"`
	EpisodeID    uint   `json:"-"`
	Hash         string `json:"hash"`
	Quality      string `json:"quality"`
	Seeds        uint   `json:"seeds"`
	Peers        uint   `json:"peers"`
	Size         string `json:"size"`
	SizeBytes    uint64 `json:"size_bytes"`
	Source       string `json:"source"` // eztv, 1337x, etc.
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
