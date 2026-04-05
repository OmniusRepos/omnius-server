package database

import (
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"

	"torrent-server/models"
)

type MovieFilter struct {
	Limit         int
	Page          int
	Quality       string
	MinimumRating float32
	QueryTerm     string
	Genre         string
	SortBy        string
	OrderBy       string
	Year          int
	MaximumYear   int
	MinimumYear   int
	Status        string
}

func (d *DB) ListMovies(filter MovieFilter) ([]models.Movie, int, error) {
	if filter.Limit <= 0 || filter.Limit > 50 {
		filter.Limit = 20
	}
	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.SortBy == "" {
		filter.SortBy = "date_uploaded"
	}
	if filter.OrderBy == "" {
		filter.OrderBy = "desc"
	}

	query := d.Model(&models.Movie{})

	if filter.MinimumRating > 0 {
		query = query.Where("rating >= ?", filter.MinimumRating)
	}
	if filter.QueryTerm != "" {
		query = query.Where("title LIKE ? OR imdb_code = ?", "%"+filter.QueryTerm+"%", filter.QueryTerm)
	}
	if filter.Genre != "" {
		query = query.Where("genres LIKE ?", "%"+filter.Genre+"%")
	}
	if filter.Year > 0 {
		query = query.Where("year = ?", filter.Year)
	}
	if filter.MinimumYear > 0 {
		query = query.Where("year >= ?", filter.MinimumYear)
	}
	if filter.MaximumYear > 0 {
		query = query.Where("year <= ?", filter.MaximumYear)
	}
	if filter.Status != "" {
		query = query.Where("COALESCE(status, 'available') = ?", filter.Status)
	}

	var totalCount int64
	if err := query.Count(&totalCount).Error; err != nil {
		return nil, 0, err
	}

	validSortColumns := map[string]string{
		"title": "title", "year": "year", "rating": "rating",
		"date_uploaded": "date_uploaded_unix", "date_added": "date_uploaded_unix",
		"seeds": "seeds", "download_count": "download_count",
	}
	sortCol, ok := validSortColumns[filter.SortBy]
	if !ok {
		sortCol = "date_uploaded_unix"
	}

	orderDir := "DESC"
	if strings.ToLower(filter.OrderBy) == "asc" {
		orderDir = "ASC"
	}

	offset := (filter.Page - 1) * filter.Limit

	var movies []models.Movie
	err := query.
		Order(fmt.Sprintf("%s %s", sortCol, orderDir)).
		Limit(filter.Limit).
		Offset(offset).
		Find(&movies).Error
	if err != nil {
		return nil, 0, err
	}

	for i := range movies {
		d.DB.Where("movie_id = ?", movies[i].ID).Find(&movies[i].Torrents)
	}

	return movies, int(totalCount), nil
}

func (d *DB) GetMovie(id uint) (*models.Movie, error) {
	var m models.Movie
	if err := d.Preload("Torrents").First(&m, id).Error; err != nil {
		return nil, err
	}
	if m.Status == "" {
		m.Status = "available"
	}
	return &m, nil
}

func (d *DB) GetMovieByIMDB(imdbCode string) (*models.Movie, error) {
	var m models.Movie
	if err := d.Preload("Torrents").Where("imdb_code = ?", imdbCode).First(&m).Error; err != nil {
		return nil, err
	}
	if m.Status == "" {
		m.Status = "available"
	}
	return &m, nil
}

func (d *DB) CreateMovie(m *models.Movie) error {
	now := time.Now()
	if m.DateUploaded == "" {
		m.DateUploaded = now.Format("2006-01-02 15:04:05")
	}
	if m.DateUploadedUnix == 0 {
		m.DateUploadedUnix = now.Unix()
	}
	if m.Status == "" {
		m.Status = "available"
	}
	return d.Create(m).Error
}

func (d *DB) UpdateMovie(m *models.Movie) error {
	return d.Save(m).Error
}

func (d *DB) DeleteMovie(id uint) error {
	return d.Delete(&models.Movie{}, id).Error
}

func (d *DB) GetMovieSuggestions(movieID uint, limit int) ([]models.Movie, error) {
	if limit <= 0 {
		limit = 4
	}

	movie, err := d.GetMovie(movieID)
	if err != nil {
		return nil, err
	}

	var movies []models.Movie
	err = d.Preload("Torrents").
		Where("id != ?", movieID).
		Order(gorm.Expr("ABS(year - ?) ASC, rating DESC", movie.Year)).
		Limit(limit).
		Find(&movies).Error

	return movies, err
}

func (d *DB) GetFranchiseMovies(movieID uint, franchise string) ([]models.Movie, error) {
	if franchise == "" {
		return nil, nil
	}

	var movies []models.Movie
	err := d.Preload("Torrents").
		Where("franchise = ? AND id != ?", franchise, movieID).
		Order("year ASC").
		Find(&movies).Error

	return movies, err
}

func (d *DB) GetMovieRating(imdbCode string) (*models.LocalRating, error) {
	var m models.Movie
	err := d.Select("rating, imdb_rating, rotten_tomatoes, metacritic").
		Where("imdb_code = ?", imdbCode).
		First(&m).Error
	if err != nil {
		return nil, err
	}

	result := &models.LocalRating{}
	if m.ImdbRating != nil && *m.ImdbRating > 0 {
		result.ImdbRating = m.ImdbRating
	} else if m.Rating > 0 {
		r := m.Rating
		result.ImdbRating = &r
	}
	if m.RottenTomatoes != nil && *m.RottenTomatoes > 0 {
		result.RottenTomatoes = m.RottenTomatoes
	}
	if m.Metacritic != nil && *m.Metacritic > 0 {
		result.Metacritic = m.Metacritic
	}

	return result, nil
}
