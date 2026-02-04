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
		Year:          parseInt(q.Get("year"), 0),
		MinimumYear:   parseInt(q.Get("minimum_year"), 0),
		MaximumYear:   parseInt(q.Get("maximum_year"), 0),
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

// HomeSectionResponse is the response format for home sections
type HomeSectionResponse struct {
	ID     string         `json:"id"`
	Title  string         `json:"title"`
	Type   string         `json:"type"`
	Movies []models.Movie `json:"movies,omitempty"`
}

// Home handles GET /api/v2/home.json - returns home page content
func (h *APIHandler) Home(w http.ResponseWriter, r *http.Request) {
	var sections []HomeSectionResponse

	// Get configured home sections from database
	dbSections, err := h.db.ListHomeSections(false)
	if err != nil || len(dbSections) == 0 {
		// Fallback to default sections if none configured
		sections = h.getDefaultHomeSections()
	} else {
		// Build sections from database config
		for _, s := range dbSections {
			var movies []models.Movie

			switch s.SectionType {
			case "recent":
				movies, _, _ = h.db.ListMovies(database.MovieFilter{
					Limit:   s.LimitCount,
					Page:    1,
					SortBy:  "date_uploaded",
					OrderBy: "desc",
				})
			case "top_rated":
				movies, _, _ = h.db.ListMovies(database.MovieFilter{
					Limit:         s.LimitCount,
					Page:          1,
					MinimumRating: s.MinimumRating,
					SortBy:        "rating",
					OrderBy:       "desc",
				})
			case "genre":
				movies, _, _ = h.db.ListMovies(database.MovieFilter{
					Limit:         s.LimitCount,
					Page:          1,
					Genre:         s.Genre,
					MinimumRating: s.MinimumRating,
					SortBy:        s.SortBy,
					OrderBy:       s.OrderBy,
				})
			case "curated_list":
				if s.CuratedListID != nil {
					list, err := h.db.GetCuratedListByID(*s.CuratedListID)
					if err == nil {
						movies, _ = h.db.GetCuratedListMovies(list)
					}
				}
			case "query":
				movies, _, _ = h.db.ListMovies(database.MovieFilter{
					Limit:         s.LimitCount,
					Page:          1,
					MinimumRating: s.MinimumRating,
					Genre:         s.Genre,
					SortBy:        s.SortBy,
					OrderBy:       s.OrderBy,
				})
			}

			if len(movies) > 0 {
				sections = append(sections, HomeSectionResponse{
					ID:     s.SectionID,
					Title:  s.Title,
					Type:   s.SectionType,
					Movies: movies,
				})
			}
		}
	}

	writeSuccess(w, map[string]interface{}{
		"sections": sections,
	})
}

// getDefaultHomeSections returns fallback sections when none are configured
func (h *APIHandler) getDefaultHomeSections() []HomeSectionResponse {
	var sections []HomeSectionResponse

	// Recently Added
	recentMovies, _, _ := h.db.ListMovies(database.MovieFilter{
		Limit: 10, Page: 1, SortBy: "date_uploaded", OrderBy: "desc",
	})
	if len(recentMovies) > 0 {
		sections = append(sections, HomeSectionResponse{ID: "recently_added", Title: "Recently Added", Type: "recent", Movies: recentMovies})
	}

	// Top Rated
	topRated, _, _ := h.db.ListMovies(database.MovieFilter{
		Limit: 10, Page: 1, MinimumRating: 7.0, SortBy: "rating", OrderBy: "desc",
	})
	if len(topRated) > 0 {
		sections = append(sections, HomeSectionResponse{ID: "top_rated", Title: "Top Rated", Type: "top_rated", Movies: topRated})
	}

	// Curated lists
	curatedLists, _ := h.db.ListCuratedLists(false)
	for _, list := range curatedLists {
		movies, _ := h.db.GetCuratedListMovies(&list)
		if len(movies) > 0 {
			sections = append(sections, HomeSectionResponse{ID: "curated_" + list.Slug, Title: list.Name, Type: "curated_list", Movies: movies})
		}
	}

	return sections
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
