package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"torrent-server/database"
	"torrent-server/services"
)

type SubtitleHandler struct {
	subtitleService *services.SubtitleService
	db              *database.DB
}

func NewSubtitleHandler(ss *services.SubtitleService, db *database.DB) *SubtitleHandler {
	return &SubtitleHandler{subtitleService: ss, db: db}
}

// Search handles GET /api/v2/subtitles/search?imdb_id={id}&languages={langs}
// Checks DB first, falls back to external API
func (h *SubtitleHandler) Search(w http.ResponseWriter, r *http.Request) {
	imdbID := r.URL.Query().Get("imdb_id")
	if imdbID == "" {
		http.Error(w, "imdb_id required", http.StatusBadRequest)
		return
	}
	languages := r.URL.Query().Get("languages")

	// Check DB first
	if h.db != nil {
		stored, err := h.db.GetSubtitlesByIMDB(imdbID, languages)
		if err == nil && len(stored) > 0 {
			// Convert stored subtitles to API response format
			subtitles := make([]services.Subtitle, 0, len(stored))
			for _, s := range stored {
				subtitles = append(subtitles, services.Subtitle{
					ID:              fmt.Sprintf("stored-%d", s.ID),
					Language:        s.Language,
					LanguageName:    s.LanguageName,
					DownloadURL:     fmt.Sprintf("/api/v2/subtitles/stored/%d", s.ID),
					ReleaseName:     s.ReleaseName,
					HearingImpaired: s.HearingImpaired,
				})
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(&services.SubtitleSearchResult{
				Subtitles:  subtitles,
				TotalCount: len(subtitles),
			})
			return
		}
	}

	// Fallback to external API
	result, err := h.subtitleService.SearchByIMDB(imdbID, languages)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// SearchByFilename handles GET /api/v2/subtitles/search_by_filename?filename={name}&languages={langs}
func (h *SubtitleHandler) SearchByFilename(w http.ResponseWriter, r *http.Request) {
	filename := r.URL.Query().Get("filename")
	if filename == "" {
		http.Error(w, "filename required", http.StatusBadRequest)
		return
	}
	languages := r.URL.Query().Get("languages")

	result, err := h.subtitleService.SearchByFilename(filename, languages)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// Download handles GET /api/v2/subtitles/download?url={encoded_url}
// Returns VTT content directly
func (h *SubtitleHandler) Download(w http.ResponseWriter, r *http.Request) {
	downloadURL := r.URL.Query().Get("url")
	if downloadURL == "" {
		http.Error(w, "url required", http.StatusBadRequest)
		return
	}

	vtt, err := h.subtitleService.DownloadSubtitle(downloadURL)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/vtt; charset=utf-8")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Write([]byte(vtt))
}

// ServeStored handles GET /api/v2/subtitles/stored/{id}
// Serves VTT content from DB
func (h *SubtitleHandler) ServeStored(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	sub, err := h.db.GetSubtitleByID(uint(id))
	if err != nil {
		http.Error(w, "subtitle not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "text/vtt; charset=utf-8")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Write([]byte(sub.VTTContent))
}

// SyncSubtitles handles POST /admin/api/subtitles/sync
func (h *SubtitleHandler) SyncSubtitles(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ImdbCode  string `json:"imdb_code"`
		Languages string `json:"languages"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.ImdbCode == "" {
		http.Error(w, "imdb_code required", http.StatusBadRequest)
		return
	}
	if req.Languages == "" {
		req.Languages = "en"
	}

	count, err := h.subtitleService.SyncSubtitles(req.ImdbCode, req.Languages)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "ok",
		"stored":  count,
		"message": fmt.Sprintf("Stored %d subtitles for %s", count, req.ImdbCode),
	})
}

// ListStored handles GET /admin/api/subtitles?imdb_code=...
func (h *SubtitleHandler) ListStored(w http.ResponseWriter, r *http.Request) {
	imdbCode := r.URL.Query().Get("imdb_code")

	stored, err := h.db.GetSubtitlesByIMDB(imdbCode, "")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"subtitles": stored,
		"count":     len(stored),
	})
}

// DeleteStored handles DELETE /admin/api/subtitles/{id}
func (h *SubtitleHandler) DeleteStored(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	if err := h.db.DeleteSubtitle(uint(id)); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// PreviewStored handles GET /admin/api/subtitles/{id}/preview
// Returns the first 20 lines of VTT content for preview
func (h *SubtitleHandler) PreviewStored(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	sub, err := h.db.GetSubtitleByID(uint(id))
	if err != nil {
		http.Error(w, "subtitle not found", http.StatusNotFound)
		return
	}

	// Extract first 20 lines
	lines := strings.SplitN(sub.VTTContent, "\n", 21)
	totalLines := strings.Count(sub.VTTContent, "\n") + 1
	preview := strings.Join(lines, "\n")
	if len(lines) > 20 {
		preview = strings.Join(lines[:20], "\n")
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":           sub.ID,
		"preview":      preview,
		"language":     sub.Language,
		"release_name": sub.ReleaseName,
		"total_lines":  totalLines,
	})
}

// Languages handles GET /api/v2/subtitle_languages
func (h *SubtitleHandler) Languages(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(services.GetSubtitleLanguages())
}
