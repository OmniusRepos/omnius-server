package database

import (
	"database/sql"
)

type HomeSection struct {
	ID          uint   `json:"id"`
	SectionID   string `json:"section_id"`
	Title       string `json:"title"`
	DisplayType string `json:"display_type"` // hero, carousel, grid, featured, banner

	// For hero/banner - single content item
	ContentType string `json:"content_type,omitempty"` // movie, series, channel
	ContentID   *uint  `json:"content_id,omitempty"`   // ID of the specific item

	// For carousel/grid/featured - query-based content
	SectionType   string  `json:"section_type,omitempty"`   // recent, top_rated, genre, curated_list, query
	Genre         string  `json:"genre,omitempty"`          // for genre sections
	CuratedListID *uint   `json:"curated_list_id,omitempty"`
	SortBy        string  `json:"sort_by"`
	OrderBy       string  `json:"order_by"`
	MinimumRating float32 `json:"minimum_rating"`
	LimitCount    int     `json:"limit_count"`

	IsActive     bool `json:"is_active"`
	DisplayOrder int  `json:"display_order"`
}

func (d *DB) ListHomeSections(includeInactive bool) ([]HomeSection, error) {
	query := `SELECT id, section_id, title, COALESCE(display_type, 'carousel'),
		content_type, content_id, section_type, genre, curated_list_id,
		sort_by, order_by, minimum_rating, limit_count, is_active, display_order
		FROM home_sections`

	if !includeInactive {
		query += " WHERE is_active = 1"
	}
	query += " ORDER BY display_order ASC, id ASC"

	rows, err := d.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sections []HomeSection
	for rows.Next() {
		var s HomeSection
		var contentType, sectionType, genre sql.NullString
		var contentID, curatedListID sql.NullInt64

		err := rows.Scan(
			&s.ID, &s.SectionID, &s.Title, &s.DisplayType,
			&contentType, &contentID, &sectionType, &genre, &curatedListID,
			&s.SortBy, &s.OrderBy, &s.MinimumRating, &s.LimitCount,
			&s.IsActive, &s.DisplayOrder,
		)
		if err != nil {
			return nil, err
		}

		if contentType.Valid {
			s.ContentType = contentType.String
		}
		if contentID.Valid {
			id := uint(contentID.Int64)
			s.ContentID = &id
		}
		if sectionType.Valid {
			s.SectionType = sectionType.String
		}
		if genre.Valid {
			s.Genre = genre.String
		}
		if curatedListID.Valid {
			id := uint(curatedListID.Int64)
			s.CuratedListID = &id
		}

		sections = append(sections, s)
	}

	return sections, nil
}

func (d *DB) GetHomeSection(id uint) (*HomeSection, error) {
	var s HomeSection
	var contentType, sectionType, genre sql.NullString
	var contentID, curatedListID sql.NullInt64

	err := d.QueryRow(`SELECT id, section_id, title, COALESCE(display_type, 'carousel'),
		content_type, content_id, section_type, genre, curated_list_id,
		sort_by, order_by, minimum_rating, limit_count, is_active, display_order
		FROM home_sections WHERE id = ?`, id).Scan(
		&s.ID, &s.SectionID, &s.Title, &s.DisplayType,
		&contentType, &contentID, &sectionType, &genre, &curatedListID,
		&s.SortBy, &s.OrderBy, &s.MinimumRating, &s.LimitCount,
		&s.IsActive, &s.DisplayOrder,
	)
	if err != nil {
		return nil, err
	}

	if contentType.Valid {
		s.ContentType = contentType.String
	}
	if contentID.Valid {
		id := uint(contentID.Int64)
		s.ContentID = &id
	}
	if sectionType.Valid {
		s.SectionType = sectionType.String
	}
	if genre.Valid {
		s.Genre = genre.String
	}
	if curatedListID.Valid {
		id := uint(curatedListID.Int64)
		s.CuratedListID = &id
	}

	return &s, nil
}

func (d *DB) CreateHomeSection(s *HomeSection) error {
	if s.DisplayType == "" {
		s.DisplayType = "carousel"
	}
	result, err := d.Exec(`INSERT INTO home_sections
		(section_id, title, display_type, content_type, content_id, section_type, genre, curated_list_id,
		sort_by, order_by, minimum_rating, limit_count, is_active, display_order)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		s.SectionID, s.Title, s.DisplayType, s.ContentType, s.ContentID, s.SectionType, s.Genre, s.CuratedListID,
		s.SortBy, s.OrderBy, s.MinimumRating, s.LimitCount, s.IsActive, s.DisplayOrder,
	)
	if err != nil {
		return err
	}

	id, _ := result.LastInsertId()
	s.ID = uint(id)
	return nil
}

func (d *DB) UpdateHomeSection(s *HomeSection) error {
	if s.DisplayType == "" {
		s.DisplayType = "carousel"
	}
	_, err := d.Exec(`UPDATE home_sections SET
		section_id = ?, title = ?, display_type = ?, content_type = ?, content_id = ?,
		section_type = ?, genre = ?, curated_list_id = ?, sort_by = ?, order_by = ?,
		minimum_rating = ?, limit_count = ?, is_active = ?, display_order = ?
		WHERE id = ?`,
		s.SectionID, s.Title, s.DisplayType, s.ContentType, s.ContentID,
		s.SectionType, s.Genre, s.CuratedListID, s.SortBy, s.OrderBy,
		s.MinimumRating, s.LimitCount, s.IsActive, s.DisplayOrder, s.ID,
	)
	return err
}

func (d *DB) DeleteHomeSection(id uint) error {
	_, err := d.Exec("DELETE FROM home_sections WHERE id = ?", id)
	return err
}

func (d *DB) ReorderHomeSections(ids []uint) error {
	tx, err := d.Begin()
	if err != nil {
		return err
	}

	for i, id := range ids {
		_, err := tx.Exec("UPDATE home_sections SET display_order = ? WHERE id = ?", i, id)
		if err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit()
}
