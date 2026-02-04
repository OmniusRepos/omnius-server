package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"torrent-server/database"
	"torrent-server/models"
)

type SeriesHandler struct {
	db *database.DB
}

func NewSeriesHandler(db *database.DB) *SeriesHandler {
	return &SeriesHandler{db: db}
}

// ListSeries handles GET /api/v2/list_series.json
func (h *SeriesHandler) ListSeries(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	limit := parseInt(q.Get("limit"), 20)
	page := parseInt(q.Get("page"), 1)

	series, totalCount, err := h.db.ListSeries(limit, page)
	if err != nil {
		writeError(w, "Failed to fetch series: "+err.Error())
		return
	}

	if series == nil {
		series = []models.Series{}
	}

	data := map[string]interface{}{
		"series_count": totalCount,
		"limit":        limit,
		"page_number":  page,
		"series":       series,
	}

	writeSuccess(w, data)
}

// SeriesDetails handles GET /api/v2/series_details.json
func (h *SeriesHandler) SeriesDetails(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	seriesID := parseInt(q.Get("series_id"), 0)
	if seriesID == 0 {
		writeError(w, "series_id is required")
		return
	}

	series, err := h.db.GetSeries(uint(seriesID))
	if err != nil {
		writeError(w, "Series not found")
		return
	}

	// Get episodes if requested
	withEpisodes := q.Get("with_episodes") == "true"
	season := parseInt(q.Get("season"), 0)

	var episodes []models.Episode
	if withEpisodes {
		episodes, _ = h.db.GetEpisodes(uint(seriesID), season)
	}

	// Get season packs
	seasonPacks, _ := h.db.GetSeasonPacks(uint(seriesID))

	data := map[string]interface{}{
		"series":       series,
		"episodes":     episodes,
		"season_packs": seasonPacks,
	}

	writeSuccess(w, data)
}

// AddSeries handles POST /admin/series
func (h *SeriesHandler) AddSeries(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	year, _ := strconv.Atoi(r.FormValue("year"))
	rating, _ := strconv.ParseFloat(r.FormValue("rating"), 32)
	totalSeasons, _ := strconv.Atoi(r.FormValue("total_seasons"))

	series := &models.Series{
		ImdbCode:        r.FormValue("imdb_code"),
		Title:           r.FormValue("title"),
		TitleSlug:       strings.ToLower(strings.ReplaceAll(r.FormValue("title"), " ", "-")),
		Year:            uint(year),
		Rating:          float32(rating),
		Genres:          strings.Split(r.FormValue("genres"), ","),
		Summary:         r.FormValue("summary"),
		PosterImage:     r.FormValue("poster_image"),
		BackgroundImage: r.FormValue("background_image"),
		TotalSeasons:    uint(totalSeasons),
		Status:          r.FormValue("status"),
	}

	if series.Status == "" {
		series.Status = "ongoing"
	}

	if err := h.db.CreateSeries(series); err != nil {
		http.Error(w, "Failed to create series: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if r.Header.Get("Accept") == "application/json" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(series)
		return
	}

	http.Redirect(w, r, "/admin", http.StatusSeeOther)
}

// AddEpisode handles POST /admin/series/{id}/episode
func (h *SeriesHandler) AddEpisode(w http.ResponseWriter, r *http.Request) {
	seriesID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "Invalid series ID", http.StatusBadRequest)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	season, _ := strconv.Atoi(r.FormValue("season"))
	episodeNum, _ := strconv.Atoi(r.FormValue("episode"))

	episode := &models.Episode{
		SeriesID: uint(seriesID),
		Season:   uint(season),
		Episode:  uint(episodeNum),
		Title:    r.FormValue("title"),
		Overview: r.FormValue("overview"),
		AirDate:  r.FormValue("air_date"),
		ImdbCode: r.FormValue("imdb_code"),
	}

	if err := h.db.CreateEpisode(episode); err != nil {
		http.Error(w, "Failed to create episode: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if r.Header.Get("Accept") == "application/json" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(episode)
		return
	}

	http.Redirect(w, r, "/admin", http.StatusSeeOther)
}

// AddEpisodeTorrent handles POST /admin/episodes/{id}/torrent
func (h *SeriesHandler) AddEpisodeTorrent(w http.ResponseWriter, r *http.Request) {
	episodeID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "Invalid episode ID", http.StatusBadRequest)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	hash := r.FormValue("hash")
	magnetLink := r.FormValue("magnet")

	if hash == "" && magnetLink != "" {
		hash = extractHashFromMagnet(magnetLink)
	}

	if hash == "" {
		http.Error(w, "Torrent hash is required", http.StatusBadRequest)
		return
	}

	sizeBytes, _ := strconv.ParseUint(r.FormValue("size_bytes"), 10, 64)
	seeds, _ := strconv.Atoi(r.FormValue("seeds"))
	peers, _ := strconv.Atoi(r.FormValue("peers"))

	torrent := &models.EpisodeTorrent{
		EpisodeID: uint(episodeID),
		Hash:      strings.ToUpper(hash),
		Quality:   r.FormValue("quality"),
		Seeds:     uint(seeds),
		Peers:     uint(peers),
		Size:      r.FormValue("size"),
		SizeBytes: sizeBytes,
		Source:    r.FormValue("source"),
	}

	if torrent.Quality == "" {
		torrent.Quality = "1080p"
	}

	if err := h.db.CreateEpisodeTorrent(torrent); err != nil {
		http.Error(w, "Failed to add torrent: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if r.Header.Get("Accept") == "application/json" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(torrent)
		return
	}

	http.Redirect(w, r, "/admin", http.StatusSeeOther)
}
