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
		SELECT id, imdb_code, title, title_slug, year, rating, COALESCE(runtime, 0), genres, summary,
		       poster_image, background_image, total_seasons, status, COALESCE(network, ''),
		       date_added, date_added_unix
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
		var genresJSON, slug, summary, poster, bg, network, dateAdded sql.NullString
		var dateAddedUnix sql.NullInt64

		err := rows.Scan(
			&s.ID, &s.ImdbCode, &s.Title, &slug, &s.Year, &s.Rating, &s.Runtime, &genresJSON, &summary,
			&poster, &bg, &s.TotalSeasons, &s.Status, &network,
			&dateAdded, &dateAddedUnix,
		)
		if err != nil {
			continue
		}

		s.TitleSlug = slug.String
		s.Summary = summary.String
		s.PosterImage = poster.String
		s.BackgroundImage = bg.String
		s.Network = network.String
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
	var genresJSON, slug, summary, poster, bg, network, dateAdded sql.NullString
	var dateAddedUnix sql.NullInt64

	err := d.QueryRow(`
		SELECT id, imdb_code, title, title_slug, year, rating, COALESCE(runtime, 0), genres, summary,
		       poster_image, background_image, total_seasons, status, COALESCE(network, ''),
		       date_added, date_added_unix
		FROM series WHERE id = $1
	`, id).Scan(
		&s.ID, &s.ImdbCode, &s.Title, &slug, &s.Year, &s.Rating, &s.Runtime, &genresJSON, &summary,
		&poster, &bg, &s.TotalSeasons, &s.Status, &network,
		&dateAdded, &dateAddedUnix,
	)
	if err != nil {
		return nil, err
	}

	s.TitleSlug = slug.String
	s.Summary = summary.String
	s.PosterImage = poster.String
	s.BackgroundImage = bg.String
	s.Network = network.String
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
		INSERT INTO series (imdb_code, title, title_slug, year, rating, runtime, genres, summary,
		                    poster_image, background_image, total_seasons, status, network,
		                    date_added, date_added_unix)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
	`, s.ImdbCode, s.Title, s.TitleSlug, s.Year, s.Rating, s.Runtime, string(genresJSON), s.Summary,
		s.PosterImage, s.BackgroundImage, s.TotalSeasons, s.Status, s.Network,
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
		SELECT id, series_id, season_number, episode_number, title, COALESCE(summary, ''), air_date, runtime, still_image
		FROM episodes WHERE series_id = $1
	`
	args := []interface{}{seriesID}

	if season > 0 {
		query += " AND season_number = $2"
		args = append(args, season)
	}
	query += " ORDER BY season_number, episode_number"

	rows, err := d.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var episodes []models.Episode
	for rows.Next() {
		var e models.Episode
		var title, summary, airDate, stillImage sql.NullString
		var runtime sql.NullInt64

		err := rows.Scan(&e.ID, &e.SeriesID, &e.SeasonNumber, &e.EpisodeNumber, &title, &summary, &airDate, &runtime, &stillImage)
		if err != nil {
			continue
		}

		e.Title = title.String
		e.Summary = summary.String
		e.AirDate = airDate.String
		if runtime.Valid {
			r := uint(runtime.Int64)
			e.Runtime = &r
		}
		e.StillImage = stillImage.String

		// Load torrents
		torrents, _ := d.GetEpisodeTorrents(e.ID)
		e.Torrents = torrents

		episodes = append(episodes, e)
	}

	return episodes, nil
}

func (d *DB) CreateEpisode(e *models.Episode) error {
	result, err := d.Exec(`
		INSERT INTO episodes (series_id, season_number, episode_number, title, summary, air_date, runtime, still_image)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT(series_id, season_number, episode_number) DO UPDATE SET
			title = excluded.title,
			summary = excluded.summary,
			air_date = excluded.air_date,
			runtime = excluded.runtime,
			still_image = excluded.still_image
	`, e.SeriesID, e.SeasonNumber, e.EpisodeNumber, e.Title, e.Summary, e.AirDate, e.Runtime, e.StillImage)
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
		SELECT id, episode_id, COALESCE(series_id, 0), COALESCE(season_number, 0), COALESCE(episode_number, 0),
		       hash, quality, video_codec, COALESCE(seeds, 0), COALESCE(peers, 0), size, COALESCE(size_bytes, 0), COALESCE(file_index, -1), release_group,
		       COALESCE(date_uploaded, ''), COALESCE(date_uploaded_unix, 0)
		FROM episode_torrents WHERE episode_id = $1
	`, episodeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var torrents []models.EpisodeTorrent
	for rows.Next() {
		var t models.EpisodeTorrent
		var quality, videoCodec, size, releaseGroup, dateUploaded sql.NullString

		err := rows.Scan(&t.ID, &t.EpisodeID, &t.SeriesID, &t.SeasonNumber, &t.EpisodeNumber,
			&t.Hash, &quality, &videoCodec, &t.Seeds, &t.Peers, &size, &t.SizeBytes, &t.FileIndex, &releaseGroup,
			&dateUploaded, &t.DateUploadedUnix)
		if err != nil {
			continue
		}

		t.Quality = quality.String
		t.VideoCodec = videoCodec.String
		t.Size = size.String
		t.ReleaseGroup = releaseGroup.String
		t.DateUploaded = dateUploaded.String

		torrents = append(torrents, t)
	}

	return torrents, nil
}

func (d *DB) CreateEpisodeTorrent(t *models.EpisodeTorrent) error {
	result, err := d.Exec(`
		INSERT INTO episode_torrents (episode_id, series_id, season_number, episode_number, hash, quality,
		                              video_codec, seeds, peers, size, size_bytes, file_index, release_group,
		                              date_uploaded, date_uploaded_unix)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
	`, t.EpisodeID, t.SeriesID, t.SeasonNumber, t.EpisodeNumber, t.Hash, t.Quality,
		t.VideoCodec, t.Seeds, t.Peers, t.Size, t.SizeBytes, t.FileIndex, t.ReleaseGroup,
		t.DateUploaded, t.DateUploadedUnix)
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
		SELECT id, series_id, season_number, hash, quality, seeds, peers, size, size_bytes, source
		FROM season_packs WHERE series_id = $1 ORDER BY season_number
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

func (d *DB) GetSeasonPack(id uint) (*models.SeasonPack, error) {
	var p models.SeasonPack
	var quality, size, source sql.NullString

	err := d.QueryRow(`
		SELECT id, series_id, season_number, hash, quality, seeds, peers, size, size_bytes, source
		FROM season_packs WHERE id = $1
	`, id).Scan(&p.ID, &p.SeriesID, &p.Season, &p.Hash, &quality, &p.Seeds, &p.Peers, &size, &p.SizeBytes, &source)
	if err != nil {
		return nil, err
	}

	p.Quality = quality.String
	p.Size = size.String
	p.Source = source.String

	return &p, nil
}

func (d *DB) CreateSeasonPack(p *models.SeasonPack) error {
	// Check if this hash already exists
	var exists int
	d.QueryRow("SELECT 1 FROM season_packs WHERE hash = $1", p.Hash).Scan(&exists)
	if exists == 1 {
		return nil // Already exists, skip
	}

	result, err := d.Exec(`
		INSERT INTO season_packs (series_id, season_number, hash, quality, seeds, peers, size, size_bytes, source)
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

func (d *DB) DeleteSeries(id uint) error {
	// Delete episode torrents first
	_, err := d.Exec(`
		DELETE FROM episode_torrents WHERE episode_id IN (
			SELECT id FROM episodes WHERE series_id = $1
		)
	`, id)
	if err != nil {
		return err
	}

	// Delete episodes
	_, err = d.Exec("DELETE FROM episodes WHERE series_id = $1", id)
	if err != nil {
		return err
	}

	// Delete season packs
	_, err = d.Exec("DELETE FROM season_packs WHERE series_id = $1", id)
	if err != nil {
		return err
	}

	// Delete the series
	_, err = d.Exec("DELETE FROM series WHERE id = $1", id)
	return err
}

func (d *DB) UpdateSeries(s *models.Series) error {
	genresJSON, _ := json.Marshal(s.Genres)

	_, err := d.Exec(`
		UPDATE series SET
			imdb_code = $1, title = $2, title_slug = $3, year = $4, rating = $5,
			genres = $6, summary = $7, poster_image = $8, background_image = $9,
			total_seasons = $10, total_episodes = $11, status = $12, runtime = $13, network = $14
		WHERE id = $15
	`, s.ImdbCode, s.Title, s.TitleSlug, s.Year, s.Rating,
		string(genresJSON), s.Summary, s.PosterImage, s.BackgroundImage,
		s.TotalSeasons, s.TotalEpisodes, s.Status, s.Runtime, s.Network, s.ID)
	return err
}
