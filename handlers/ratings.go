package handlers

import (
	"encoding/json"
	"net/http"

	"torrent-server/database"
	"torrent-server/models"
	"torrent-server/services"
)

type RatingsHandler struct {
	db          *database.DB
	syncService *services.SyncService
}

func NewRatingsHandler(db *database.DB, syncService *services.SyncService) *RatingsHandler {
	return &RatingsHandler{db: db, syncService: syncService}
}

// GetRatings handles POST /api/v2/get_ratings
// Takes array of IMDB codes, returns map of ratings
func (h *RatingsHandler) GetRatings(w http.ResponseWriter, r *http.Request) {
	var imdbCodes []string
	if err := json.NewDecoder(r.Body).Decode(&imdbCodes); err != nil {
		writeError(w, "Invalid request body")
		return
	}

	ratings := make(map[string]models.LocalRating)

	for _, code := range imdbCodes {
		rating, err := h.db.GetMovieRating(code)
		if err == nil && rating != nil {
			ratings[code] = *rating
		}
	}

	writeSuccess(w, ratings)
}

// SyncMovie handles POST /api/v2/sync_movie
// Receives IMDB code and fetches full movie data from providers
func (h *RatingsHandler) SyncMovie(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ImdbCode  string `json:"imdb_code"`
		Franchise string `json:"franchise"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, "Invalid request body")
		return
	}

	if req.ImdbCode == "" {
		writeError(w, "imdb_code is required")
		return
	}

	// Check if movie already exists
	existing, _ := h.db.GetMovieByIMDB(req.ImdbCode)
	if existing != nil {
		// Update franchise if provided
		if req.Franchise != "" && existing.Franchise != req.Franchise {
			existing.Franchise = req.Franchise
			h.db.UpdateMovie(existing)
		}
		writeSuccess(w, map[string]interface{}{
			"synced": false,
			"exists": true,
			"id":     existing.ID,
		})
		return
	}

	// Use SyncService to fetch complete data (OMDB + torrents)
	if h.syncService == nil {
		writeError(w, "Sync service not available")
		return
	}

	movie, err := h.syncService.SyncMovie(req.ImdbCode)
	if err != nil {
		writeError(w, "Failed to sync movie: "+err.Error())
		return
	}

	// Set franchise if provided
	if req.Franchise != "" {
		movie.Franchise = req.Franchise
		h.db.UpdateMovie(movie)
	}

	writeSuccess(w, map[string]interface{}{
		"synced": true,
		"id":     movie.ID,
	})
}

// SyncSeriesRequest represents the request body for syncing a series
type SyncSeriesRequest struct {
	models.Series
	Episodes []SyncEpisodeData `json:"episodes,omitempty"`
}

// SyncEpisodeData represents episode data with torrents for syncing
type SyncEpisodeData struct {
	SeasonNumber  uint                `json:"season_number"`
	EpisodeNumber uint                `json:"episode_number"`
	Title         string              `json:"title"`
	Summary       string              `json:"summary,omitempty"`
	AirDate       string              `json:"air_date,omitempty"`
	Torrents      []SyncTorrentData   `json:"torrents,omitempty"`
}

// SyncTorrentData represents torrent data for syncing
type SyncTorrentData struct {
	Hash          string `json:"hash"`
	Quality       string `json:"quality"`
	Seeds         uint   `json:"seeds"`
	Peers         uint   `json:"peers"`
	Size          string `json:"size"`
	SizeBytes     uint64 `json:"size_bytes"`
	SeasonNumber  uint   `json:"season_number"`
	EpisodeNumber uint   `json:"episode_number"`
}

// SyncSeries handles POST /api/v2/sync_series
// Syncs a TV series - accepts full series data from frontend
func (h *RatingsHandler) SyncSeries(w http.ResponseWriter, r *http.Request) {
	var req SyncSeriesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, "Invalid request body")
		return
	}

	if req.ImdbCode == "" {
		writeError(w, "imdb_code is required")
		return
	}

	// Check if series already exists
	existing, _ := h.db.GetSeriesByIMDB(req.ImdbCode)
	if existing != nil {
		writeSuccess(w, map[string]interface{}{
			"synced": false,
			"exists": true,
			"id":     existing.ID,
		})
		return
	}

	// Create the series with provided data
	series := &req.Series
	if err := h.db.CreateSeries(series); err != nil {
		writeError(w, "Failed to create series: "+err.Error())
		return
	}

	// Process episodes and their torrents
	episodesCreated := 0
	torrentsCreated := 0
	for _, epData := range req.Episodes {
		// Create episode
		episode := &models.Episode{
			SeriesID:      series.ID,
			SeasonNumber:  epData.SeasonNumber,
			EpisodeNumber: epData.EpisodeNumber,
			Title:         epData.Title,
			Summary:       epData.Summary,
			AirDate:       epData.AirDate,
		}
		if err := h.db.CreateEpisode(episode); err != nil {
			continue
		}
		episodesCreated++

		// Create torrents for this episode
		for _, tData := range epData.Torrents {
			torrent := &models.EpisodeTorrent{
				EpisodeID:     episode.ID,
				SeriesID:      series.ID,
				SeasonNumber:  tData.SeasonNumber,
				EpisodeNumber: tData.EpisodeNumber,
				Hash:          tData.Hash,
				Quality:       tData.Quality,
				Seeds:         tData.Seeds,
				Peers:         tData.Peers,
				Size:          tData.Size,
				SizeBytes:     tData.SizeBytes,
			}
			if err := h.db.CreateEpisodeTorrent(torrent); err == nil {
				torrentsCreated++
			}
		}
	}

	writeSuccess(w, map[string]interface{}{
		"synced":           true,
		"id":               series.ID,
		"episodes_created": episodesCreated,
		"torrents_created": torrentsCreated,
	})
}

// SyncMovies handles POST /api/v2/sync_movies
// Batch sync multiple movies
func (h *RatingsHandler) SyncMovies(w http.ResponseWriter, r *http.Request) {
	var movies []models.Movie
	if err := json.NewDecoder(r.Body).Decode(&movies); err != nil {
		writeError(w, "Invalid request body")
		return
	}

	synced := 0
	for _, movie := range movies {
		existing, _ := h.db.GetMovieByIMDB(movie.ImdbCode)
		if existing != nil {
			movie.ID = existing.ID
			h.db.UpdateMovie(&movie)
		} else {
			if err := h.db.CreateMovie(&movie); err == nil {
				synced++
			}
		}

		// Sync torrents
		for _, t := range movie.Torrents {
			t.MovieID = movie.ID
			existing, _ := h.db.GetTorrentByHash(t.Hash)
			if existing == nil {
				h.db.CreateTorrent(&t)
			}
		}
	}

	writeSuccess(w, map[string]interface{}{
		"synced": synced,
		"total":  len(movies),
	})
}

// RefreshAllMovies handles POST /admin/api/refresh_all_movies
// Refreshes all movies in the background
func (h *RatingsHandler) RefreshAllMovies(w http.ResponseWriter, r *http.Request) {
	if h.syncService == nil {
		writeError(w, "Sync service not available")
		return
	}

	// Start background refresh
	go h.syncService.RefreshAllMovies()

	writeSuccess(w, map[string]interface{}{
		"started": true,
		"message": "Background refresh started",
	})
}

func (h *RatingsHandler) RefreshAllSeries(w http.ResponseWriter, r *http.Request) {
	if h.syncService == nil {
		writeError(w, "Sync service not available")
		return
	}

	// Start background refresh
	go h.syncService.RefreshAllSeries()

	writeSuccess(w, map[string]interface{}{
		"started": true,
		"message": "Background TV series refresh started",
	})
}

// RefreshMovie handles POST /api/v2/refresh_movie
// Re-fetches movie data from IMDB/OMDB for existing movies
func (h *RatingsHandler) RefreshMovie(w http.ResponseWriter, r *http.Request) {
	var req struct {
		MovieID  uint   `json:"movie_id"`
		ImdbCode string `json:"imdb_code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, "Invalid request body")
		return
	}

	// Get existing movie
	var movie *models.Movie
	var err error
	if req.MovieID > 0 {
		movie, err = h.db.GetMovie(req.MovieID)
	} else if req.ImdbCode != "" {
		movie, err = h.db.GetMovieByIMDB(req.ImdbCode)
	}
	if err != nil || movie == nil {
		writeError(w, "Movie not found")
		return
	}

	// Use SyncService to refresh data
	if h.syncService == nil {
		writeError(w, "Sync service not available")
		return
	}

	// Refresh the movie data
	refreshed, err := h.syncService.RefreshMovie(movie)
	if err != nil {
		writeError(w, "Failed to refresh movie: "+err.Error())
		return
	}

	writeSuccess(w, map[string]interface{}{
		"refreshed": true,
		"movie":     refreshed,
	})
}

// RefreshSeries handles POST /api/v2/refresh_series
// Re-fetches series data, episodes, and torrents from IMDB/EZTV
func (h *RatingsHandler) RefreshSeries(w http.ResponseWriter, r *http.Request) {
	var req struct {
		SeriesID uint   `json:"series_id"`
		ImdbCode string `json:"imdb_code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, "Invalid request body")
		return
	}

	// Get existing series
	var series *models.Series
	var err error
	if req.SeriesID > 0 {
		series, err = h.db.GetSeries(req.SeriesID)
	} else if req.ImdbCode != "" {
		series, err = h.db.GetSeriesByIMDB(req.ImdbCode)
	}
	if err != nil || series == nil {
		writeError(w, "Series not found")
		return
	}

	// Use SyncService to refresh data
	if h.syncService == nil {
		writeError(w, "Sync service not available")
		return
	}

	// Refresh the series data including episodes
	refreshed, err := h.syncService.RefreshSeries(series)
	if err != nil {
		writeError(w, "Failed to refresh series: "+err.Error())
		return
	}

	writeSuccess(w, map[string]interface{}{
		"refreshed": true,
		"series":    refreshed,
	})
}

// TorrentStats handles GET /api/v2/torrent_stats
// Returns real-time seed/peer info for torrent hashes
func (h *RatingsHandler) TorrentStats(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	// Single hash
	hash := q.Get("hash")
	if hash != "" {
		stats := h.getTorrentStats(hash)
		writeSuccess(w, stats)
		return
	}

	// Multiple hashes
	hashes := q["hashes"]
	if len(hashes) > 0 {
		result := make(map[string]models.TorrentStats)
		for _, hash := range hashes {
			result[hash] = h.getTorrentStats(hash)
		}
		writeSuccess(w, result)
		return
	}

	writeError(w, "hash or hashes parameter required")
}

func (h *RatingsHandler) getTorrentStats(hash string) models.TorrentStats {
	// Try to get from database first
	torrent, err := h.db.GetTorrentByHash(hash)
	if err == nil && torrent != nil {
		return models.TorrentStats{
			Hash:  hash,
			Seeds: torrent.Seeds,
			Peers: torrent.Peers,
			Found: true,
		}
	}

	// Not found
	return models.TorrentStats{
		Hash:  hash,
		Seeds: 0,
		Peers: 0,
		Found: false,
	}
}
