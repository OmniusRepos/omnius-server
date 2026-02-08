package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
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
// Returns stored subs from DB, supplements with external API for missing languages
func (h *SubtitleHandler) Search(w http.ResponseWriter, r *http.Request) {
	imdbID := r.URL.Query().Get("imdb_id")
	if imdbID == "" {
		http.Error(w, "imdb_id required", http.StatusBadRequest)
		return
	}
	languages := r.URL.Query().Get("languages")

	var subtitles []services.Subtitle

	// Check DB for stored subtitles
	if h.db != nil {
		stored, err := h.db.GetSubtitlesByIMDB(imdbID, languages)
		if err == nil && len(stored) > 0 {
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
		}
	}

	// Find which requested languages are missing from stored results
	if languages != "" {
		haveLangs := make(map[string]bool)
		for _, s := range subtitles {
			haveLangs[s.Language] = true
		}
		var missingLangs []string
		for _, lang := range strings.Split(languages, ",") {
			lang = strings.TrimSpace(lang)
			if lang != "" && !haveLangs[lang] {
				missingLangs = append(missingLangs, lang)
			}
		}

		// Search external APIs for missing languages
		if len(missingLangs) > 0 {
			extResult, err := h.subtitleService.SearchByIMDB(imdbID, strings.Join(missingLangs, ","))
			if err == nil {
				subtitles = append(subtitles, extResult.Subtitles...)
			}
		}
	} else if len(subtitles) == 0 {
		// No languages specified and nothing stored — search externally
		extResult, err := h.subtitleService.SearchByIMDB(imdbID, "")
		if err == nil {
			subtitles = append(subtitles, extResult.Subtitles...)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(&services.SubtitleSearchResult{
		Subtitles:  subtitles,
		TotalCount: len(subtitles),
	})
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
// Serves VTT content from disk file (fallback to DB for old rows)
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

	var vttContent string
	if sub.VTTPath != "" {
		data, err := os.ReadFile(sub.VTTPath)
		if err == nil {
			vttContent = string(data)
		} else {
			// File missing, fallback to DB content
			vttContent = sub.VTTContent
		}
	} else {
		vttContent = sub.VTTContent
	}

	w.Header().Set("Content-Type", "text/vtt; charset=utf-8")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Write([]byte(vttContent))
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

	// Remove file from disk if it exists
	sub, err := h.db.GetSubtitleByID(uint(id))
	if err == nil && sub.VTTPath != "" {
		os.Remove(sub.VTTPath)
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

	// Read VTT content from file or DB fallback
	vttContent := sub.VTTContent
	if sub.VTTPath != "" {
		if data, err := os.ReadFile(sub.VTTPath); err == nil {
			vttContent = string(data)
		}
	}

	// Extract first 20 lines
	lines := strings.SplitN(vttContent, "\n", 21)
	totalLines := strings.Count(vttContent, "\n") + 1
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
