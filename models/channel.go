package models

import "time"

type Channel struct {
	ID         string      `json:"id" gorm:"primaryKey"`
	Name       string      `json:"name" gorm:"index;not null"`
	Country    string      `json:"country,omitempty" gorm:"index"`
	Languages  StringSlice `json:"languages,omitempty" gorm:"type:text"`
	Categories StringSlice `json:"categories,omitempty" gorm:"type:text"`
	Logo       string      `json:"logo,omitempty"`
	StreamURL  string      `json:"stream_url,omitempty"`
	IsNSFW     bool        `json:"is_nsfw,omitempty" gorm:"default:false"`
	Website    string      `json:"website,omitempty"`
	CreatedAt  time.Time   `json:"-" gorm:"autoCreateTime"`
	UpdatedAt  time.Time   `json:"-" gorm:"autoUpdateTime"`
}

type ChannelCountry struct {
	Code         string `json:"code" gorm:"primaryKey"`
	Name         string `json:"name" gorm:"not null"`
	Flag         string `json:"flag,omitempty"`
	ChannelCount int    `json:"channel_count,omitempty" gorm:"-"`
}

func (ChannelCountry) TableName() string { return "channel_countries" }

type ChannelCategory struct {
	ID           string `json:"id" gorm:"primaryKey"`
	Name         string `json:"name" gorm:"not null"`
	ChannelCount int    `json:"channel_count,omitempty" gorm:"-"`
}

func (ChannelCategory) TableName() string { return "channel_categories" }

type ChannelEPG struct {
	ID          uint   `json:"id" gorm:"primaryKey"`
	ChannelID   string `json:"channel_id" gorm:"index;not null"`
	Title       string `json:"title" gorm:"not null"`
	Description string `json:"description,omitempty"`
	StartTime   string `json:"start_time" gorm:"index;not null"`
	EndTime     string `json:"end_time" gorm:"index;not null"`
}

func (ChannelEPG) TableName() string { return "channel_epg" }

type ChannelBlocklist struct {
	ChannelID string    `json:"channel_id" gorm:"primaryKey"`
	Reason    string    `json:"reason" gorm:"default:'dead_stream'"`
	BlockedAt time.Time `json:"blocked_at" gorm:"autoCreateTime"`
}

func (ChannelBlocklist) TableName() string { return "channel_blocklist" }

// API response wrappers

type ChannelListData struct {
	ChannelCount int       `json:"channel_count"`
	Limit        int       `json:"limit"`
	PageNumber   int       `json:"page_number"`
	Channels     []Channel `json:"channels"`
}
