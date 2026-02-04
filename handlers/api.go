package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"torrent-server/database"
	"torrent-server/models"
)

type APIHandler struct {
	db *database.DB
}

func NewAPIHandler(db *database.DB) *APIHandler {
	return &APIHandler{db: db}
}

// ListMovies handles GET /api/v2/list_movies.json
func (h *APIHandler) ListMovies(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	filter := database.MovieFilter{
		Limit:         parseInt(q.Get("limit"), 20),
		Page:          parseInt(q.Get("page"), 1),
		Quality:       q.Get("quality"),
		MinimumRating: parseFloat(q.Get("minimum_rating"), 0),
		QueryTerm:     q.Get("query_term"),
		Genre:         q.Get("genre"),
		SortBy:        q.Get("sort_by"),
		OrderBy:       q.Get("order_by"),
	}

	movies, totalCount, err := h.db.ListMovies(filter)
	if err != nil {
		writeError(w, "Failed to fetch movies: "+err.Error())
		return
	}

	if movies == nil {
		movies = []models.Movie{}
	}

	data := models.MovieListData{
		MovieCount: totalCount,
		Limit:      filter.Limit,
		PageNumber: filter.Page,
		Movies:     movies,
	}

	writeSuccess(w, data)
}

// MovieDetails handles GET /api/v2/movie_details.json
func (h *APIHandler) MovieDetails(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	movieID := parseInt(q.Get("movie_id"), 0)
	if movieID == 0 {
		writeError(w, "movie_id is required")
		return
	}

	movie, err := h.db.GetMovie(uint(movieID))
	if err != nil {
		writeError(w, "Movie not found")
		return
	}

	data := models.MovieDetailsData{
		Movie: *movie,
	}

	writeSuccess(w, data)
}

// MovieSuggestions handles GET /api/v2/movie_suggestions.json
func (h *APIHandler) MovieSuggestions(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	movieID := parseInt(q.Get("movie_id"), 0)
	if movieID == 0 {
		writeError(w, "movie_id is required")
		return
	}

	movies, err := h.db.GetMovieSuggestions(uint(movieID), 4)
	if err != nil {
		movies = []models.Movie{}
	}

	data := models.MovieSuggestionsData{
		MovieCount: len(movies),
		Movies:     movies,
	}

	writeSuccess(w, data)
}

func writeSuccess(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	response := models.ApiResponse[interface{}]{
		Status:        "ok",
		StatusMessage: "Query was successful",
		Data:          data,
	}

	json.NewEncoder(w).Encode(response)
}

func writeError(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(http.StatusBadRequest)

	response := models.NewErrorResponse(message)
	json.NewEncoder(w).Encode(response)
}

func parseInt(s string, defaultVal int) int {
	if s == "" {
		return defaultVal
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return defaultVal
	}
	return v
}

func parseFloat(s string, defaultVal float32) float32 {
	if s == "" {
		return defaultVal
	}
	v, err := strconv.ParseFloat(s, 32)
	if err != nil {
		return defaultVal
	}
	return float32(v)
}
