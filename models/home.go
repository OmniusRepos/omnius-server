package models

import "time"

type HomeSection struct {
	ID          uint   `json:"id" gorm:"primaryKey"`
	SectionID   string `json:"section_id" gorm:"uniqueIndex;not null"`
	Title       string `json:"title" gorm:"not null"`
	DisplayType string `json:"display_type" gorm:"not null;default:'carousel'"` // hero, carousel, grid, top10, banner

	// For hero/banner - single content item
	ContentType string `json:"content_type,omitempty"`
	ContentID   *uint  `json:"content_id,omitempty"`

	// For carousel/grid - query-based content
	SectionType   string  `json:"section_type,omitempty" gorm:"default:'query'"`
	Genre         string  `json:"genre,omitempty"`
	CuratedListID *uint   `json:"curated_list_id,omitempty"`
	SortBy        string  `json:"sort_by" gorm:"default:'rating'"`
	OrderBy       string  `json:"order_by" gorm:"default:'desc'"`
	MinimumRating float32 `json:"minimum_rating" gorm:"default:0"`
	LimitCount    int     `json:"limit_count" gorm:"default:10"`

	IsActive     bool      `json:"is_active" gorm:"default:true"`
	DisplayOrder int       `json:"display_order" gorm:"index;default:0"`
	CreatedAt    time.Time `json:"-" gorm:"autoCreateTime"`
}

func (HomeSection) TableName() string { return "home_sections" }
