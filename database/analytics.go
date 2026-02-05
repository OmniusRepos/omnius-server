package database

import (
	"time"

	"torrent-server/models"
)

// RecordView records a content view
func (d *DB) RecordView(contentType string, contentID uint, imdbCode string, deviceID string, duration int, completed bool, quality string) error {
	today := time.Now().Format("2006-01-02")
	completedInt := 0
	if completed {
		completedInt = 1
	}

	// Upsert view record (increment if exists for same device+date)
	_, err := d.Exec(`
		INSERT INTO content_views (content_type, content_id, imdb_code, device_id, view_date, view_count, watch_duration, completed, quality)
		VALUES (?, ?, ?, ?, ?, 1, ?, ?, ?)
		ON CONFLICT(content_type, content_id, device_id, view_date) DO UPDATE SET
			view_count = view_count + 1,
			watch_duration = watch_duration + excluded.watch_duration,
			completed = MAX(completed, excluded.completed),
			quality = COALESCE(excluded.quality, quality)
	`, contentType, contentID, imdbCode, deviceID, today, duration, completedInt, quality)

	if err != nil {
		return err
	}

	// Update daily stats
	_, err = d.Exec(`
		INSERT INTO content_stats_daily (content_type, content_id, stat_date, view_count, unique_viewers, total_watch_time, completions)
		VALUES (?, ?, ?, 1, 1, ?, ?)
		ON CONFLICT(content_type, content_id, stat_date) DO UPDATE SET
			view_count = view_count + 1,
			total_watch_time = total_watch_time + excluded.total_watch_time,
			completions = completions + excluded.completions
	`, contentType, contentID, today, duration, completedInt)

	return err
}

// GetTopMovies returns top movies by view count for a time period
func (d *DB) GetTopMovies(days int, genre string, limit int) ([]models.Movie, error) {
	if limit <= 0 {
		limit = 10
	}

	startDate := time.Now().AddDate(0, 0, -days).Format("2006-01-02")

	query := `
		SELECT m.id, m.imdb_code, m.title, COALESCE(m.title_english, ''), m.title_long, m.slug,
			m.year, m.rating, m.runtime, COALESCE(m.genres, '[]'), m.summary, m.description_full,
			COALESCE(m.synopsis, ''), COALESCE(m.yt_trailer_code, ''), m.language,
			COALESCE(m.background_image, ''), COALESCE(m.small_cover_image, ''),
			COALESCE(m.medium_cover_image, ''), COALESCE(m.large_cover_image, ''),
			COALESCE(m.date_uploaded, ''), COALESCE(m.date_uploaded_unix, 0),
			COALESCE(m.mpa_rating, ''), COALESCE(m.background_image_original, ''),
			m.imdb_rating, m.rotten_tomatoes, m.metacritic, COALESCE(m.imdb_votes, ''),
			COALESCE(SUM(s.view_count), 0) as total_views
		FROM movies m
		LEFT JOIN content_stats_daily s ON s.content_type = 'movie' AND s.content_id = m.id AND s.stat_date >= ?
	`

	args := []interface{}{startDate}

	if genre != "" {
		query += ` WHERE m.genres LIKE ?`
		args = append(args, "%"+genre+"%")
	}

	query += `
		GROUP BY m.id
		HAVING total_views > 0
		ORDER BY total_views DESC
		LIMIT ?
	`
	args = append(args, limit)

	rows, err := d.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var movies []models.Movie
	for rows.Next() {
		var m models.Movie
		var genresJSON string
		var totalViews int

		err := rows.Scan(
			&m.ID, &m.ImdbCode, &m.Title, &m.TitleEnglish, &m.TitleLong, &m.Slug,
			&m.Year, &m.Rating, &m.Runtime, &genresJSON, &m.Summary, &m.DescriptionFull,
			&m.Synopsis, &m.YtTrailerCode, &m.Language,
			&m.BackgroundImage, &m.SmallCoverImage, &m.MediumCoverImage, &m.LargeCoverImage,
			&m.DateUploaded, &m.DateUploadedUnix,
			&m.MpaRating, &m.BackgroundImageOriginal,
			&m.ImdbRating, &m.RottenTomatoes, &m.Metacritic, &m.ImdbVotes,
			&totalViews,
		)
		if err != nil {
			continue
		}

		m.ParseGenres(genresJSON)
		m.DownloadCount = uint(totalViews) // Use download_count to pass view count to frontend

		// Get torrents for this movie
		m.Torrents, _ = d.GetTorrentsForMovie(m.ID)

		movies = append(movies, m)
	}

	return movies, nil
}

// GetTopSeries returns top series by view count for a time period
func (d *DB) GetTopSeries(days int, limit int) ([]models.Series, error) {
	if limit <= 0 {
		limit = 10
	}

	startDate := time.Now().AddDate(0, 0, -days).Format("2006-01-02")

	query := `
		SELECT s.id, s.imdb_code, s.title, COALESCE(s.title_slug, ''), s.year,
			s.rating, COALESCE(s.genres, '[]'), COALESCE(s.summary, ''),
			COALESCE(s.poster_image, ''), COALESCE(s.background_image, ''),
			s.total_seasons, s.status,
			COALESCE(SUM(cs.view_count), 0) as total_views
		FROM series s
		LEFT JOIN content_stats_daily cs ON cs.content_type = 'series' AND cs.content_id = s.id AND cs.stat_date >= ?
		GROUP BY s.id
		HAVING total_views > 0
		ORDER BY total_views DESC
		LIMIT ?
	`

	rows, err := d.Query(query, startDate, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var series []models.Series
	for rows.Next() {
		var s models.Series
		var genresJSON string
		var totalViews int

		err := rows.Scan(
			&s.ID, &s.ImdbCode, &s.Title, &s.TitleSlug, &s.Year,
			&s.Rating, &genresJSON, &s.Summary,
			&s.PosterImage, &s.BackgroundImage,
			&s.TotalSeasons, &s.Status,
			&totalViews,
		)
		if err != nil {
			continue
		}

		s.ParseGenres(genresJSON)
		series = append(series, s)
	}

	return series, nil
}

// StartStream records a new active stream
func (d *DB) StartStream(deviceID string, contentType string, contentID uint, imdbCode string, quality string) error {
	_, err := d.Exec(`
		INSERT INTO active_streams (device_id, content_type, content_id, imdb_code, quality, started_at, last_heartbeat)
		VALUES (?, ?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		ON CONFLICT(device_id) DO UPDATE SET
			content_type = excluded.content_type,
			content_id = excluded.content_id,
			imdb_code = excluded.imdb_code,
			quality = excluded.quality,
			started_at = CURRENT_TIMESTAMP,
			last_heartbeat = CURRENT_TIMESTAMP
	`, deviceID, contentType, contentID, imdbCode, quality)
	return err
}

// HeartbeatStream updates the last heartbeat for an active stream
func (d *DB) HeartbeatStream(deviceID string) error {
	_, err := d.Exec(`UPDATE active_streams SET last_heartbeat = CURRENT_TIMESTAMP WHERE device_id = ?`, deviceID)
	return err
}

// EndStream removes an active stream
func (d *DB) EndStream(deviceID string) error {
	_, err := d.Exec(`DELETE FROM active_streams WHERE device_id = ?`, deviceID)
	return err
}

// GetActiveStreamCount returns count of streams active in the last 2 minutes
func (d *DB) GetActiveStreamCount() int {
	var count int
	d.QueryRow(`SELECT COUNT(*) FROM active_streams WHERE last_heartbeat > datetime('now', '-2 minutes')`).Scan(&count)
	return count
}

// CleanupStaleStreams removes streams that haven't sent a heartbeat in 5 minutes
func (d *DB) CleanupStaleStreams() {
	d.Exec(`DELETE FROM active_streams WHERE last_heartbeat < datetime('now', '-5 minutes')`)
}

// GetViewStats returns aggregate view statistics
func (d *DB) GetViewStats(days int) (map[string]interface{}, error) {
	startDate := time.Now().AddDate(0, 0, -days).Format("2006-01-02")

	stats := make(map[string]interface{})

	// Total views
	var totalViews int
	d.QueryRow(`SELECT COALESCE(SUM(view_count), 0) FROM content_stats_daily WHERE stat_date >= ?`, startDate).Scan(&totalViews)
	stats["total_views"] = totalViews

	// Unique content viewed
	var uniqueContent int
	d.QueryRow(`SELECT COUNT(DISTINCT content_id || '-' || content_type) FROM content_stats_daily WHERE stat_date >= ?`, startDate).Scan(&uniqueContent)
	stats["unique_content"] = uniqueContent

	// Total watch time (hours)
	var totalWatchTime int
	d.QueryRow(`SELECT COALESCE(SUM(total_watch_time), 0) FROM content_stats_daily WHERE stat_date >= ?`, startDate).Scan(&totalWatchTime)
	stats["total_watch_hours"] = totalWatchTime / 3600

	// Completions
	var completions int
	d.QueryRow(`SELECT COALESCE(SUM(completions), 0) FROM content_stats_daily WHERE stat_date >= ?`, startDate).Scan(&completions)
	stats["completions"] = completions

	return stats, nil
}
