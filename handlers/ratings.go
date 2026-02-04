package handlers

import (
	"encoding/json"
	"net/http"

	"torrent-server/database"
	"torrent-server/models"
)

type RatingsHandler struct {
	db *database.DB
}

func NewRatingsHandler(db *database.DB) *RatingsHandler {
	return &RatingsHandler{db: db}
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
// Receives a movie from client and saves to local DB
func (h *RatingsHandler) SyncMovie(w http.ResponseWriter, r *http.Request) {
	var movie models.Movie
	if err := json.NewDecoder(r.Body).Decode(&movie); err != nil {
		writeError(w, "Invalid request body")
		return
	}

	// Check if movie exists
	existing, _ := h.db.GetMovieByIMDB(movie.ImdbCode)
	if existing != nil {
		// Update existing
		movie.ID = existing.ID
		if err := h.db.UpdateMovie(&movie); err != nil {
			writeError(w, "Failed to update movie: "+err.Error())
			return
		}
	} else {
		// Create new
		if err := h.db.CreateMovie(&movie); err != nil {
			writeError(w, "Failed to create movie: "+err.Error())
			return
		}
	}

	// Also sync torrents if present
	for _, t := range movie.Torrents {
		t.MovieID = movie.ID
		existing, _ := h.db.GetTorrentByHash(t.Hash)
		if existing == nil {
			h.db.CreateTorrent(&t)
		}
	}

	writeSuccess(w, map[string]interface{}{
		"synced": true,
		"id":     movie.ID,
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
