package handlers

import (
	"encoding/json"
	"fmt"
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

// SeasonEpisodes handles GET /api/v2/season_episodes.json
func (h *SeriesHandler) SeasonEpisodes(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	seriesID := parseInt(q.Get("series_id"), 0)
	if seriesID == 0 {
		writeError(w, "series_id is required")
		return
	}

	season := parseInt(q.Get("season"), 0)
	if season == 0 {
		writeError(w, "season is required")
		return
	}

	episodes, err := h.db.GetEpisodes(uint(seriesID), season)
	if err != nil {
		writeError(w, "Failed to fetch episodes: "+err.Error())
		return
	}

	if episodes == nil {
		episodes = []models.Episode{}
	}

	writeSuccess(w, episodes)
}

// AddSeries handles POST /admin/series
func (h *SeriesHandler) AddSeries(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	year, _ := strconv.Atoi(r.FormValue("year"))
	rating, _ := strconv.ParseFloat(r.FormValue("rating"), 32)
	runtime, _ := strconv.Atoi(r.FormValue("runtime"))
	totalSeasons, _ := strconv.Atoi(r.FormValue("total_seasons"))

	// Parse genres - trim whitespace from each
	var genres []string
	for _, g := range strings.Split(r.FormValue("genres"), ",") {
		if trimmed := strings.TrimSpace(g); trimmed != "" {
			genres = append(genres, trimmed)
		}
	}

	series := &models.Series{
		ImdbCode:        r.FormValue("imdb_code"),
		Title:           r.FormValue("title"),
		TitleSlug:       strings.ToLower(strings.ReplaceAll(r.FormValue("title"), " ", "-")),
		Year:            uint(year),
		Rating:          float32(rating),
		Runtime:         uint(runtime),
		Genres:          genres,
		Summary:         r.FormValue("summary"),
		PosterImage:     r.FormValue("poster_image"),
		BackgroundImage: r.FormValue("background_image"),
		TotalSeasons:    uint(totalSeasons),
		Status:          r.FormValue("status"),
		Network:         r.FormValue("network"),
	}

	if series.Status == "" {
		series.Status = "Continuing"
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
		SeriesID:      uint(seriesID),
		SeasonNumber:  uint(season),
		EpisodeNumber: uint(episodeNum),
		Title:         r.FormValue("title"),
		Summary:       r.FormValue("summary"),
		AirDate:       r.FormValue("air_date"),
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
	seriesID, _ := strconv.Atoi(r.FormValue("series_id"))
	seasonNumber, _ := strconv.Atoi(r.FormValue("season_number"))
	episodeNumber, _ := strconv.Atoi(r.FormValue("episode_number"))
	fileIndex, _ := strconv.Atoi(r.FormValue("file_index"))

	torrent := &models.EpisodeTorrent{
		EpisodeID:     uint(episodeID),
		SeriesID:      uint(seriesID),
		SeasonNumber:  uint(seasonNumber),
		EpisodeNumber: uint(episodeNumber),
		Hash:          strings.ToUpper(hash),
		Quality:       r.FormValue("quality"),
		VideoCodec:    r.FormValue("video_codec"),
		Seeds:         uint(seeds),
		Peers:         uint(peers),
		Size:          r.FormValue("size"),
		SizeBytes:     sizeBytes,
		ReleaseGroup:  r.FormValue("release_group"),
		FileIndex:     fileIndex,
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

// DeleteSeries handles DELETE /admin/api/series/{id}
func (h *SeriesHandler) DeleteSeries(w http.ResponseWriter, r *http.Request) {
	seriesID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "Invalid series ID", http.StatusBadRequest)
		return
	}

	if err := h.db.DeleteSeries(uint(seriesID)); err != nil {
		http.Error(w, "Failed to delete series: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

// UpdateSeries handles PUT /admin/api/series/{id}
func (h *SeriesHandler) UpdateSeries(w http.ResponseWriter, r *http.Request) {
	seriesID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "Invalid series ID", http.StatusBadRequest)
		return
	}

	var updates struct {
		ImdbCode        string   `json:"imdb_code"`
		Title           string   `json:"title"`
		Year            uint     `json:"year"`
		Rating          float32  `json:"rating"`
		Runtime         uint     `json:"runtime"`
		Genres          string   `json:"genres"`
		Summary         string   `json:"summary"`
		PosterImage     string   `json:"poster_image"`
		BackgroundImage string   `json:"background_image"`
		TotalSeasons    uint     `json:"total_seasons"`
		Status          string   `json:"status"`
		Network         string   `json:"network"`
	}

	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Parse genres from comma-separated string
	var genres []string
	if updates.Genres != "" {
		for _, g := range strings.Split(updates.Genres, ",") {
			genres = append(genres, strings.TrimSpace(g))
		}
	}

	series := &models.Series{
		ID:              uint(seriesID),
		ImdbCode:        updates.ImdbCode,
		Title:           updates.Title,
		TitleSlug:       strings.ToLower(strings.ReplaceAll(updates.Title, " ", "-")),
		Year:            updates.Year,
		Rating:          updates.Rating,
		Runtime:         updates.Runtime,
		Genres:          genres,
		Summary:         updates.Summary,
		PosterImage:     updates.PosterImage,
		BackgroundImage: updates.BackgroundImage,
		TotalSeasons:    updates.TotalSeasons,
		Status:          updates.Status,
		Network:         updates.Network,
	}

	if err := h.db.UpdateSeries(series); err != nil {
		http.Error(w, "Failed to update series: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(series)
}

// ExpandSeasonPack handles POST /admin/api/season-packs/{id}/expand
// Creates episode torrents for each episode in the season from the season pack
func (h *SeriesHandler) ExpandSeasonPack(w http.ResponseWriter, r *http.Request) {
	packID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "Invalid season pack ID", http.StatusBadRequest)
		return
	}

	// Get the season pack
	pack, err := h.db.GetSeasonPack(uint(packID))
	if err != nil {
		http.Error(w, "Season pack not found", http.StatusNotFound)
		return
	}

	// Get all episodes for this season
	episodes, err := h.db.GetEpisodes(pack.SeriesID, int(pack.Season))
	if err != nil || len(episodes) == 0 {
		http.Error(w, "No episodes found for this season", http.StatusNotFound)
		return
	}

	// Calculate per-episode size (approximate)
	perEpisodeSize := pack.SizeBytes / uint64(len(episodes))
	perEpisodeSizeStr := formatSize(perEpisodeSize)

	// Create episode torrents for each episode
	created := 0
	for i, ep := range episodes {
		// Check if this episode already has a torrent with this hash
		existingTorrents, _ := h.db.GetEpisodeTorrents(ep.ID)
		hasHash := false
		for _, t := range existingTorrents {
			if strings.EqualFold(t.Hash, pack.Hash) {
				hasHash = true
				break
			}
		}
		if hasHash {
			continue
		}

		torrent := &models.EpisodeTorrent{
			EpisodeID:     ep.ID,
			SeriesID:      pack.SeriesID,
			SeasonNumber:  uint(pack.Season),
			EpisodeNumber: ep.EpisodeNumber,
			Hash:          pack.Hash,
			Quality:       pack.Quality,
			Seeds:         pack.Seeds,
			Peers:         pack.Peers,
			Size:          perEpisodeSizeStr,
			SizeBytes:     perEpisodeSize,
			FileIndex:     i, // File index based on episode order (0-indexed)
		}

		if err := h.db.CreateEpisodeTorrent(torrent); err == nil {
			created++
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":  true,
		"created":  created,
		"episodes": len(episodes),
	})
}

func formatSize(bytes uint64) string {
	const unit = 1024
	if bytes < unit {
		return "0 B"
	}
	div, exp := uint64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return strings.TrimSpace(strings.Replace(
		strings.Replace(
			strings.Replace(
				strings.Replace(
					fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp]),
					".0 ", " ", 1),
				"  ", " ", 1),
			"  ", " ", 1),
		"  ", " ", 1))
}
