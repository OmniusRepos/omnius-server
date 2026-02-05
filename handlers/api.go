package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

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
		Status:        q.Get("status"), // "available", "coming_soon", or "" for all
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
	ID          string         `json:"id"`
	Title       string         `json:"title"`
	Type        string         `json:"type"`         // data source: recent, top_rated, genre, curated_list, query
	DisplayType string         `json:"display_type"` // UI layout: carousel, grid, featured, hero, banner, slider
	Movies      []models.Movie `json:"movies,omitempty"`
}

// HeroSliderResponse is returned for the hero slider specifically
type HeroSliderResponse struct {
	Movies []models.Movie `json:"movies"`
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

			// Hero/Banner sections with specific content_id
			if (s.DisplayType == "hero" || s.DisplayType == "banner") && s.ContentID != nil {
				if s.ContentType == "movie" {
					movie, err := h.db.GetMovie(*s.ContentID)
					if err == nil && movie != nil {
						movies = []models.Movie{*movie}
					}
				}
				// TODO: Handle series and channels for hero/banner
			} else {
				// Query-based sections (carousel, grid, featured, top10)
				switch s.SectionType {
				case "top_viewed":
					// Get top movies from analytics (last 7 days)
					movies, _ = h.db.GetTopMovies(7, s.Genre, s.LimitCount)
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
				default:
					// Default query with custom sort/filter
					movies, _, _ = h.db.ListMovies(database.MovieFilter{
						Limit:         s.LimitCount,
						Page:          1,
						MinimumRating: s.MinimumRating,
						Genre:         s.Genre,
						SortBy:        s.SortBy,
						OrderBy:       s.OrderBy,
					})
				}
			}

			if len(movies) > 0 {
				sections = append(sections, HomeSectionResponse{
					ID:          s.SectionID,
					Title:       s.Title,
					Type:        s.SectionType,
					DisplayType: s.DisplayType,
					Movies:      movies,
				})
			}
		}
	}

	// Build hero slider from all hero/banner sections
	var heroSlider []models.Movie
	for _, s := range dbSections {
		if (s.DisplayType == "hero" || s.DisplayType == "banner" || s.DisplayType == "slider") && s.ContentID != nil {
			if s.ContentType == "movie" {
				movie, err := h.db.GetMovie(*s.ContentID)
				if err == nil && movie != nil {
					heroSlider = append(heroSlider, *movie)
				}
			}
		}
	}

	// If no hero movies configured, use top 5 rated
	if len(heroSlider) == 0 {
		topMovies, _, _ := h.db.ListMovies(database.MovieFilter{
			Limit:         5,
			Page:          1,
			MinimumRating: 8.0,
			SortBy:        "rating",
			OrderBy:       "desc",
		})
		heroSlider = topMovies
	}

	writeSuccess(w, map[string]interface{}{
		"sections":    sections,
		"hero_slider": heroSlider,
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
		sections = append(sections, HomeSectionResponse{ID: "recently_added", Title: "Recently Added", Type: "recent", DisplayType: "carousel", Movies: recentMovies})
	}

	// Top Rated
	topRated, _, _ := h.db.ListMovies(database.MovieFilter{
		Limit: 10, Page: 1, MinimumRating: 7.0, SortBy: "rating", OrderBy: "desc",
	})
	if len(topRated) > 0 {
		sections = append(sections, HomeSectionResponse{ID: "top_rated", Title: "Top Rated", Type: "top_rated", DisplayType: "carousel", Movies: topRated})
	}

	// Curated lists
	curatedLists, _ := h.db.ListCuratedLists(false)
	for _, list := range curatedLists {
		movies, _ := h.db.GetCuratedListMovies(&list)
		if len(movies) > 0 {
			sections = append(sections, HomeSectionResponse{ID: "curated_" + list.Slug, Title: list.Name, Type: "curated_list", DisplayType: "carousel", Movies: movies})
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

// FranchiseMovies handles GET /api/v2/franchise_movies.json
// Returns all movies in the same franchise as the given movie
func (h *APIHandler) FranchiseMovies(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	movieID := parseInt(q.Get("movie_id"), 0)
	if movieID == 0 {
		writeError(w, "movie_id is required")
		return
	}

	// Get the movie to find its franchise
	movie, err := h.db.GetMovie(uint(movieID))
	if err != nil {
		writeError(w, "Movie not found")
		return
	}

	if movie.Franchise == "" {
		// No franchise, return empty
		data := models.MovieSuggestionsData{
			MovieCount: 0,
			Movies:     []models.Movie{},
		}
		writeSuccess(w, data)
		return
	}

	movies, err := h.db.GetFranchiseMovies(uint(movieID), movie.Franchise)
	if err != nil {
		movies = []models.Movie{}
	}

	data := models.MovieSuggestionsData{
		MovieCount: len(movies),
		Movies:     movies,
	}

	writeSuccess(w, data)
}

// AvailabilityInfo represents availability status for a single movie
type AvailabilityInfo struct {
	Available bool   `json:"available"`
	Title     string `json:"title,omitempty"`
	ID        uint   `json:"id,omitempty"`
	Poster    string `json:"poster,omitempty"`
}

// CheckAvailability handles GET /api/v2/check_availability
// Returns availability status for multiple IMDB codes
// Used by Streamer app to check if "coming soon" movies are now available
func (h *APIHandler) CheckAvailability(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	// Get comma-separated IMDB codes
	imdbCodesParam := q.Get("imdb_codes")
	if imdbCodesParam == "" {
		writeError(w, "imdb_codes parameter is required")
		return
	}

	imdbCodes := splitAndTrim(imdbCodesParam)
	result := make(map[string]AvailabilityInfo)

	for _, imdbCode := range imdbCodes {
		if imdbCode == "" {
			continue
		}

		movie, err := h.db.GetMovieByIMDB(imdbCode)
		if err != nil || movie == nil {
			// Movie not found in database
			result[imdbCode] = AvailabilityInfo{Available: false}
			continue
		}

		// Movie is available if it has torrents and is not "coming_soon"
		hasContent := len(movie.Torrents) > 0 && movie.Status != "coming_soon"

		result[imdbCode] = AvailabilityInfo{
			Available: hasContent,
			Title:     movie.Title,
			ID:        movie.ID,
			Poster:    movie.MediumCoverImage,
		}
	}

	writeSuccess(w, result)
}

// splitAndTrim splits a comma-separated string and trims whitespace
func splitAndTrim(s string) []string {
	var result []string
	for _, part := range strings.Split(s, ",") {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

// SearchResult represents a unified search result
type SearchResult struct {
	Type string      `json:"type"` // "movie", "series", "channel"
	Data interface{} `json:"data"`
}

// UnifiedSearchResponse is the response for unified search
type UnifiedSearchResponse struct {
	Query    string         `json:"query"`
	Movies   []models.Movie `json:"movies"`
	Series   []models.Series `json:"series"`
	Channels []models.Channel `json:"channels"`
}

// UnifiedSearch handles GET /api/v2/search.json
// Searches across movies, series, and channels
func (h *APIHandler) UnifiedSearch(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	query := q.Get("query")
	if query == "" {
		writeError(w, "query parameter is required")
		return
	}

	limit := parseInt(q.Get("limit"), 10)
	if limit > 50 {
		limit = 50
	}

	// Search movies
	movies, _, _ := h.db.ListMovies(database.MovieFilter{
		Limit:     limit,
		Page:      1,
		QueryTerm: query,
	})
	if movies == nil {
		movies = []models.Movie{}
	}

	// Search series
	series, _, _ := h.db.ListSeries(limit, 1)
	var matchedSeries []models.Series
	queryLower := strings.ToLower(query)
	for _, s := range series {
		if strings.Contains(strings.ToLower(s.Title), queryLower) {
			matchedSeries = append(matchedSeries, s)
			if len(matchedSeries) >= limit {
				break
			}
		}
	}
	if matchedSeries == nil {
		matchedSeries = []models.Series{}
	}

	// Search channels
	channels, _, _ := h.db.ListChannels(database.ChannelFilter{
		Limit:     limit,
		Page:      1,
		QueryTerm: query,
	})
	if channels == nil {
		channels = []models.Channel{}
	}

	response := UnifiedSearchResponse{
		Query:    query,
		Movies:   movies,
		Series:   matchedSeries,
		Channels: channels,
	}

	writeSuccess(w, response)
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
