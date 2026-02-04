package handlers

import (
	"encoding/json"
	"net/http"

	"torrent-server/services"
)

type SyncHandler struct {
	syncService *services.SyncService
}

func NewSyncHandler(ss *services.SyncService) *SyncHandler {
	return &SyncHandler{syncService: ss}
}

// SyncMovie handles POST /admin/sync/movie
func (h *SyncHandler) SyncMovie(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	imdbCode := r.FormValue("imdb_code")
	if imdbCode == "" {
		http.Error(w, "imdb_code is required", http.StatusBadRequest)
		return
	}

	movie, err := h.syncService.SyncMovie(imdbCode)
	if err != nil {
		http.Error(w, "Failed to sync movie: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "ok",
		"message": "Movie synced successfully",
		"movie":   movie,
	})
}

// SyncSeries handles POST /admin/sync/series
func (h *SyncHandler) SyncSeries(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	imdbCode := r.FormValue("imdb_code")
	if imdbCode == "" {
		http.Error(w, "imdb_code is required", http.StatusBadRequest)
		return
	}

	series, err := h.syncService.SyncSeries(imdbCode)
	if err != nil {
		http.Error(w, "Failed to sync series: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "ok",
		"message": "Series synced successfully",
		"series":  series,
	})
}
