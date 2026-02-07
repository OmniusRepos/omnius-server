package database

import (
	"database/sql"
	"time"

	"torrent-server/models"
)

func (d *DB) GetTorrentsForMovie(movieID uint) ([]models.Torrent, error) {
	rows, err := d.Query(`
		SELECT id, movie_id, url, hash, quality, type, video_codec,
		       seeds, peers, size, size_bytes, date_uploaded, date_uploaded_unix
		FROM torrents WHERE movie_id = $1
	`, movieID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var torrents []models.Torrent
	for rows.Next() {
		var t models.Torrent
		var url, quality, ttype, codec, size, dateUploaded sql.NullString
		var dateUploadedUnix sql.NullInt64

		err := rows.Scan(
			&t.ID, &t.MovieID, &url, &t.Hash, &quality, &ttype, &codec,
			&t.Seeds, &t.Peers, &size, &t.SizeBytes, &dateUploaded, &dateUploadedUnix,
		)
		if err != nil {
			continue
		}

		t.URL = url.String
		t.Quality = quality.String
		t.Type = ttype.String
		t.VideoCodec = codec.String
		t.Size = size.String
		t.DateUploaded = dateUploaded.String
		t.DateUploadedUnix = dateUploadedUnix.Int64

		torrents = append(torrents, t)
	}

	return torrents, nil
}

func (d *DB) GetTorrentByHash(hash string) (*models.Torrent, error) {
	var t models.Torrent
	var url, quality, ttype, codec, size, dateUploaded sql.NullString
	var dateUploadedUnix sql.NullInt64

	err := d.QueryRow(`
		SELECT id, movie_id, url, hash, quality, type, video_codec,
		       seeds, peers, size, size_bytes, date_uploaded, date_uploaded_unix
		FROM torrents WHERE hash = $1
	`, hash).Scan(
		&t.ID, &t.MovieID, &url, &t.Hash, &quality, &ttype, &codec,
		&t.Seeds, &t.Peers, &size, &t.SizeBytes, &dateUploaded, &dateUploadedUnix,
	)
	if err != nil {
		return nil, err
	}

	t.URL = url.String
	t.Quality = quality.String
	t.Type = ttype.String
	t.VideoCodec = codec.String
	t.Size = size.String
	t.DateUploaded = dateUploaded.String
	t.DateUploadedUnix = dateUploadedUnix.Int64

	return &t, nil
}

func (d *DB) CreateTorrent(t *models.Torrent) error {
	now := time.Now()
	if t.DateUploaded == "" {
		t.DateUploaded = now.Format("2006-01-02 15:04:05")
	}
	if t.DateUploadedUnix == 0 {
		t.DateUploadedUnix = now.Unix()
	}

	result, err := d.Exec(`
		INSERT INTO torrents (movie_id, url, hash, quality, type, video_codec,
		                      seeds, peers, size, size_bytes, date_uploaded, date_uploaded_unix)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`, t.MovieID, t.URL, t.Hash, t.Quality, t.Type, t.VideoCodec,
		t.Seeds, t.Peers, t.Size, t.SizeBytes, t.DateUploaded, t.DateUploadedUnix)
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

func (d *DB) DeleteTorrent(id uint) error {
	_, err := d.Exec("DELETE FROM torrents WHERE id = $1", id)
	return err
}

// GetIMDBByHash looks up the IMDB code for a torrent by its info hash.
func (d *DB) GetIMDBByHash(hash string) (string, error) {
	var imdbCode string
	err := d.QueryRow(`
		SELECT m.imdb_code FROM torrents t
		JOIN movies m ON m.id = t.movie_id
		WHERE t.hash = $1
	`, hash).Scan(&imdbCode)
	return imdbCode, err
}

func (d *DB) DeleteTorrentsByMovie(movieID uint) error {
	_, err := d.Exec("DELETE FROM torrents WHERE movie_id = $1", movieID)
	return err
}
