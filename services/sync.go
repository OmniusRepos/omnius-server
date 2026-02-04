package services

import (
	"log"
	"sync"
	"time"

	"torrent-server/database"
	"torrent-server/models"
	"torrent-server/providers"
)

type SyncService struct {
	db        *database.DB
	providers []providers.TorrentProvider
	omdb      *OMDBService
	mu        sync.Mutex
	running   bool
}

func NewSyncService(db *database.DB, omdb *OMDBService) *SyncService {
	return &SyncService{
		db:   db,
		omdb: omdb,
		providers: []providers.TorrentProvider{
			providers.NewYTSProvider(),
			providers.NewEZTVProvider(),
			providers.NewL337xProvider(),
		},
	}
}

// SyncMovie fetches metadata and torrents for a movie
func (s *SyncService) SyncMovie(imdbCode string) (*models.Movie, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if movie already exists
	existing, err := s.db.GetMovieByIMDB(imdbCode)
	if err == nil && existing != nil {
		// Update torrents
		s.syncMovieTorrents(existing)
		return existing, nil
	}

	// Fetch from OMDB
	movie, err := s.omdb.FetchByIMDB(imdbCode)
	if err != nil {
		return nil, err
	}

	// Save movie
	if err := s.db.CreateMovie(movie); err != nil {
		return nil, err
	}

	// Fetch torrents
	s.syncMovieTorrents(movie)

	return movie, nil
}

func (s *SyncService) syncMovieTorrents(movie *models.Movie) {
	for _, provider := range s.providers {
		results, err := provider.SearchMovie(movie.Title, int(movie.Year))
		if err != nil {
			log.Printf("Failed to search %s for %s: %v", provider.Name(), movie.Title, err)
			continue
		}

		for _, result := range results {
			// Check if torrent already exists
			existing, _ := s.db.GetTorrentByHash(result.Hash)
			if existing != nil {
				continue
			}

			torrent := result.ToMovieTorrent(movie.ID)
			torrent.DateUploaded = time.Now().Format("2006-01-02 15:04:05")
			torrent.DateUploadedUnix = time.Now().Unix()

			if err := s.db.CreateTorrent(torrent); err != nil {
				log.Printf("Failed to save torrent: %v", err)
			}
		}
	}
}

// SyncSeries fetches metadata and torrents for a series
func (s *SyncService) SyncSeries(imdbCode string) (*models.Series, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if series already exists
	existing, err := s.db.GetSeriesByIMDB(imdbCode)
	if err == nil && existing != nil {
		return existing, nil
	}

	// For now, just create a basic series entry
	// In production, you'd fetch from TMDB or similar
	series := &models.Series{
		ImdbCode: imdbCode,
		Status:   "ongoing",
	}

	if err := s.db.CreateSeries(series); err != nil {
		return nil, err
	}

	return series, nil
}

// SyncEpisode fetches torrents for a specific episode
func (s *SyncService) SyncEpisode(series *models.Series, season, episode int) error {
	// Create episode if not exists
	episodes, _ := s.db.GetEpisodes(series.ID, season)
	var ep *models.Episode
	for i := range episodes {
		if episodes[i].Episode == uint(episode) {
			ep = &episodes[i]
			break
		}
	}

	if ep == nil {
		ep = &models.Episode{
			SeriesID: series.ID,
			Season:   uint(season),
			Episode:  uint(episode),
		}
		if err := s.db.CreateEpisode(ep); err != nil {
			return err
		}
	}

	// Search for torrents
	for _, provider := range s.providers {
		results, err := provider.SearchSeries(series.Title, season, episode)
		if err != nil {
			continue
		}

		for _, result := range results {
			torrent := result.ToEpisodeTorrent(ep.ID)
			if err := s.db.CreateEpisodeTorrent(torrent); err != nil {
				log.Printf("Failed to save episode torrent: %v", err)
			}
		}
	}

	return nil
}

// StartBackgroundSync starts periodic sync of all content
func (s *SyncService) StartBackgroundSync(interval time.Duration) {
	if s.running {
		return
	}
	s.running = true

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for range ticker.C {
			s.syncAll()
		}
	}()
}

func (s *SyncService) syncAll() {
	log.Println("Starting background sync...")

	// Sync all movies
	movies, _, err := s.db.ListMovies(database.MovieFilter{Limit: 1000})
	if err != nil {
		log.Printf("Failed to list movies for sync: %v", err)
		return
	}

	for _, movie := range movies {
		s.syncMovieTorrents(&movie)
		time.Sleep(1 * time.Second) // Rate limiting
	}

	log.Println("Background sync completed")
}
