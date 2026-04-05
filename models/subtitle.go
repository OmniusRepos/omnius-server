package models

import "time"

type StoredSubtitle struct {
	ID              uint      `json:"id" gorm:"primaryKey"`
	ImdbCode        string    `json:"imdb_code" gorm:"index;not null"`
	Language        string    `json:"language" gorm:"index;not null"`
	LanguageName    string    `json:"language_name"`
	ReleaseName     string    `json:"release_name"`
	HearingImpaired bool      `json:"hearing_impaired" gorm:"default:false"`
	Source          string    `json:"source"`
	VTTContent      string    `json:"-" gorm:"column:vtt_content"`
	VTTPath         string    `json:"-" gorm:"column:vtt_path;default:''"`
	SeasonNumber    int       `json:"season_number,omitempty" gorm:"default:0"`
	EpisodeNumber   int       `json:"episode_number,omitempty" gorm:"default:0"`
	CreatedAt       time.Time `json:"created_at,omitempty" gorm:"autoCreateTime"`
}

func (StoredSubtitle) TableName() string { return "subtitles" }
