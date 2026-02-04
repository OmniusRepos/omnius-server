package models

type CuratedList struct {
	ID            uint    `json:"id"`
	Name          string  `json:"name"`
	Slug          string  `json:"slug"`
	Description   string  `json:"description,omitempty"`
	SortBy        string  `json:"sort_by"`
	OrderBy       string  `json:"order_by"`
	MinimumRating float32 `json:"minimum_rating,omitempty"`
	MaximumRating float32 `json:"maximum_rating,omitempty"`
	MinimumYear   int     `json:"minimum_year,omitempty"`
	MaximumYear   int     `json:"maximum_year,omitempty"`
	Genre         string  `json:"genre,omitempty"`
	LimitCount    int     `json:"limit"`
	IsActive      bool    `json:"is_active"`
	DisplayOrder  int     `json:"display_order"`
	CreatedAt     string  `json:"created_at,omitempty"`
	// Movies included (for hand-picked lists or API response)
	Movies []Movie `json:"movies,omitempty"`
}

type CuratedListData struct {
	Lists []CuratedList `json:"lists"`
}

type CuratedListDetailsData struct {
	List CuratedList `json:"list"`
}
