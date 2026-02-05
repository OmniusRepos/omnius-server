package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"torrent-server/database"
	"torrent-server/services"
)

type AnalyticsHandler struct {
	db             *database.DB
	torrentService *services.TorrentService
}

func NewAnalyticsHandler(db *database.DB, torrentService *services.TorrentService) *AnalyticsHandler {
	return &AnalyticsHandler{db: db, torrentService: torrentService}
}

type StreamStats struct {
	TotalStreams  int    `json:"total_streams"`
	ActiveStreams int    `json:"active_streams"`
	PeakToday     int    `json:"peak_today"`
	PeakTime      string `json:"peak_time"`
	AvgDuration   string `json:"avg_duration"`
	TotalChange   float64 `json:"total_change"`
}

type BandwidthStats struct {
	TotalToday   string  `json:"total_today"`
	AvgPerStream string  `json:"avg_per_stream"`
	PeakRate     string  `json:"peak_rate"`
	TotalMonth   string  `json:"total_month"`
	TodayChange  float64 `json:"today_change"`
}

type UserStats struct {
	UniqueToday   int     `json:"unique_today"`
	UniqueWeek    int     `json:"unique_week"`
	UniqueMonth   int     `json:"unique_month"`
	ReturningRate string  `json:"returning_rate"`
	TodayChange   float64 `json:"today_change"`
}

type TopItem struct {
	ID     uint   `json:"id"`
	Name   string `json:"name"`
	Count  int    `json:"count"`
	Change float64 `json:"change"`
	Image  string `json:"image,omitempty"`
}

type QualityItem struct {
	Quality    string `json:"quality"`
	Percentage int    `json:"percentage"`
}

type AnalyticsResponse struct {
	StreamStats         StreamStats   `json:"stream_stats"`
	BandwidthStats      BandwidthStats `json:"bandwidth_stats"`
	UserStats           UserStats     `json:"user_stats"`
	TopMovies           []TopItem     `json:"top_movies"`
	TopGenres           []TopItem     `json:"top_genres"`
	QualityDistribution []QualityItem `json:"quality_distribution"`
	HourlyActivity      []int         `json:"hourly_activity"`
}

// GetAnalytics returns all analytics data
func (h *AnalyticsHandler) GetAnalytics(w http.ResponseWriter, r *http.Request) {
	period := r.URL.Query().Get("period")
	if period == "" {
		period = "week"
	}

	days := 7
	switch period {
	case "today":
		days = 1
	case "week":
		days = 7
	case "month":
		days = 30
	case "year":
		days = 365
	}

	response := AnalyticsResponse{}

	// Get streaming stats
	response.StreamStats = h.getStreamStats(days)

	// Get bandwidth stats
	response.BandwidthStats = h.getBandwidthStats(days)

	// Get user stats
	response.UserStats = h.getUserStats()

	// Get top movies
	response.TopMovies = h.getTopMovies(days)

	// Get top genres
	response.TopGenres = h.getTopGenres(days)

	// Get quality distribution
	response.QualityDistribution = h.getQualityDistribution(days)

	// Get hourly activity
	response.HourlyActivity = h.getHourlyActivity()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (h *AnalyticsHandler) getStreamStats(days int) StreamStats {
	startDate := time.Now().AddDate(0, 0, -days).Format("2006-01-02")
	prevStartDate := time.Now().AddDate(0, 0, -days*2).Format("2006-01-02")

	var totalViews, prevViews int
	h.db.QueryRow(`SELECT COALESCE(SUM(view_count), 0) FROM content_stats_daily WHERE stat_date >= ?`, startDate).Scan(&totalViews)
	h.db.QueryRow(`SELECT COALESCE(SUM(view_count), 0) FROM content_stats_daily WHERE stat_date >= ? AND stat_date < ?`, prevStartDate, startDate).Scan(&prevViews)

	var totalWatchTime int
	h.db.QueryRow(`SELECT COALESCE(SUM(total_watch_time), 0) FROM content_stats_daily WHERE stat_date >= ?`, startDate).Scan(&totalWatchTime)

	avgMinutes := 0
	if totalViews > 0 {
		avgMinutes = (totalWatchTime / totalViews) / 60
	}

	change := 0.0
	if prevViews > 0 {
		change = float64(totalViews-prevViews) / float64(prevViews) * 100
	}

	// Get peak from hourly data (simplified - just using daily for now)
	var peakCount int
	h.db.QueryRow(`SELECT COALESCE(MAX(view_count), 0) FROM content_stats_daily WHERE stat_date = ?`, time.Now().Format("2006-01-02")).Scan(&peakCount)

	return StreamStats{
		TotalStreams:  totalViews,
		ActiveStreams: h.getActiveStreams(),
		PeakToday:     peakCount,
		PeakTime:      "8:30 PM",
		AvgDuration:   formatDuration(avgMinutes),
		TotalChange:   change,
	}
}

func (h *AnalyticsHandler) getActiveStreams() int {
	// Clean up stale streams first
	h.db.CleanupStaleStreams()
	// Get count from database (heartbeat-based tracking)
	return h.db.GetActiveStreamCount()
}

func (h *AnalyticsHandler) getBandwidthStats(days int) BandwidthStats {
	// Estimate bandwidth based on watch time and average bitrate
	startDate := time.Now().AddDate(0, 0, -days).Format("2006-01-02")
	monthStart := time.Now().AddDate(0, 0, -30).Format("2006-01-02")

	var todayWatchTime, monthWatchTime int
	h.db.QueryRow(`SELECT COALESCE(SUM(total_watch_time), 0) FROM content_stats_daily WHERE stat_date = ?`, time.Now().Format("2006-01-02")).Scan(&todayWatchTime)
	h.db.QueryRow(`SELECT COALESCE(SUM(total_watch_time), 0) FROM content_stats_daily WHERE stat_date >= ?`, monthStart).Scan(&monthWatchTime)

	var totalViews int
	h.db.QueryRow(`SELECT COALESCE(SUM(view_count), 0) FROM content_stats_daily WHERE stat_date >= ?`, startDate).Scan(&totalViews)

	// Assume average bitrate of 5 Mbps
	avgBitrate := 5.0 // Mbps
	todayGB := float64(todayWatchTime) * avgBitrate / 8 / 1024 // seconds * Mbps / 8 / 1024 = GB
	monthTB := float64(monthWatchTime) * avgBitrate / 8 / 1024 / 1024

	avgPerStream := 0.0
	if totalViews > 0 {
		avgPerStream = todayGB / float64(totalViews)
	}

	return BandwidthStats{
		TotalToday:   formatBytes(todayGB * 1024 * 1024 * 1024),
		AvgPerStream: formatBytes(avgPerStream * 1024 * 1024 * 1024),
		PeakRate:     "N/A",
		TotalMonth:   formatBytes(monthTB * 1024 * 1024 * 1024 * 1024),
		TodayChange:  0,
	}
}

func (h *AnalyticsHandler) getUserStats() UserStats {
	today := time.Now().Format("2006-01-02")
	weekStart := time.Now().AddDate(0, 0, -7).Format("2006-01-02")
	monthStart := time.Now().AddDate(0, 0, -30).Format("2006-01-02")

	var todayUsers, weekUsers, monthUsers int
	h.db.QueryRow(`SELECT COUNT(DISTINCT device_id) FROM content_views WHERE view_date = ?`, today).Scan(&todayUsers)
	h.db.QueryRow(`SELECT COUNT(DISTINCT device_id) FROM content_views WHERE view_date >= ?`, weekStart).Scan(&weekUsers)
	h.db.QueryRow(`SELECT COUNT(DISTINCT device_id) FROM content_views WHERE view_date >= ?`, monthStart).Scan(&monthUsers)

	// Calculate returning users (users who viewed on multiple days)
	var returningUsers int
	h.db.QueryRow(`
		SELECT COUNT(*) FROM (
			SELECT device_id FROM content_views
			WHERE view_date >= ?
			GROUP BY device_id
			HAVING COUNT(DISTINCT view_date) > 1
		)`, monthStart).Scan(&returningUsers)

	returningRate := 0
	if monthUsers > 0 {
		returningRate = returningUsers * 100 / monthUsers
	}

	return UserStats{
		UniqueToday:   todayUsers,
		UniqueWeek:    weekUsers,
		UniqueMonth:   monthUsers,
		ReturningRate: formatPercent(returningRate),
		TodayChange:   0,
	}
}

func (h *AnalyticsHandler) getTopMovies(days int) []TopItem {
	startDate := time.Now().AddDate(0, 0, -days).Format("2006-01-02")
	prevStartDate := time.Now().AddDate(0, 0, -days*2).Format("2006-01-02")

	rows, err := h.db.Query(`
		SELECT m.id, m.title, COALESCE(m.medium_cover_image, ''),
			COALESCE(SUM(s.view_count), 0) as views
		FROM movies m
		LEFT JOIN content_stats_daily s ON s.content_type = 'movie' AND s.content_id = m.id AND s.stat_date >= ?
		GROUP BY m.id
		HAVING views > 0
		ORDER BY views DESC
		LIMIT 10
	`, startDate)
	if err != nil {
		return []TopItem{}
	}
	defer rows.Close()

	var items []TopItem
	for rows.Next() {
		var item TopItem
		rows.Scan(&item.ID, &item.Name, &item.Image, &item.Count)

		// Get previous period count for change calculation
		var prevCount int
		h.db.QueryRow(`
			SELECT COALESCE(SUM(view_count), 0) FROM content_stats_daily
			WHERE content_type = 'movie' AND content_id = ? AND stat_date >= ? AND stat_date < ?
		`, item.ID, prevStartDate, startDate).Scan(&prevCount)

		if prevCount > 0 {
			item.Change = float64(item.Count-prevCount) / float64(prevCount) * 100
		}

		items = append(items, item)
	}

	return items
}

func (h *AnalyticsHandler) getTopGenres(days int) []TopItem {
	startDate := time.Now().AddDate(0, 0, -days).Format("2006-01-02")

	// Get genre counts by joining movies with their views
	rows, err := h.db.Query(`
		SELECT m.genres, COALESCE(SUM(s.view_count), 0) as views
		FROM movies m
		JOIN content_stats_daily s ON s.content_type = 'movie' AND s.content_id = m.id AND s.stat_date >= ?
		WHERE m.genres IS NOT NULL AND m.genres != '' AND m.genres != '[]'
		GROUP BY m.genres
		ORDER BY views DESC
	`, startDate)
	if err != nil {
		return []TopItem{}
	}
	defer rows.Close()

	genreCounts := make(map[string]int)
	for rows.Next() {
		var genresJSON string
		var views int
		rows.Scan(&genresJSON, &views)

		var genres []string
		json.Unmarshal([]byte(genresJSON), &genres)
		for _, g := range genres {
			genreCounts[g] += views
		}
	}

	// Sort and take top 10
	var items []TopItem
	for genre, count := range genreCounts {
		items = append(items, TopItem{Name: genre, Count: count})
	}

	// Simple bubble sort for top 10
	for i := 0; i < len(items)-1; i++ {
		for j := i + 1; j < len(items); j++ {
			if items[j].Count > items[i].Count {
				items[i], items[j] = items[j], items[i]
			}
		}
	}

	if len(items) > 10 {
		items = items[:10]
	}

	return items
}

func (h *AnalyticsHandler) getQualityDistribution(days int) []QualityItem {
	startDate := time.Now().AddDate(0, 0, -days).Format("2006-01-02")
	qualities := []string{"2160p", "1080p", "720p", "480p"}
	var items []QualityItem

	// Get quality from actual views
	var viewTotal int
	h.db.QueryRow(`SELECT COUNT(*) FROM content_views WHERE view_date >= ? AND quality IS NOT NULL AND quality != ''`, startDate).Scan(&viewTotal)

	// If we have quality data from views, use it
	if viewTotal > 0 {
		for _, q := range qualities {
			var count int
			h.db.QueryRow(`SELECT COUNT(*) FROM content_views WHERE view_date >= ? AND quality = ?`, startDate, q).Scan(&count)

			percentage := 0
			if viewTotal > 0 {
				percentage = count * 100 / viewTotal
			}

			label := q
			if q == "2160p" {
				label = "4K"
			} else if q == "480p" {
				label = "SD"
			}

			items = append(items, QualityItem{Quality: label, Percentage: percentage})
		}
		return items
	}

	// Fall back to torrent availability
	var total int
	h.db.QueryRow(`SELECT COUNT(*) FROM torrents`).Scan(&total)

	for _, q := range qualities {
		var count int
		h.db.QueryRow(`SELECT COUNT(*) FROM torrents WHERE quality = ?`, q).Scan(&count)

		percentage := 0
		if total > 0 {
			percentage = count * 100 / total
		}

		label := q
		if q == "2160p" {
			label = "4K"
		} else if q == "480p" {
			label = "SD"
		}

		items = append(items, QualityItem{Quality: label, Percentage: percentage})
	}

	return items
}

func (h *AnalyticsHandler) getHourlyActivity() []int {
	today := time.Now().Format("2006-01-02")

	// For now return hourly breakdown based on total views
	// In production, you'd track view timestamps
	activity := make([]int, 24)

	var totalViews int
	h.db.QueryRow(`SELECT COALESCE(SUM(view_count), 0) FROM content_stats_daily WHERE stat_date = ?`, today).Scan(&totalViews)

	// Simulate hourly distribution (peak evening hours)
	distribution := []float64{
		0.02, 0.01, 0.01, 0.01, 0.01, 0.02, 0.03, 0.04, 0.05, 0.06, 0.06, 0.07,
		0.08, 0.07, 0.06, 0.06, 0.07, 0.09, 0.10, 0.11, 0.09, 0.07, 0.05, 0.03,
	}

	for i, d := range distribution {
		activity[i] = int(float64(totalViews) * d)
	}

	return activity
}

// StreamStart handles POST /api/analytics/stream/start
func (h *AnalyticsHandler) StreamStart(w http.ResponseWriter, r *http.Request) {
	var req struct {
		DeviceID    string `json:"device_id"`
		ContentType string `json:"content_type"`
		ContentID   uint   `json:"content_id"`
		ImdbCode    string `json:"imdb_code"`
		Quality     string `json:"quality"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	if req.DeviceID == "" {
		req.DeviceID = "anonymous"
	}
	if req.ContentType == "" {
		req.ContentType = "movie"
	}

	err := h.db.StartStream(req.DeviceID, req.ContentType, req.ContentID, req.ImdbCode, req.Quality)
	if err != nil {
		http.Error(w, "Failed to start stream tracking", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// StreamHeartbeat handles POST /api/analytics/stream/heartbeat
func (h *AnalyticsHandler) StreamHeartbeat(w http.ResponseWriter, r *http.Request) {
	var req struct {
		DeviceID string `json:"device_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	if req.DeviceID == "" {
		http.Error(w, "device_id required", http.StatusBadRequest)
		return
	}

	h.db.HeartbeatStream(req.DeviceID)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// StreamEnd handles POST /api/analytics/stream/end
func (h *AnalyticsHandler) StreamEnd(w http.ResponseWriter, r *http.Request) {
	var req struct {
		DeviceID string `json:"device_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	if req.DeviceID == "" {
		http.Error(w, "device_id required", http.StatusBadRequest)
		return
	}

	h.db.EndStream(req.DeviceID)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// RecordView handles POST /api/analytics/view
func (h *AnalyticsHandler) RecordView(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ContentType string `json:"content_type"`
		ContentID   uint   `json:"content_id"`
		ImdbCode    string `json:"imdb_code"`
		DeviceID    string `json:"device_id"`
		Duration    int    `json:"duration"`
		Completed   bool   `json:"completed"`
		Quality     string `json:"quality"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	if req.ContentType == "" {
		req.ContentType = "movie"
	}

	if req.DeviceID == "" {
		req.DeviceID = "anonymous"
	}

	err := h.db.RecordView(req.ContentType, req.ContentID, req.ImdbCode, req.DeviceID, req.Duration, req.Completed, req.Quality)
	if err != nil {
		http.Error(w, "Failed to record view", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// GetTopMovies handles GET /api/analytics/top-movies
func (h *AnalyticsHandler) GetTopMoviesAPI(w http.ResponseWriter, r *http.Request) {
	days := parseInt(r.URL.Query().Get("days"), 7)
	genre := r.URL.Query().Get("genre")
	limit := parseInt(r.URL.Query().Get("limit"), 10)

	movies, err := h.db.GetTopMovies(days, genre, limit)
	if err != nil {
		http.Error(w, "Failed to get top movies", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(movies)
}

// Helper functions
func formatDuration(minutes int) string {
	if minutes < 60 {
		return fmt.Sprintf("%d min", minutes)
	}
	hours := minutes / 60
	mins := minutes % 60
	if mins == 0 {
		return fmt.Sprintf("%d hr", hours)
	}
	return fmt.Sprintf("%d hr %d min", hours, mins)
}

func formatBytes(bytes float64) string {
	if bytes < 1024 {
		return fmt.Sprintf("%.0f B", bytes)
	}
	kb := bytes / 1024
	if kb < 1024 {
		return fmt.Sprintf("%.1f KB", kb)
	}
	mb := kb / 1024
	if mb < 1024 {
		return fmt.Sprintf("%.1f MB", mb)
	}
	gb := mb / 1024
	if gb < 1024 {
		return fmt.Sprintf("%.1f GB", gb)
	}
	tb := gb / 1024
	return fmt.Sprintf("%.1f TB", tb)
}

func formatPercent(p int) string {
	return fmt.Sprintf("%d%%", p)
}
