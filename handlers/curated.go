package handlers

import (
	"encoding/json"
	"net/http"

	"torrent-server/database"
	"torrent-server/models"

	"github.com/go-chi/chi/v5"
)

type CuratedHandler struct {
	db *database.DB
}

func NewCuratedHandler(db *database.DB) *CuratedHandler {
	return &CuratedHandler{db: db}
}

// ListCuratedLists handles GET /api/v2/curated_lists.json
func (h *CuratedHandler) ListCuratedLists(w http.ResponseWriter, r *http.Request) {
	lists, err := h.db.ListCuratedLists(false)
	if err != nil {
		writeError(w, "Failed to fetch curated lists")
		return
	}

	if lists == nil {
		lists = []models.CuratedList{}
	}

	writeSuccess(w, models.CuratedListData{Lists: lists})
}

// GetCuratedList handles GET /api/v2/curated_list.json?list_id=X or ?slug=X
func (h *CuratedHandler) GetCuratedList(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	idOrSlug := q.Get("list_id")
	if idOrSlug == "" {
		idOrSlug = q.Get("slug")
	}
	if idOrSlug == "" {
		writeError(w, "list_id or slug is required")
		return
	}

	list, err := h.db.GetCuratedList(idOrSlug)
	if err != nil {
		writeError(w, "Curated list not found")
		return
	}

	// Get movies for this list
	movies, _ := h.db.GetCuratedListMovies(list)
	list.Movies = movies

	writeSuccess(w, models.CuratedListDetailsData{List: *list})
}

// =============== Admin Handlers ===============

// AdminListCuratedLists handles GET /admin/api/curated (includes inactive)
func (h *CuratedHandler) AdminListCuratedLists(w http.ResponseWriter, r *http.Request) {
	lists, err := h.db.ListCuratedLists(true)
	if err != nil {
		http.Error(w, "Failed to fetch curated lists", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(lists)
}

// AdminCreateCuratedList handles POST /admin/api/curated
func (h *CuratedHandler) AdminCreateCuratedList(w http.ResponseWriter, r *http.Request) {
	var list models.CuratedList
	if err := json.NewDecoder(r.Body).Decode(&list); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Set defaults
	if list.SortBy == "" {
		list.SortBy = "rating"
	}
	if list.OrderBy == "" {
		list.OrderBy = "desc"
	}
	if list.LimitCount == 0 {
		list.LimitCount = 50
	}
	list.IsActive = true

	if err := h.db.CreateCuratedList(&list); err != nil {
		http.Error(w, "Failed to create curated list: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(list)
}

// AdminUpdateCuratedList handles PUT /admin/api/curated/{id}
func (h *CuratedHandler) AdminUpdateCuratedList(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var list models.CuratedList
	if err := json.NewDecoder(r.Body).Decode(&list); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	list.ID = uint(parseInt(id, 0))
	if err := h.db.UpdateCuratedList(&list); err != nil {
		http.Error(w, "Failed to update curated list", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(list)
}

// AdminDeleteCuratedList handles DELETE /admin/api/curated/{id}
func (h *CuratedHandler) AdminDeleteCuratedList(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if err := h.db.DeleteCuratedList(uint(parseInt(id, 0))); err != nil {
		http.Error(w, "Failed to delete curated list", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// AdminAddMovieToList handles POST /admin/api/curated/{id}/movies
func (h *CuratedHandler) AdminAddMovieToList(w http.ResponseWriter, r *http.Request) {
	listID := chi.URLParam(r, "id")

	var req struct {
		MovieID uint `json:"movie_id"`
		Order   int  `json:"order"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if err := h.db.AddMovieToCuratedList(uint(parseInt(listID, 0)), req.MovieID, req.Order); err != nil {
		http.Error(w, "Failed to add movie to list", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

// AdminRemoveMovieFromList handles DELETE /admin/api/curated/{id}/movies/{movieId}
func (h *CuratedHandler) AdminRemoveMovieFromList(w http.ResponseWriter, r *http.Request) {
	listID := chi.URLParam(r, "id")
	movieID := chi.URLParam(r, "movieId")

	if err := h.db.RemoveMovieFromCuratedList(uint(parseInt(listID, 0)), uint(parseInt(movieID, 0))); err != nil {
		http.Error(w, "Failed to remove movie from list", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
