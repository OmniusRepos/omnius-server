package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"torrent-server/database"
	"torrent-server/models"
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
	TotalStreams  int     `json:"total_streams"`
	ActiveStreams int     `json:"active_streams"`
	PeakToday    int     `json:"peak_today"`
	PeakTime     string  `json:"peak_time"`
	AvgDuration  string  `json:"avg_duration"`
	TotalChange  float64 `json:"total_change"`
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
	ID     uint    `json:"id"`
	Name   string  `json:"name"`
	Count  int     `json:"count"`
	Change float64 `json:"change"`
	Image  string  `json:"image,omitempty"`
}

type QualityItem struct {
	Quality    string `json:"quality"`
	Percentage int    `json:"percentage"`
}

type AnalyticsResponse struct {
	StreamStats         StreamStats    `json:"stream_stats"`
	BandwidthStats      BandwidthStats `json:"bandwidth_stats"`
	UserStats           UserStats      `json:"user_stats"`
	TopMovies           []TopItem      `json:"top_movies"`
	TopGenres           []TopItem      `json:"top_genres"`
	QualityDistribution []QualityItem  `json:"quality_distribution"`
	HourlyActivity      []int          `json:"hourly_activity"`
}

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

	response := AnalyticsResponse{
		StreamStats:         h.getStreamStats(days),
		BandwidthStats:      h.getBandwidthStats(days),
		UserStats:           h.getUserStats(),
		TopMovies:           h.getTopMovies(days),
		TopGenres:           h.getTopGenres(days),
		QualityDistribution: h.getQualityDistribution(days),
		HourlyActivity:      h.getHourlyActivity(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (h *AnalyticsHandler) getStreamStats(days int) StreamStats {
	startDate := time.Now().AddDate(0, 0, -days).Format("2006-01-02")
	prevStartDate := time.Now().AddDate(0, 0, -days*2).Format("2006-01-02")

	var totalViews, prevViews, totalWatchTime, peakCount int64

	h.db.Model(&models.ContentStatsDaily{}).Where("stat_date >= ?", startDate).
		Select("COALESCE(SUM(view_count), 0)").Scan(&totalViews)
	h.db.Model(&models.ContentStatsDaily{}).Where("stat_date >= ? AND stat_date < ?", prevStartDate, startDate).
		Select("COALESCE(SUM(view_count), 0)").Scan(&prevViews)
	h.db.Model(&models.ContentStatsDaily{}).Where("stat_date >= ?", startDate).
		Select("COALESCE(SUM(total_watch_time), 0)").Scan(&totalWatchTime)
	h.db.Model(&models.ContentStatsDaily{}).Where("stat_date = ?", time.Now().Format("2006-01-02")).
		Select("COALESCE(MAX(view_count), 0)").Scan(&peakCount)

	avgMinutes := 0
	if totalViews > 0 {
		avgMinutes = int(totalWatchTime/totalViews) / 60
	}

	change := 0.0
	if prevViews > 0 {
		change = float64(totalViews-prevViews) / float64(prevViews) * 100
	}

	return StreamStats{
		TotalStreams:  int(totalViews),
		ActiveStreams: h.getActiveStreams(),
		PeakToday:    int(peakCount),
		PeakTime:     "8:30 PM",
		AvgDuration:  formatDuration(avgMinutes),
		TotalChange:  change,
	}
}

func (h *AnalyticsHandler) getActiveStreams() int {
	h.db.CleanupStaleStreams()
	return h.db.GetActiveStreamCount()
}

func (h *AnalyticsHandler) getBandwidthStats(days int) BandwidthStats {
	startDate := time.Now().AddDate(0, 0, -days).Format("2006-01-02")
	monthStart := time.Now().AddDate(0, 0, -30).Format("2006-01-02")

	var todayWatchTime, monthWatchTime, totalViews int64

	h.db.Model(&models.ContentStatsDaily{}).Where("stat_date = ?", time.Now().Format("2006-01-02")).
		Select("COALESCE(SUM(total_watch_time), 0)").Scan(&todayWatchTime)
	h.db.Model(&models.ContentStatsDaily{}).Where("stat_date >= ?", monthStart).
		Select("COALESCE(SUM(total_watch_time), 0)").Scan(&monthWatchTime)
	h.db.Model(&models.ContentStatsDaily{}).Where("stat_date >= ?", startDate).
		Select("COALESCE(SUM(view_count), 0)").Scan(&totalViews)

	avgBitrate := 5.0
	todayGB := float64(todayWatchTime) * avgBitrate / 8 / 1024
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

	var todayUsers, weekUsers, monthUsers int64

	h.db.Model(&models.ContentView{}).Where("view_date = ?", today).Distinct("device_id").Count(&todayUsers)
	h.db.Model(&models.ContentView{}).Where("view_date >= ?", weekStart).Distinct("device_id").Count(&weekUsers)
	h.db.Model(&models.ContentView{}).Where("view_date >= ?", monthStart).Distinct("device_id").Count(&monthUsers)

	var returningUsers int64
	h.db.Raw(`SELECT COUNT(*) FROM (
		SELECT device_id FROM content_views
		WHERE view_date >= ?
		GROUP BY device_id
		HAVING COUNT(DISTINCT view_date) > 1
	) sub`, monthStart).Scan(&returningUsers)

	returningRate := 0
	if monthUsers > 0 {
		returningRate = int(returningUsers) * 100 / int(monthUsers)
	}

	return UserStats{
		UniqueToday:   int(todayUsers),
		UniqueWeek:    int(weekUsers),
		UniqueMonth:   int(monthUsers),
		ReturningRate: formatPercent(returningRate),
		TodayChange:   0,
	}
}

func (h *AnalyticsHandler) getTopMovies(days int) []TopItem {
	startDate := time.Now().AddDate(0, 0, -days).Format("2006-01-02")
	prevStartDate := time.Now().AddDate(0, 0, -days*2).Format("2006-01-02")

	type topResult struct {
		ID    uint
		Title string
		Image string
		Views int
	}

	var results []topResult
	h.db.Raw(`
		SELECT m.id, m.title, COALESCE(m.medium_cover_image, '') as image,
			COALESCE(SUM(s.view_count), 0) as views
		FROM movies m
		LEFT JOIN content_stats_daily s ON s.content_type = 'movie' AND s.content_id = m.id AND s.stat_date >= ?
		GROUP BY m.id, m.title, m.medium_cover_image
		HAVING COALESCE(SUM(s.view_count), 0) > 0
		ORDER BY views DESC
		LIMIT 10
	`, startDate).Scan(&results)

	var items []TopItem
	for _, r := range results {
		item := TopItem{ID: r.ID, Name: r.Title, Image: r.Image, Count: r.Views}

		var prevCount int64
		h.db.Model(&models.ContentStatsDaily{}).
			Where("content_type = 'movie' AND content_id = ? AND stat_date >= ? AND stat_date < ?", r.ID, prevStartDate, startDate).
			Select("COALESCE(SUM(view_count), 0)").Scan(&prevCount)

		if prevCount > 0 {
			item.Change = float64(int64(r.Views)-prevCount) / float64(prevCount) * 100
		}

		items = append(items, item)
	}

	return items
}

func (h *AnalyticsHandler) getTopGenres(days int) []TopItem {
	startDate := time.Now().AddDate(0, 0, -days).Format("2006-01-02")

	type genreResult struct {
		Genres string
		Views  int
	}

	var results []genreResult
	h.db.Raw(`
		SELECT m.genres, COALESCE(SUM(s.view_count), 0) as views
		FROM movies m
		JOIN content_stats_daily s ON s.content_type = 'movie' AND s.content_id = m.id AND s.stat_date >= ?
		WHERE m.genres IS NOT NULL AND m.genres != '' AND m.genres != '[]'
		GROUP BY m.genres
		ORDER BY views DESC
	`, startDate).Scan(&results)

	genreCounts := make(map[string]int)
	for _, r := range results {
		var genres []string
		json.Unmarshal([]byte(r.Genres), &genres)
		for _, g := range genres {
			genreCounts[g] += r.Views
		}
	}

	var items []TopItem
	for genre, count := range genreCounts {
		items = append(items, TopItem{Name: genre, Count: count})
	}

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

	var viewTotal int64
	h.db.Model(&models.ContentView{}).
		Where("view_date >= ? AND quality IS NOT NULL AND quality != ''", startDate).
		Count(&viewTotal)

	if viewTotal > 0 {
		for _, q := range qualities {
			var count int64
			h.db.Model(&models.ContentView{}).
				Where("view_date >= ? AND quality = ?", startDate, q).
				Count(&count)

			percentage := 0
			if viewTotal > 0 {
				percentage = int(count) * 100 / int(viewTotal)
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

	var total int64
	h.db.Model(&models.Torrent{}).Count(&total)

	for _, q := range qualities {
		var count int64
		h.db.Model(&models.Torrent{}).Where("quality = ?", q).Count(&count)

		percentage := 0
		if total > 0 {
			percentage = int(count) * 100 / int(total)
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
	activity := make([]int, 24)

	var totalViews int64
	h.db.Model(&models.ContentStatsDaily{}).Where("stat_date = ?", today).
		Select("COALESCE(SUM(view_count), 0)").Scan(&totalViews)

	distribution := []float64{
		0.02, 0.01, 0.01, 0.01, 0.01, 0.02, 0.03, 0.04, 0.05, 0.06, 0.06, 0.07,
		0.08, 0.07, 0.06, 0.06, 0.07, 0.09, 0.10, 0.11, 0.09, 0.07, 0.05, 0.03,
	}

	for i, d := range distribution {
		activity[i] = int(float64(totalViews) * d)
	}

	return activity
}

func (h *AnalyticsHandler) HandleEvent(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Event       string `json:"event"`
		DeviceID    string `json:"device_id"`
		ContentType string `json:"content_type"`
		ContentID   uint   `json:"content_id"`
		ImdbCode    string `json:"imdb_code"`
		Quality     string `json:"quality"`
		Duration    int    `json:"duration"`
		Completed   bool   `json:"completed"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	if req.Event == "" {
		http.Error(w, "Missing event field", http.StatusBadRequest)
		return
	}

	if req.DeviceID == "" {
		req.DeviceID = "anonymous"
	}
	if req.ContentType == "" {
		req.ContentType = "movie"
	}

	switch req.Event {
	case "view", "stream_start", "stream_heartbeat", "stream_end":
	default:
		http.Error(w, "Unknown event: "+req.Event, http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})

	go func() {
		switch req.Event {
		case "view":
			if err := h.db.RecordView(req.ContentType, req.ContentID, req.ImdbCode, req.DeviceID, req.Duration, req.Completed, req.Quality); err != nil {
				fmt.Printf("[analytics] view error: %v\n", err)
			}
		case "stream_start":
			if err := h.db.StartStream(req.DeviceID, req.ContentType, req.ContentID, req.ImdbCode, req.Quality); err != nil {
				fmt.Printf("[analytics] stream_start error: %v\n", err)
			}
		case "stream_heartbeat":
			h.db.HeartbeatStream(req.DeviceID)
		case "stream_end":
			h.db.EndStream(req.DeviceID)
		}
	}()
}

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
