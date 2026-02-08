package services

import (
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"torrent-server/database"
	"torrent-server/models"
	"torrent-server/providers"
)

type SyncService struct {
	db              *database.DB
	providers       []providers.TorrentProvider
	imdb            *IMDBService
	subtitleService *SubtitleService
	mu              sync.Mutex
	running         bool
}

func NewSyncService(db *database.DB, subtitlesDir string) *SyncService {
	return &SyncService{
		db:              db,
		imdb:            NewIMDBService(),
		subtitleService: NewSubtitleServiceWithDB(db, subtitlesDir),
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
		go s.syncSubtitles(imdbCode)
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

	// Sync subtitles in background
	go s.syncSubtitles(imdbCode)

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

// RefreshAllSeries refreshes all series in the database
func (s *SyncService) RefreshAllSeries() {
	log.Println("Starting refresh all TV series...")

	seriesList, _, err := s.db.ListSeries(1000, 1)
	if err != nil {
		log.Printf("Failed to list series: %v", err)
		return
	}

	total := len(seriesList)
	refreshed := 0
	failed := 0

	for i, series := range seriesList {
		if series.ImdbCode == "" {
			log.Printf("[%d/%d] Skipping %s - no IMDB code", i+1, total, series.Title)
			continue
		}

		log.Printf("[%d/%d] Refreshing %s (%s)...", i+1, total, series.Title, series.ImdbCode)
		_, err := s.RefreshSeries(&series)
		if err != nil {
			log.Printf("  Failed: %v", err)
			failed++
		} else {
			log.Printf("  Done")
			refreshed++
		}

		// Rate limiting - don't hammer the APIs
		time.Sleep(3 * time.Second)
	}

	log.Printf("Refresh all TV series completed: %d refreshed, %d failed, %d total", refreshed, failed, total)
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

	// Sync subtitles in background
	go s.syncSubtitles(movie.ImdbCode)

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

// SyncSeriesWithData creates a series with provided metadata
func (s *SyncService) SyncSeriesWithData(series *models.Series) (*models.Series, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if series already exists
	existing, err := s.db.GetSeriesByIMDB(series.ImdbCode)
	if err == nil && existing != nil {
		return existing, nil
	}

	if err := s.db.CreateSeries(series); err != nil {
		return nil, err
	}

	log.Printf("[SyncSeries] Created series: %s (ID: %d)", series.Title, series.ID)
	return series, nil
}

// SyncSeries fetches metadata and torrents for a series (legacy, just creates basic entry)
func (s *SyncService) SyncSeries(imdbCode string) (*models.Series, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if series already exists
	existing, err := s.db.GetSeriesByIMDB(imdbCode)
	if err == nil && existing != nil {
		return existing, nil
	}

	series := &models.Series{
		ImdbCode: imdbCode,
		Status:   "ongoing",
	}

	if err := s.db.CreateSeries(series); err != nil {
		return nil, err
	}

	return series, nil
}

// RefreshSeries re-fetches all data for an existing series including episodes
func (s *SyncService) RefreshSeries(series *models.Series) (*models.Series, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if series.ImdbCode == "" {
		return nil, fmt.Errorf("series has no IMDB code")
	}

	// Fetch rich data from IMDB
	richData, err := s.imdb.FetchRichSeriesData(series.ImdbCode)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch from IMDB: %w", err)
	}

	// Update series with rich data
	series.Title = richData.Title
	series.TitleSlug = strings.ToLower(strings.ReplaceAll(richData.Title, " ", "-"))
	series.Year = uint(richData.Year)
	if richData.EndYear != nil {
		endYear := uint(*richData.EndYear)
		series.EndYear = &endYear
	}
	series.Runtime = uint(richData.Runtime)
	series.Genres = richData.Genres
	series.Summary = richData.Plot
	series.Status = richData.Status
	series.TotalSeasons = uint(richData.TotalSeasons)

	if richData.Rating > 0 {
		series.Rating = float32(richData.Rating)
		r := float32(richData.Rating)
		series.ImdbRating = &r
	}

	if richData.PosterURL != "" {
		series.PosterImage = richData.PosterURL
	}
	if richData.BackgroundURL != "" {
		series.BackgroundImage = richData.BackgroundURL
	}

	log.Printf("Refreshed series %s from IMDB: %d seasons found", series.ImdbCode, len(richData.Seasons))

	// Save updated series
	if err := s.db.UpdateSeries(series); err != nil {
		return nil, fmt.Errorf("failed to save refreshed series: %w", err)
	}

	// Sync episodes from IMDB data
	totalEpisodes := 0
	for _, season := range richData.Seasons {
		for _, ep := range season.Episodes {
			// Parse season number from string
			seasonNum := season.Season

			// Format air date from ReleaseDate
			airDate := ""
			if ep.ReleaseDate != nil {
				airDate = fmt.Sprintf("%04d-%02d-%02d", ep.ReleaseDate.Year, ep.ReleaseDate.Month, ep.ReleaseDate.Day)
			}

			episode := &models.Episode{
				SeriesID:      series.ID,
				SeasonNumber:  uint(seasonNum),
				EpisodeNumber: uint(ep.EpisodeNumber),
				Title:         ep.Title,
				Summary:       ep.Plot,
				AirDate:       airDate,
			}
			if ep.RuntimeSeconds > 0 {
				runtime := uint(ep.RuntimeSeconds / 60)
				episode.Runtime = &runtime
			}
			if ep.PrimaryImage != nil {
				episode.StillImage = ep.PrimaryImage.URL
			}

			// CreateEpisode uses ON CONFLICT DO UPDATE, so it will update existing
			if err := s.db.CreateEpisode(episode); err != nil {
				log.Printf("Failed to save episode S%02dE%02d: %v", seasonNum, ep.EpisodeNumber, err)
			} else {
				totalEpisodes++
			}
		}
	}

	log.Printf("Synced %d episodes for %s", totalEpisodes, series.Title)

	// Update total episodes count
	series.TotalEpisodes = uint(totalEpisodes)
	s.db.UpdateSeries(series)

	// Sync torrents from EZTV
	s.syncSeriesEpisodeTorrents(series)

	// Sync subtitles per-episode in background
	go s.syncSeriesSubtitles(series)

	return series, nil
}

// syncSeriesEpisodeTorrents fetches torrents from EZTV for all episodes
func (s *SyncService) syncSeriesEpisodeTorrents(series *models.Series) {
	// Strip "tt" prefix for EZTV API
	imdbID := strings.TrimPrefix(series.ImdbCode, "tt")

	// Get all torrents from EZTV for this series
	results, err := providers.FetchEZTVTorrents(imdbID)
	if err != nil {
		log.Printf("Failed to fetch EZTV torrents for %s: %v", series.Title, err)
		return
	}

	log.Printf("Found %d torrents from EZTV for %s", len(results), series.Title)

	seasonPackCount := 0
	episodeTorrentCount := 0

	// Group episode torrents by S##E##-quality, keep only best (most seeds) per quality
	type epKey struct {
		Season  int
		Episode int
		Quality string
	}
	bestPerQuality := make(map[epKey]providers.EZTVSeriesResult)

	for _, result := range results {
		if result.Season == 0 {
			continue
		}

		// Season packs go straight through
		if result.Episode == 0 {
			pack := &models.SeasonPack{
				SeriesID:  series.ID,
				Season:    uint(result.Season),
				Hash:      result.Hash,
				Quality:   result.Quality,
				Seeds:     uint(result.Seeds),
				Peers:     uint(result.Peers),
				Size:      result.Size,
				SizeBytes: result.SizeBytes,
				Source:    "EZTV",
			}
			if err := s.db.CreateSeasonPack(pack); err == nil {
				seasonPackCount++
			}
			continue
		}

		// Normalize quality - only keep 720p, 1080p, 2160p tiers
		q := result.Quality
		switch q {
		case "720p", "1080p", "2160p":
			// valid
		case "480p", "":
			q = "480p"
		default:
			q = "480p"
		}
		result.Quality = q

		key := epKey{Season: result.Season, Episode: result.Episode, Quality: q}
		if existing, ok := bestPerQuality[key]; !ok || result.Seeds > existing.Seeds {
			bestPerQuality[key] = result
		}
	}

	// Now save only the best torrent per episode per quality
	for _, result := range bestPerQuality {
		// Get or create episode
		episodes, _ := s.db.GetEpisodes(series.ID, result.Season)
		var episode *models.Episode
		for i := range episodes {
			if episodes[i].EpisodeNumber == uint(result.Episode) {
				episode = &episodes[i]
				break
			}
		}

		if episode == nil {
			episode = &models.Episode{
				SeriesID:      series.ID,
				SeasonNumber:  uint(result.Season),
				EpisodeNumber: uint(result.Episode),
				Title:         fmt.Sprintf("Episode %d", result.Episode),
			}
			if err := s.db.CreateEpisode(episode); err != nil {
				continue
			}
		}

		torrent := &models.EpisodeTorrent{
			EpisodeID:     episode.ID,
			SeriesID:      series.ID,
			SeasonNumber:  uint(result.Season),
			EpisodeNumber: uint(result.Episode),
			Hash:          result.Hash,
			Quality:       result.Quality,
			Seeds:         uint(result.Seeds),
			Peers:         uint(result.Peers),
			Size:          result.Size,
			SizeBytes:     result.SizeBytes,
		}

		if err := s.db.CreateEpisodeTorrent(torrent); err == nil {
			episodeTorrentCount++
		}
	}

	log.Printf("Saved %d season packs and %d episode torrents for %s (deduped from %d)", seasonPackCount, episodeTorrentCount, series.Title, len(results))
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

func (s *SyncService) syncSeriesSubtitles(series *models.Series) {
	if s.subtitleService == nil || series.ImdbCode == "" {
		return
	}
	languages := "en,sq,es,fr,de,it,pt,tr,ar"

	for seasonNum := 1; seasonNum <= int(series.TotalSeasons); seasonNum++ {
		episodes, err := s.db.GetEpisodes(series.ID, seasonNum)
		if err != nil {
			continue
		}
		for _, ep := range episodes {
			// Skip if already synced
			count, _ := s.db.CountSubtitlesByIMDBEpisode(series.ImdbCode, int(ep.SeasonNumber), int(ep.EpisodeNumber))
			if count > 0 {
				continue
			}
			n, err := s.subtitleService.SyncEpisodeSubtitles(series.ImdbCode, languages, int(ep.SeasonNumber), int(ep.EpisodeNumber))
			if err != nil {
				log.Printf("[SyncService] Failed to sync subtitles for %s S%02dE%02d: %v", series.ImdbCode, ep.SeasonNumber, ep.EpisodeNumber, err)
			} else if n > 0 {
				log.Printf("[SyncService] Synced %d subtitles for %s S%02dE%02d", n, series.ImdbCode, ep.SeasonNumber, ep.EpisodeNumber)
			}
			time.Sleep(1 * time.Second) // Rate limit
		}
	}
}

func (s *SyncService) syncSubtitles(imdbCode string) {
	if s.subtitleService == nil {
		return
	}
	// Skip if subtitles already synced for this movie
	count, _ := s.db.CountSubtitlesByIMDB(imdbCode)
	if count > 0 {
		return
	}
	languages := "en,sq,es,fr,de,it,pt,tr,ar"
	count, err := s.subtitleService.SyncSubtitles(imdbCode, languages)
	if err != nil {
		log.Printf("[SyncService] Failed to sync subtitles for %s: %v", imdbCode, err)
	} else if count > 0 {
		log.Printf("[SyncService] Auto-synced %d subtitles for %s", count, imdbCode)
	}
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
		// Auto-sync subtitles for movies that don't have any
		if movie.ImdbCode != "" {
			s.syncSubtitles(movie.ImdbCode)
		}
		time.Sleep(1 * time.Second) // Rate limiting
	}

	log.Println("Background sync completed")
}
