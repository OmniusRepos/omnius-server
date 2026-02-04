package providers

import "torrent-server/models"

// TorrentProvider is the interface for torrent search providers
type TorrentProvider interface {
	Name() string
	SearchMovie(title string, year int) ([]TorrentResult, error)
	SearchSeries(title string, season, episode int) ([]TorrentResult, error)
}

// TorrentResult represents a torrent search result
type TorrentResult struct {
	Title     string
	Hash      string
	MagnetURL string
	Quality   string
	Type      string // web, bluray, etc.
	Seeds     uint
	Peers     uint
	Size      string
	SizeBytes uint64
	Source    string // Provider name
}

// ToMovieTorrent converts TorrentResult to models.Torrent
func (r TorrentResult) ToMovieTorrent(movieID uint) *models.Torrent {
	return &models.Torrent{
		MovieID:   movieID,
		URL:       r.MagnetURL,
		Hash:      r.Hash,
		Quality:   r.Quality,
		Type:      r.Type,
		Seeds:     r.Seeds,
		Peers:     r.Peers,
		Size:      r.Size,
		SizeBytes: r.SizeBytes,
	}
}

// ToEpisodeTorrent converts TorrentResult to models.EpisodeTorrent
func (r TorrentResult) ToEpisodeTorrent(episodeID uint) *models.EpisodeTorrent {
	return &models.EpisodeTorrent{
		EpisodeID:    episodeID,
		Hash:         r.Hash,
		Quality:      r.Quality,
		Seeds:        r.Seeds,
		Peers:        r.Peers,
		Size:         r.Size,
		SizeBytes:    r.SizeBytes,
		ReleaseGroup: r.Source, // Use Source as release group
	}
}

// ToSeasonPack converts TorrentResult to models.SeasonPack
func (r TorrentResult) ToSeasonPack(seriesID uint, season uint) *models.SeasonPack {
	return &models.SeasonPack{
		SeriesID:  seriesID,
		Season:    season,
		Hash:      r.Hash,
		Quality:   r.Quality,
		Seeds:     r.Seeds,
		Peers:     r.Peers,
		Size:      r.Size,
		SizeBytes: r.SizeBytes,
		Source:    r.Source,
	}
}
