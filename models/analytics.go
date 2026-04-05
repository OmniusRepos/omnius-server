package models

import "time"

type ContentView struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	ContentType string    `json:"content_type" gorm:"not null;uniqueIndex:idx_views_unique"`
	ContentID   uint      `json:"content_id" gorm:"not null;uniqueIndex:idx_views_unique;index:idx_views_content"`
	ImdbCode    string    `json:"imdb_code,omitempty"`
	DeviceID    string    `json:"device_id" gorm:"uniqueIndex:idx_views_unique"`
	ViewDate    string    `json:"view_date" gorm:"not null;uniqueIndex:idx_views_unique;index:idx_views_date"`
	ViewCount   int       `json:"view_count" gorm:"default:1"`
	WatchDuration int    `json:"watch_duration" gorm:"default:0"`
	Completed   bool      `json:"completed" gorm:"default:false"`
	Quality     string    `json:"quality,omitempty"`
	CreatedAt   time.Time `json:"-" gorm:"autoCreateTime"`
}

func (ContentView) TableName() string { return "content_views" }

type ContentStatsDaily struct {
	ID             uint   `json:"id" gorm:"primaryKey"`
	ContentType    string `json:"content_type" gorm:"not null;uniqueIndex:idx_stats_unique;index:idx_stats_content"`
	ContentID      uint   `json:"content_id" gorm:"not null;uniqueIndex:idx_stats_unique;index:idx_stats_content"`
	StatDate       string `json:"stat_date" gorm:"not null;uniqueIndex:idx_stats_unique;index:idx_stats_date"`
	ViewCount      int    `json:"view_count" gorm:"default:0"`
	UniqueViewers  int    `json:"unique_viewers" gorm:"default:0"`
	TotalWatchTime int    `json:"total_watch_time" gorm:"default:0"`
	Completions    int    `json:"completions" gorm:"default:0"`
}

func (ContentStatsDaily) TableName() string { return "content_stats_daily" }

type ActiveStream struct {
	ID            uint      `json:"id" gorm:"primaryKey"`
	DeviceID      string    `json:"device_id" gorm:"uniqueIndex;not null"`
	ContentType   string    `json:"content_type" gorm:"not null"`
	ContentID     *uint     `json:"content_id,omitempty"`
	ImdbCode      string    `json:"imdb_code,omitempty"`
	Quality       string    `json:"quality,omitempty"`
	StartedAt     time.Time `json:"started_at" gorm:"autoCreateTime"`
	LastHeartbeat time.Time `json:"last_heartbeat" gorm:"autoUpdateTime"`
}

func (ActiveStream) TableName() string { return "active_streams" }
