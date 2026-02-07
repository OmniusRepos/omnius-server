package handlers

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/go-chi/chi/v5"

	"torrent-server/services"
)

type IMDBHandler struct {
	imdbService *services.IMDBService
	client      *http.Client
}

func NewIMDBHandler(is *services.IMDBService) *IMDBHandler {
	return &IMDBHandler{
		imdbService: is,
		client:      &http.Client{Timeout: 15 * time.Second},
	}
}

// Images handles GET /api/v2/imdb/images/{imdbCode}
func (h *IMDBHandler) Images(w http.ResponseWriter, r *http.Request) {
	imdbCode := chi.URLParam(r, "imdbCode")
	if imdbCode == "" {
		http.Error(w, "imdbCode required", http.StatusBadRequest)
		return
	}

	images, err := h.imdbService.FetchImages(imdbCode)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(images)
}

// Search handles GET /api/v2/imdb/search?query={q}&type={type}
func (h *IMDBHandler) Search(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("query")
	if query == "" {
		http.Error(w, "query required", http.StatusBadRequest)
		return
	}

	apiURL := "https://api.imdbapi.dev/search/titles?query=" + url.QueryEscape(query)

	// Add type filter if provided (e.g., "tvSeries")
	if typeFilter := r.URL.Query().Get("type"); typeFilter != "" {
		apiURL += "&type=" + url.QueryEscape(typeFilter)
	}

	resp, err := h.client.Get(apiURL)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	w.Header().Set("Content-Type", "application/json")
	io.Copy(w, resp.Body)
}

// Title handles GET /api/v2/imdb/title/{imdbCode}
func (h *IMDBHandler) Title(w http.ResponseWriter, r *http.Request) {
	imdbCode := chi.URLParam(r, "imdbCode")
	if imdbCode == "" {
		http.Error(w, "imdbCode required", http.StatusBadRequest)
		return
	}

	resp, err := h.client.Get("https://api.imdbapi.dev/titles/" + imdbCode)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	w.Header().Set("Content-Type", "application/json")
	io.Copy(w, resp.Body)
}
