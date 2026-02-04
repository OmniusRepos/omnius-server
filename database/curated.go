package database

import (
	"database/sql"
	"strconv"
	"strings"

	"torrent-server/models"
)

// ListCuratedLists returns all active curated lists
func (d *DB) ListCuratedLists(includeInactive bool) ([]models.CuratedList, error) {
	query := `SELECT id, name, slug, description, sort_by, order_by,
	                 minimum_rating, maximum_rating, minimum_year, maximum_year,
	                 genre, limit_count, is_active, display_order, created_at
	          FROM curated_lists`
	if !includeInactive {
		query += " WHERE is_active = 1"
	}
	query += " ORDER BY display_order ASC, name ASC"

	rows, err := d.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lists []models.CuratedList
	for rows.Next() {
		var l models.CuratedList
		var desc, genre sql.NullString
		var minYear, maxYear sql.NullInt64
		var minRating, maxRating sql.NullFloat64
		var isActive int

		err := rows.Scan(
			&l.ID, &l.Name, &l.Slug, &desc, &l.SortBy, &l.OrderBy,
			&minRating, &maxRating, &minYear, &maxYear,
			&genre, &l.LimitCount, &isActive, &l.DisplayOrder, &l.CreatedAt,
		)
		if err != nil {
			continue
		}
		l.IsActive = isActive == 1

		l.Description = desc.String
		l.Genre = genre.String
		if minYear.Valid {
			l.MinimumYear = int(minYear.Int64)
		}
		if maxYear.Valid {
			l.MaximumYear = int(maxYear.Int64)
		}
		if minRating.Valid {
			l.MinimumRating = float32(minRating.Float64)
		}
		if maxRating.Valid {
			l.MaximumRating = float32(maxRating.Float64)
		}

		lists = append(lists, l)
	}

	return lists, nil
}

// GetCuratedListByID returns a single curated list by numeric ID
func (d *DB) GetCuratedListByID(id uint) (*models.CuratedList, error) {
	return d.GetCuratedList(strconv.Itoa(int(id)))
}

// GetCuratedList returns a single curated list by ID or slug
func (d *DB) GetCuratedList(idOrSlug string) (*models.CuratedList, error) {
	query := `SELECT id, name, slug, description, sort_by, order_by,
	                 minimum_rating, maximum_rating, minimum_year, maximum_year,
	                 genre, limit_count, is_active, display_order, created_at
	          FROM curated_lists WHERE id = $1 OR slug = $1`

	var l models.CuratedList
	var desc, genre sql.NullString
	var minYear, maxYear sql.NullInt64
	var minRating, maxRating sql.NullFloat64
	var isActive int

	err := d.QueryRow(query, idOrSlug).Scan(
		&l.ID, &l.Name, &l.Slug, &desc, &l.SortBy, &l.OrderBy,
		&minRating, &maxRating, &minYear, &maxYear,
		&genre, &l.LimitCount, &isActive, &l.DisplayOrder, &l.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	l.IsActive = isActive == 1

	l.Description = desc.String
	l.Genre = genre.String
	if minYear.Valid {
		l.MinimumYear = int(minYear.Int64)
	}
	if maxYear.Valid {
		l.MaximumYear = int(maxYear.Int64)
	}
	if minRating.Valid {
		l.MinimumRating = float32(minRating.Float64)
	}
	if maxRating.Valid {
		l.MaximumRating = float32(maxRating.Float64)
	}

	return &l, nil
}

// GetCuratedListMovies returns movies for a curated list (either from filter or hand-picked)
func (d *DB) GetCuratedListMovies(list *models.CuratedList) ([]models.Movie, error) {
	// First check if there are hand-picked movies
	handPickedQuery := `SELECT movie_id FROM curated_list_movies
	                    WHERE list_id = $1 ORDER BY display_order ASC`
	rows, err := d.Query(handPickedQuery, list.ID)
	if err == nil {
		defer rows.Close()
		var movieIDs []uint
		for rows.Next() {
			var id uint
			if err := rows.Scan(&id); err == nil {
				movieIDs = append(movieIDs, id)
			}
		}
		if len(movieIDs) > 0 {
			// Return hand-picked movies
			var movies []models.Movie
			for _, id := range movieIDs {
				if m, err := d.GetMovie(id); err == nil {
					movies = append(movies, *m)
				}
			}
			return movies, nil
		}
	}

	// Otherwise, use filter-based selection
	filter := MovieFilter{
		Limit:         list.LimitCount,
		SortBy:        list.SortBy,
		OrderBy:       list.OrderBy,
		MinimumRating: list.MinimumRating,
		MinimumYear:   list.MinimumYear,
		MaximumYear:   list.MaximumYear,
		Genre:         list.Genre,
	}

	movies, _, err := d.ListMovies(filter)
	return movies, err
}

// CreateCuratedList creates a new curated list
func (d *DB) CreateCuratedList(list *models.CuratedList) error {
	// Generate slug from name if not provided
	if list.Slug == "" {
		list.Slug = strings.ToLower(strings.ReplaceAll(list.Name, " ", "-"))
	}

	result, err := d.Exec(`
		INSERT INTO curated_lists (name, slug, description, sort_by, order_by,
		                          minimum_rating, maximum_rating, minimum_year, maximum_year,
		                          genre, limit_count, is_active, display_order)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`, list.Name, list.Slug, list.Description, list.SortBy, list.OrderBy,
		list.MinimumRating, list.MaximumRating, nullInt(list.MinimumYear), nullInt(list.MaximumYear),
		list.Genre, list.LimitCount, list.IsActive, list.DisplayOrder)
	if err != nil {
		return err
	}

	id, _ := result.LastInsertId()
	list.ID = uint(id)
	return nil
}

// UpdateCuratedList updates an existing curated list
func (d *DB) UpdateCuratedList(list *models.CuratedList) error {
	_, err := d.Exec(`
		UPDATE curated_lists SET
			name = $1, slug = $2, description = $3, sort_by = $4, order_by = $5,
			minimum_rating = $6, maximum_rating = $7, minimum_year = $8, maximum_year = $9,
			genre = $10, limit_count = $11, is_active = $12, display_order = $13
		WHERE id = $14
	`, list.Name, list.Slug, list.Description, list.SortBy, list.OrderBy,
		list.MinimumRating, list.MaximumRating, nullInt(list.MinimumYear), nullInt(list.MaximumYear),
		list.Genre, list.LimitCount, list.IsActive, list.DisplayOrder, list.ID)
	return err
}

// DeleteCuratedList deletes a curated list
func (d *DB) DeleteCuratedList(id uint) error {
	_, err := d.Exec("DELETE FROM curated_lists WHERE id = $1", id)
	return err
}

// AddMovieToCuratedList adds a movie to a hand-picked curated list
func (d *DB) AddMovieToCuratedList(listID, movieID uint, order int) error {
	_, err := d.Exec(`
		INSERT OR REPLACE INTO curated_list_movies (list_id, movie_id, display_order)
		VALUES ($1, $2, $3)
	`, listID, movieID, order)
	return err
}

// RemoveMovieFromCuratedList removes a movie from a curated list
func (d *DB) RemoveMovieFromCuratedList(listID, movieID uint) error {
	_, err := d.Exec("DELETE FROM curated_list_movies WHERE list_id = $1 AND movie_id = $2", listID, movieID)
	return err
}

func nullInt(v int) interface{} {
	if v == 0 {
		return nil
	}
	return v
}
