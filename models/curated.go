package models

import "time"

type CuratedList struct {
	ID            uint      `json:"id" gorm:"primaryKey"`
	Name          string    `json:"name" gorm:"not null"`
	Slug          string    `json:"slug" gorm:"uniqueIndex;not null"`
	Description   string    `json:"description,omitempty"`
	SortBy        string    `json:"sort_by" gorm:"default:'rating'"`
	OrderBy       string    `json:"order_by" gorm:"default:'desc'"`
	MinimumRating float32   `json:"minimum_rating,omitempty" gorm:"default:0"`
	MaximumRating float32   `json:"maximum_rating,omitempty" gorm:"default:10"`
	MinimumYear   int       `json:"minimum_year,omitempty"`
	MaximumYear   int       `json:"maximum_year,omitempty"`
	Genre         string    `json:"genre,omitempty"`
	LimitCount    int       `json:"limit" gorm:"column:limit_count;default:50"`
	IsActive      bool      `json:"is_active" gorm:"default:true"`
	DisplayOrder  int       `json:"display_order" gorm:"default:0"`
	CreatedAt     time.Time `json:"created_at,omitempty" gorm:"autoCreateTime"`

	// Relationships
	Movies []Movie `json:"movies,omitempty" gorm:"many2many:curated_list_movies;joinForeignKey:ListID;joinReferences:MovieID"`
}

func (CuratedList) TableName() string { return "curated_lists" }

type CuratedListMovie struct {
	ListID       uint `json:"list_id" gorm:"primaryKey"`
	MovieID      uint `json:"movie_id" gorm:"primaryKey"`
	DisplayOrder int  `json:"display_order" gorm:"default:0"`
}

func (CuratedListMovie) TableName() string { return "curated_list_movies" }

// API response wrappers

type CuratedListData struct {
	Lists []CuratedList `json:"lists"`
}

type CuratedListDetailsData struct {
	List CuratedList `json:"list"`
}
