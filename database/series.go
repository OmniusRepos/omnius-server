package database

import (
	"database/sql"
	"encoding/json"
	"time"

	"torrent-server/models"
)

func (d *DB) ListSeries(limit, page int) ([]models.Series, int, error) {
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	if page <= 0 {
		page = 1
	}

	var totalCount int
	if err := d.QueryRow("SELECT COUNT(*) FROM series").Scan(&totalCount); err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * limit
	rows, err := d.Query(`
		SELECT id, imdb_code, title, title_slug, year, rating, genres, summary,
		       poster_image, background_image, total_seasons, status, date_added, date_added_unix
		FROM series
		ORDER BY date_added_unix DESC
		LIMIT $1 OFFSET $2
	`, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var seriesList []models.Series
	for rows.Next() {
		var s models.Series
		var genresJSON, slug, summary, poster, bg, dateAdded sql.NullString
		var dateAddedUnix sql.NullInt64

		err := rows.Scan(
			&s.ID, &s.ImdbCode, &s.Title, &slug, &s.Year, &s.Rating, &genresJSON, &summary,
			&poster, &bg, &s.TotalSeasons, &s.Status, &dateAdded, &dateAddedUnix,
		)
		if err != nil {
			continue
		}

		s.TitleSlug = slug.String
		s.Summary = summary.String
		s.PosterImage = poster.String
		s.BackgroundImage = bg.String
		s.DateAdded = dateAdded.String
		s.DateAddedUnix = dateAddedUnix.Int64
		if genresJSON.String != "" {
			json.Unmarshal([]byte(genresJSON.String), &s.Genres)
		}

		seriesList = append(seriesList, s)
	}

	return seriesList, totalCount, nil
}

func (d *DB) GetSeries(id uint) (*models.Series, error) {
	var s models.Series
	var genresJSON, slug, summary, poster, bg, dateAdded sql.NullString
	var dateAddedUnix sql.NullInt64

	err := d.QueryRow(`
		SELECT id, imdb_code, title, title_slug, year, rating, genres, summary,
		       poster_image, background_image, total_seasons, status, date_added, date_added_unix
		FROM series WHERE id = $1
	`, id).Scan(
		&s.ID, &s.ImdbCode, &s.Title, &slug, &s.Year, &s.Rating, &genresJSON, &summary,
		&poster, &bg, &s.TotalSeasons, &s.Status, &dateAdded, &dateAddedUnix,
	)
	if err != nil {
		return nil, err
	}

	s.TitleSlug = slug.String
	s.Summary = summary.String
	s.PosterImage = poster.String
	s.BackgroundImage = bg.String
	s.DateAdded = dateAdded.String
	s.DateAddedUnix = dateAddedUnix.Int64
	if genresJSON.String != "" {
		json.Unmarshal([]byte(genresJSON.String), &s.Genres)
	}

	return &s, nil
}

func (d *DB) GetSeriesByIMDB(imdbCode string) (*models.Series, error) {
	var id uint
	err := d.QueryRow("SELECT id FROM series WHERE imdb_code = $1", imdbCode).Scan(&id)
	if err != nil {
		return nil, err
	}
	return d.GetSeries(id)
}

func (d *DB) CreateSeries(s *models.Series) error {
	now := time.Now()
	if s.DateAdded == "" {
		s.DateAdded = now.Format("2006-01-02 15:04:05")
	}
	if s.DateAddedUnix == 0 {
		s.DateAddedUnix = now.Unix()
	}

	genresJSON, _ := json.Marshal(s.Genres)

	result, err := d.Exec(`
		INSERT INTO series (imdb_code, title, title_slug, year, rating, genres, summary,
		                    poster_image, background_image, total_seasons, status,
		                    date_added, date_added_unix)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`, s.ImdbCode, s.Title, s.TitleSlug, s.Year, s.Rating, string(genresJSON), s.Summary,
		s.PosterImage, s.BackgroundImage, s.TotalSeasons, s.Status,
		s.DateAdded, s.DateAddedUnix)
	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	s.ID = uint(id)
	return nil
}

func (d *DB) GetEpisodes(seriesID uint, season int) ([]models.Episode, error) {
	query := `
		SELECT id, series_id, season, episode, title, overview, air_date, imdb_code
		FROM episodes WHERE series_id = $1
	`
	args := []interface{}{seriesID}

	if season > 0 {
		query += " AND season = $2"
		args = append(args, season)
	}
	query += " ORDER BY season, episode"

	rows, err := d.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var episodes []models.Episode
	for rows.Next() {
		var e models.Episode
		var title, overview, airDate, imdbCode sql.NullString

		err := rows.Scan(&e.ID, &e.SeriesID, &e.Season, &e.Episode, &title, &overview, &airDate, &imdbCode)
		if err != nil {
			continue
		}

		e.Title = title.String
		e.Overview = overview.String
		e.AirDate = airDate.String
		e.ImdbCode = imdbCode.String

		// Load torrents
		torrents, _ := d.GetEpisodeTorrents(e.ID)
		e.Torrents = torrents

		episodes = append(episodes, e)
	}

	return episodes, nil
}

func (d *DB) CreateEpisode(e *models.Episode) error {
	result, err := d.Exec(`
		INSERT INTO episodes (series_id, season, episode, title, overview, air_date, imdb_code)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT(series_id, season, episode) DO UPDATE SET
			title = excluded.title,
			overview = excluded.overview,
			air_date = excluded.air_date,
			imdb_code = excluded.imdb_code
	`, e.SeriesID, e.Season, e.Episode, e.Title, e.Overview, e.AirDate, e.ImdbCode)
	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	e.ID = uint(id)
	return nil
}

func (d *DB) GetEpisodeTorrents(episodeID uint) ([]models.EpisodeTorrent, error) {
	rows, err := d.Query(`
		SELECT id, episode_id, hash, quality, seeds, peers, size, size_bytes, source
		FROM episode_torrents WHERE episode_id = $1
	`, episodeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var torrents []models.EpisodeTorrent
	for rows.Next() {
		var t models.EpisodeTorrent
		var quality, size, source sql.NullString

		err := rows.Scan(&t.ID, &t.EpisodeID, &t.Hash, &quality, &t.Seeds, &t.Peers, &size, &t.SizeBytes, &source)
		if err != nil {
			continue
		}

		t.Quality = quality.String
		t.Size = size.String
		t.Source = source.String

		torrents = append(torrents, t)
	}

	return torrents, nil
}

func (d *DB) CreateEpisodeTorrent(t *models.EpisodeTorrent) error {
	result, err := d.Exec(`
		INSERT INTO episode_torrents (episode_id, hash, quality, seeds, peers, size, size_bytes, source)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, t.EpisodeID, t.Hash, t.Quality, t.Seeds, t.Peers, t.Size, t.SizeBytes, t.Source)
	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	t.ID = uint(id)
	return nil
}

func (d *DB) GetSeasonPacks(seriesID uint) ([]models.SeasonPack, error) {
	rows, err := d.Query(`
		SELECT id, series_id, season, hash, quality, seeds, peers, size, size_bytes, source
		FROM season_packs WHERE series_id = $1 ORDER BY season
	`, seriesID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var packs []models.SeasonPack
	for rows.Next() {
		var p models.SeasonPack
		var quality, size, source sql.NullString

		err := rows.Scan(&p.ID, &p.SeriesID, &p.Season, &p.Hash, &quality, &p.Seeds, &p.Peers, &size, &p.SizeBytes, &source)
		if err != nil {
			continue
		}

		p.Quality = quality.String
		p.Size = size.String
		p.Source = source.String

		packs = append(packs, p)
	}

	return packs, nil
}

func (d *DB) CreateSeasonPack(p *models.SeasonPack) error {
	result, err := d.Exec(`
		INSERT INTO season_packs (series_id, season, hash, quality, seeds, peers, size, size_bytes, source)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, p.SeriesID, p.Season, p.Hash, p.Quality, p.Seeds, p.Peers, p.Size, p.SizeBytes, p.Source)
	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	p.ID = uint(id)
	return nil
}
