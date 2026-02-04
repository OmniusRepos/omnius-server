package database

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

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
}

func (d *DB) ListMovies(filter MovieFilter) ([]models.Movie, int, error) {
	// Set defaults
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

	// Build query
	var conditions []string
	var args []interface{}
	argNum := 1

	if filter.MinimumRating > 0 {
		conditions = append(conditions, fmt.Sprintf("m.rating >= $%d", argNum))
		args = append(args, filter.MinimumRating)
		argNum++
	}

	if filter.QueryTerm != "" {
		conditions = append(conditions, fmt.Sprintf("(m.title LIKE $%d OR m.imdb_code = $%d)", argNum, argNum+1))
		args = append(args, "%"+filter.QueryTerm+"%", filter.QueryTerm)
		argNum += 2
	}

	if filter.Genre != "" {
		conditions = append(conditions, fmt.Sprintf("m.genres LIKE $%d", argNum))
		args = append(args, "%"+filter.Genre+"%")
		argNum++
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	// Validate sort column
	validSortColumns := map[string]bool{
		"title": true, "year": true, "rating": true,
		"date_uploaded": true, "seeds": true,
	}
	if !validSortColumns[filter.SortBy] {
		filter.SortBy = "date_uploaded"
	}

	sortColumn := "m." + filter.SortBy
	if filter.SortBy == "seeds" {
		sortColumn = "(SELECT MAX(seeds) FROM torrents WHERE movie_id = m.id)"
	}

	orderDir := "DESC"
	if strings.ToLower(filter.OrderBy) == "asc" {
		orderDir = "ASC"
	}

	// Get total count
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM movies m %s", whereClause)
	var totalCount int
	if err := d.QueryRow(countQuery, args...).Scan(&totalCount); err != nil {
		return nil, 0, err
	}

	// Get movies
	offset := (filter.Page - 1) * filter.Limit
	query := fmt.Sprintf(`
		SELECT m.id, m.imdb_code, m.title, m.title_english, m.title_long, m.slug,
		       m.year, m.rating, m.runtime, m.genres, m.summary, m.description_full,
		       m.synopsis, m.yt_trailer_code, m.language, m.background_image,
		       m.small_cover_image, m.medium_cover_image, m.large_cover_image,
		       m.date_uploaded, m.date_uploaded_unix
		FROM movies m
		%s
		ORDER BY %s %s
		LIMIT $%d OFFSET $%d
	`, whereClause, sortColumn, orderDir, argNum, argNum+1)

	args = append(args, filter.Limit, offset)

	rows, err := d.Query(query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var movies []models.Movie
	for rows.Next() {
		var m models.Movie
		var genresJSON, titleEng, titleLong, slug, summary, descFull, synopsis sql.NullString
		var ytCode, lang, bgImg, smallImg, medImg, largeImg, dateUploaded sql.NullString
		var dateUploadedUnix sql.NullInt64

		err := rows.Scan(
			&m.ID, &m.ImdbCode, &m.Title, &titleEng, &titleLong, &slug,
			&m.Year, &m.Rating, &m.Runtime, &genresJSON, &summary, &descFull,
			&synopsis, &ytCode, &lang, &bgImg,
			&smallImg, &medImg, &largeImg,
			&dateUploaded, &dateUploadedUnix,
		)
		if err != nil {
			return nil, 0, err
		}

		m.TitleEnglish = titleEng.String
		m.TitleLong = titleLong.String
		m.Slug = slug.String
		m.Summary = summary.String
		m.DescriptionFull = descFull.String
		m.Synopsis = synopsis.String
		m.YtTrailerCode = ytCode.String
		m.Language = lang.String
		m.BackgroundImage = bgImg.String
		m.SmallCoverImage = smallImg.String
		m.MediumCoverImage = medImg.String
		m.LargeCoverImage = largeImg.String
		m.DateUploaded = dateUploaded.String
		m.DateUploadedUnix = dateUploadedUnix.Int64
		m.ParseGenres(genresJSON.String)

		// Load torrents for this movie
		torrents, _ := d.GetTorrentsForMovie(m.ID)
		m.Torrents = torrents

		movies = append(movies, m)
	}

	return movies, totalCount, nil
}

func (d *DB) GetMovie(id uint) (*models.Movie, error) {
	var m models.Movie
	var genresJSON, titleEng, titleLong, slug, summary, descFull, synopsis sql.NullString
	var ytCode, lang, bgImg, smallImg, medImg, largeImg, dateUploaded sql.NullString
	var dateUploadedUnix sql.NullInt64

	err := d.QueryRow(`
		SELECT id, imdb_code, title, title_english, title_long, slug,
		       year, rating, runtime, genres, summary, description_full,
		       synopsis, yt_trailer_code, language, background_image,
		       small_cover_image, medium_cover_image, large_cover_image,
		       date_uploaded, date_uploaded_unix
		FROM movies WHERE id = $1
	`, id).Scan(
		&m.ID, &m.ImdbCode, &m.Title, &titleEng, &titleLong, &slug,
		&m.Year, &m.Rating, &m.Runtime, &genresJSON, &summary, &descFull,
		&synopsis, &ytCode, &lang, &bgImg,
		&smallImg, &medImg, &largeImg,
		&dateUploaded, &dateUploadedUnix,
	)
	if err != nil {
		return nil, err
	}

	m.TitleEnglish = titleEng.String
	m.TitleLong = titleLong.String
	m.Slug = slug.String
	m.Summary = summary.String
	m.DescriptionFull = descFull.String
	m.Synopsis = synopsis.String
	m.YtTrailerCode = ytCode.String
	m.Language = lang.String
	m.BackgroundImage = bgImg.String
	m.SmallCoverImage = smallImg.String
	m.MediumCoverImage = medImg.String
	m.LargeCoverImage = largeImg.String
	m.DateUploaded = dateUploaded.String
	m.DateUploadedUnix = dateUploadedUnix.Int64
	m.ParseGenres(genresJSON.String)

	torrents, _ := d.GetTorrentsForMovie(m.ID)
	m.Torrents = torrents

	return &m, nil
}

func (d *DB) GetMovieByIMDB(imdbCode string) (*models.Movie, error) {
	var id uint
	err := d.QueryRow("SELECT id FROM movies WHERE imdb_code = $1", imdbCode).Scan(&id)
	if err != nil {
		return nil, err
	}
	return d.GetMovie(id)
}

func (d *DB) CreateMovie(m *models.Movie) error {
	now := time.Now()
	if m.DateUploaded == "" {
		m.DateUploaded = now.Format("2006-01-02 15:04:05")
	}
	if m.DateUploadedUnix == 0 {
		m.DateUploadedUnix = now.Unix()
	}

	result, err := d.Exec(`
		INSERT INTO movies (imdb_code, title, title_english, title_long, slug, year, rating, runtime,
		                    genres, summary, description_full, synopsis, yt_trailer_code, language,
		                    background_image, small_cover_image, medium_cover_image, large_cover_image,
		                    date_uploaded, date_uploaded_unix)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20)
	`, m.ImdbCode, m.Title, m.TitleEnglish, m.TitleLong, m.Slug, m.Year, m.Rating, m.Runtime,
		m.GenresJSON(), m.Summary, m.DescriptionFull, m.Synopsis, m.YtTrailerCode, m.Language,
		m.BackgroundImage, m.SmallCoverImage, m.MediumCoverImage, m.LargeCoverImage,
		m.DateUploaded, m.DateUploadedUnix)
	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	m.ID = uint(id)
	return nil
}

func (d *DB) UpdateMovie(m *models.Movie) error {
	_, err := d.Exec(`
		UPDATE movies SET
			imdb_code = $1, title = $2, title_english = $3, title_long = $4, slug = $5,
			year = $6, rating = $7, runtime = $8, genres = $9, summary = $10,
			description_full = $11, synopsis = $12, yt_trailer_code = $13, language = $14,
			background_image = $15, small_cover_image = $16, medium_cover_image = $17,
			large_cover_image = $18
		WHERE id = $19
	`, m.ImdbCode, m.Title, m.TitleEnglish, m.TitleLong, m.Slug,
		m.Year, m.Rating, m.Runtime, m.GenresJSON(), m.Summary,
		m.DescriptionFull, m.Synopsis, m.YtTrailerCode, m.Language,
		m.BackgroundImage, m.SmallCoverImage, m.MediumCoverImage, m.LargeCoverImage, m.ID)
	return err
}

func (d *DB) DeleteMovie(id uint) error {
	_, err := d.Exec("DELETE FROM movies WHERE id = $1", id)
	return err
}

func (d *DB) GetMovieSuggestions(movieID uint, limit int) ([]models.Movie, error) {
	if limit <= 0 {
		limit = 4
	}

	// Get similar movies by genre or year
	movie, err := d.GetMovie(movieID)
	if err != nil {
		return nil, err
	}

	query := `
		SELECT id FROM movies
		WHERE id != $1
		ORDER BY ABS(year - $2), rating DESC
		LIMIT $3
	`

	rows, err := d.Query(query, movieID, movie.Year, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var movies []models.Movie
	for rows.Next() {
		var id uint
		if err := rows.Scan(&id); err != nil {
			continue
		}
		if m, err := d.GetMovie(id); err == nil {
			movies = append(movies, *m)
		}
	}

	return movies, nil
}

// GetMovieRating returns the rating info for a movie by IMDB code
func (d *DB) GetMovieRating(imdbCode string) (*models.LocalRating, error) {
	var rating, imdbRating float32
	var rottenTomatoes, metacritic int

	err := d.QueryRow(`
		SELECT rating, COALESCE(imdb_rating, rating), COALESCE(rotten_tomatoes, 0), COALESCE(metacritic, 0)
		FROM movies WHERE imdb_code = $1
	`, imdbCode).Scan(&rating, &imdbRating, &rottenTomatoes, &metacritic)
	if err != nil {
		return nil, err
	}

	result := &models.LocalRating{}
	if imdbRating > 0 {
		result.ImdbRating = &imdbRating
	}
	if rottenTomatoes > 0 {
		result.RottenTomatoes = &rottenTomatoes
	}
	if metacritic > 0 {
		result.Metacritic = &metacritic
	}

	return result, nil
}
