package services

import (
	"fmt"
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
	imdb      *IMDBService
	mu        sync.Mutex
	running   bool
}

func NewSyncService(db *database.DB) *SyncService {
	return &SyncService{
		db:   db,
		imdb: NewIMDBService(),
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

	// Fetch rich data from IMDB API
	richData, err := s.imdb.FetchRichData(imdbCode)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch from IMDB: %w", err)
	}

	movie := richData.ToMovie(imdbCode)
	// Set additional data from rich response
	if len(richData.Directors) > 0 {
		movie.Director = richData.Directors[0]
	}
	movie.Writers = richData.Writers
	movie.Cast = richData.Cast
	movie.Budget = richData.Budget
	movie.BoxOfficeGross = richData.BoxOfficeGross
	movie.AllImages = richData.AllImages
	movie.Provider = "imdb"
	movie.ContentType = "movie"
	log.Printf("Fetched rich data from IMDB for %s", imdbCode)

	// Save movie
	if err := s.db.CreateMovie(movie); err != nil {
		return nil, err
	}

	// Fetch torrents
	s.syncMovieTorrents(movie)

	return movie, nil
}

// RefreshAllMovies refreshes all movies in the database
func (s *SyncService) RefreshAllMovies() {
	log.Println("Starting refresh all movies...")

	movies, _, err := s.db.ListMovies(database.MovieFilter{Limit: 10000})
	if err != nil {
		log.Printf("Failed to list movies: %v", err)
		return
	}

	total := len(movies)
	refreshed := 0
	failed := 0

	for i, movie := range movies {
		if movie.ImdbCode == "" {
			log.Printf("[%d/%d] Skipping %s - no IMDB code", i+1, total, movie.Title)
			continue
		}

		log.Printf("[%d/%d] Refreshing %s (%s)...", i+1, total, movie.Title, movie.ImdbCode)
		_, err := s.RefreshMovie(&movie)
		if err != nil {
			log.Printf("  Failed: %v", err)
			failed++
		} else {
			log.Printf("  Done")
			refreshed++
		}

		// Rate limiting - don't hammer the APIs
		time.Sleep(2 * time.Second)
	}

	log.Printf("Refresh all movies completed: %d refreshed, %d failed, %d total", refreshed, failed, total)
}

// RefreshMovie re-fetches data for an existing movie
func (s *SyncService) RefreshMovie(movie *models.Movie) (*models.Movie, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if movie.ImdbCode == "" {
		return nil, fmt.Errorf("movie has no IMDB code")
	}

	// Fetch rich data from IMDB
	richData, err := s.imdb.FetchRichData(movie.ImdbCode)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch from IMDB: %w", err)
	}

	// Update movie with rich data
	movie.Title = richData.Title
	if richData.OriginalTitle != "" {
		movie.TitleEnglish = richData.OriginalTitle
	}
	movie.Year = uint(richData.Year)
	movie.Runtime = uint(richData.Runtime)
	movie.Genres = richData.Genres
	movie.Summary = richData.Plot
	movie.DescriptionFull = richData.Plot
	movie.MpaRating = richData.ContentRating

	if richData.Rating > 0 {
		r := float32(richData.Rating)
		movie.Rating = r
		movie.ImdbRating = &r
	}
	if richData.VoteCount > 0 {
		movie.ImdbVotes = formatVotes(richData.VoteCount)
	}
	if richData.Metacritic > 0 {
		movie.Metacritic = &richData.Metacritic
	}
	if len(richData.Directors) > 0 {
		movie.Director = richData.Directors[0]
	}
	movie.Writers = richData.Writers
	movie.Cast = richData.Cast
	movie.Budget = richData.Budget
	movie.BoxOfficeGross = richData.BoxOfficeGross
	movie.AllImages = richData.AllImages

	if richData.PosterURL != "" {
		movie.SmallCoverImage = richData.PosterURL
		movie.MediumCoverImage = richData.PosterURL
		movie.LargeCoverImage = richData.PosterURL
	}
	if richData.BackgroundURL != "" {
		movie.BackgroundImage = richData.BackgroundURL
	}

	movie.Provider = "imdb"
	log.Printf("Refreshed movie %s from IMDB", movie.ImdbCode)

	// Save updated movie
	if err := s.db.UpdateMovie(movie); err != nil {
		return nil, fmt.Errorf("failed to save refreshed movie: %w", err)
	}

	// Also sync torrents
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
		if episodes[i].EpisodeNumber == uint(episode) {
			ep = &episodes[i]
			break
		}
	}

	if ep == nil {
		ep = &models.Episode{
			SeriesID:      series.ID,
			SeasonNumber:  uint(season),
			EpisodeNumber: uint(episode),
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
