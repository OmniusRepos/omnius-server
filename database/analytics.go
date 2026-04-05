package database

import (
	"time"

	"gorm.io/gorm/clause"

	"torrent-server/models"
)

func (d *DB) RecordView(contentType string, contentID uint, imdbCode string, deviceID string, duration int, completed bool, quality string) error {
	today := time.Now().Format("2006-01-02")

	view := models.ContentView{
		ContentType:   contentType,
		ContentID:     contentID,
		ImdbCode:      imdbCode,
		DeviceID:      deviceID,
		ViewDate:      today,
		ViewCount:     1,
		WatchDuration: duration,
		Completed:     completed,
		Quality:       quality,
	}
	d.Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "content_type"}, {Name: "content_id"}, {Name: "device_id"}, {Name: "view_date"},
		},
		DoUpdates: clause.Assignments(map[string]interface{}{
			"view_count":     clause.Expr{SQL: "content_views.view_count + 1"},
			"watch_duration": clause.Expr{SQL: "content_views.watch_duration + ?", Vars: []interface{}{duration}},
			"quality":        quality,
		}),
	}).Create(&view)

	completedInt := 0
	if completed {
		completedInt = 1
	}
	stat := models.ContentStatsDaily{
		ContentType:    contentType,
		ContentID:      contentID,
		StatDate:       today,
		ViewCount:      1,
		UniqueViewers:  1,
		TotalWatchTime: duration,
		Completions:    completedInt,
	}
	d.Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "content_type"}, {Name: "content_id"}, {Name: "stat_date"},
		},
		DoUpdates: clause.Assignments(map[string]interface{}{
			"view_count":       clause.Expr{SQL: "content_stats_daily.view_count + 1"},
			"total_watch_time": clause.Expr{SQL: "content_stats_daily.total_watch_time + ?", Vars: []interface{}{duration}},
			"completions":      clause.Expr{SQL: "content_stats_daily.completions + ?", Vars: []interface{}{completedInt}},
		}),
	}).Create(&stat)

	return nil
}

func (d *DB) GetTopMovies(days int, genre string, limit int) ([]models.Movie, error) {
	if limit <= 0 {
		limit = 10
	}

	startDate := time.Now().AddDate(0, 0, -days).Format("2006-01-02")

	query := d.Model(&models.Movie{}).
		Select("movies.*, COALESCE(SUM(s.view_count), 0) as download_count").
		Joins("LEFT JOIN content_stats_daily s ON s.content_type = 'movie' AND s.content_id = movies.id AND s.stat_date >= ?", startDate)

	if genre != "" {
		query = query.Where("movies.genres LIKE ?", "%"+genre+"%")
	}

	var movies []models.Movie
	err := query.
		Group("movies.id").
		Having("COALESCE(SUM(s.view_count), 0) > 0").
		Order("download_count DESC").
		Limit(limit).
		Find(&movies).Error
	if err != nil {
		return nil, err
	}

	for i := range movies {
		d.DB.Where("movie_id = ?", movies[i].ID).Find(&movies[i].Torrents)
	}

	return movies, nil
}

func (d *DB) GetTopSeries(days int, limit int) ([]models.Series, error) {
	if limit <= 0 {
		limit = 10
	}

	startDate := time.Now().AddDate(0, 0, -days).Format("2006-01-02")

	var series []models.Series
	err := d.Model(&models.Series{}).
		Select("series.*, COALESCE(SUM(cs.view_count), 0) as total_views").
		Joins("LEFT JOIN content_stats_daily cs ON cs.content_type = 'series' AND cs.content_id = series.id AND cs.stat_date >= ?", startDate).
		Group("series.id").
		Having("COALESCE(SUM(cs.view_count), 0) > 0").
		Order("total_views DESC").
		Limit(limit).
		Find(&series).Error

	return series, err
}

func (d *DB) StartStream(deviceID string, contentType string, contentID uint, imdbCode string, quality string) error {
	cid := contentID
	stream := models.ActiveStream{
		DeviceID:    deviceID,
		ContentType: contentType,
		ContentID:   &cid,
		ImdbCode:    imdbCode,
		Quality:     quality,
	}
	return d.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "device_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"content_type", "content_id", "imdb_code", "quality", "started_at", "last_heartbeat"}),
	}).Create(&stream).Error
}

func (d *DB) HeartbeatStream(deviceID string) error {
	return d.Model(&models.ActiveStream{}).
		Where("device_id = ?", deviceID).
		Update("last_heartbeat", time.Now()).Error
}

func (d *DB) EndStream(deviceID string) error {
	return d.Where("device_id = ?", deviceID).Delete(&models.ActiveStream{}).Error
}

func (d *DB) GetActiveStreamCount() int {
	var count int64
	cutoff := time.Now().Add(-2 * time.Minute)
	d.Model(&models.ActiveStream{}).Where("last_heartbeat > ?", cutoff).Count(&count)
	return int(count)
}

func (d *DB) CleanupStaleStreams() {
	cutoff := time.Now().Add(-5 * time.Minute)
	d.Where("last_heartbeat < ?", cutoff).Delete(&models.ActiveStream{})
}

func (d *DB) GetViewStats(days int) (map[string]interface{}, error) {
	startDate := time.Now().AddDate(0, 0, -days).Format("2006-01-02")
	stats := make(map[string]interface{})

	var totalViews int64
	d.Model(&models.ContentStatsDaily{}).Where("stat_date >= ?", startDate).
		Select("COALESCE(SUM(view_count), 0)").Scan(&totalViews)
	stats["total_views"] = totalViews

	var uniqueContent int64
	d.Model(&models.ContentStatsDaily{}).Where("stat_date >= ?", startDate).
		Distinct("content_id", "content_type").Count(&uniqueContent)
	stats["unique_content"] = uniqueContent

	var totalWatchTime int64
	d.Model(&models.ContentStatsDaily{}).Where("stat_date >= ?", startDate).
		Select("COALESCE(SUM(total_watch_time), 0)").Scan(&totalWatchTime)
	stats["total_watch_hours"] = totalWatchTime / 3600

	var completions int64
	d.Model(&models.ContentStatsDaily{}).Where("stat_date >= ?", startDate).
		Select("COALESCE(SUM(completions), 0)").Scan(&completions)
	stats["completions"] = completions

	return stats, nil
}
