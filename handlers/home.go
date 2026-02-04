package handlers

import (
	"encoding/json"
	"net/http"

	"torrent-server/database"

	"github.com/go-chi/chi/v5"
)

type HomeHandler struct {
	db *database.DB
}

func NewHomeHandler(db *database.DB) *HomeHandler {
	return &HomeHandler{db: db}
}

// AdminListHomeSections handles GET /admin/api/home/sections
func (h *HomeHandler) AdminListHomeSections(w http.ResponseWriter, r *http.Request) {
	sections, err := h.db.ListHomeSections(true) // include inactive
	if err != nil {
		http.Error(w, "Failed to fetch home sections", http.StatusInternalServerError)
		return
	}

	if sections == nil {
		sections = []database.HomeSection{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(sections)
}

// AdminCreateHomeSection handles POST /admin/api/home/sections
func (h *HomeHandler) AdminCreateHomeSection(w http.ResponseWriter, r *http.Request) {
	var section database.HomeSection
	if err := json.NewDecoder(r.Body).Decode(&section); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Set defaults
	if section.SortBy == "" {
		section.SortBy = "rating"
	}
	if section.OrderBy == "" {
		section.OrderBy = "desc"
	}
	if section.LimitCount == 0 {
		section.LimitCount = 10
	}
	section.IsActive = true

	if err := h.db.CreateHomeSection(&section); err != nil {
		http.Error(w, "Failed to create section: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(section)
}

// AdminUpdateHomeSection handles PUT /admin/api/home/sections/{id}
func (h *HomeHandler) AdminUpdateHomeSection(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var section database.HomeSection
	if err := json.NewDecoder(r.Body).Decode(&section); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	section.ID = uint(parseInt(id, 0))
	if err := h.db.UpdateHomeSection(&section); err != nil {
		http.Error(w, "Failed to update section", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(section)
}

// AdminDeleteHomeSection handles DELETE /admin/api/home/sections/{id}
func (h *HomeHandler) AdminDeleteHomeSection(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if err := h.db.DeleteHomeSection(uint(parseInt(id, 0))); err != nil {
		http.Error(w, "Failed to delete section", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// AdminReorderHomeSections handles POST /admin/api/home/sections/reorder
func (h *HomeHandler) AdminReorderHomeSections(w http.ResponseWriter, r *http.Request) {
	var req struct {
		IDs []uint `json:"ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if err := h.db.ReorderHomeSections(req.IDs); err != nil {
		http.Error(w, "Failed to reorder sections", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
