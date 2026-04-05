package database

import (
	"time"

	"torrent-server/models"
)

func (d *DB) GetTorrentsForMovie(movieID uint) ([]models.Torrent, error) {
	var torrents []models.Torrent
	err := d.Where("movie_id = ?", movieID).Find(&torrents).Error
	return torrents, err
}

func (d *DB) GetTorrentByHash(hash string) (*models.Torrent, error) {
	var t models.Torrent
	if err := d.Where("hash = ?", hash).First(&t).Error; err != nil {
		return nil, err
	}
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
	return d.Create(t).Error
}

func (d *DB) DeleteTorrent(id uint) error {
	return d.Delete(&models.Torrent{}, id).Error
}

func (d *DB) GetIMDBByHash(hash string) (string, error) {
	var result struct{ ImdbCode string }
	err := d.Model(&models.Torrent{}).
		Select("movies.imdb_code").
		Joins("JOIN movies ON movies.id = torrents.movie_id").
		Where("torrents.hash = ?", hash).
		Scan(&result).Error
	return result.ImdbCode, err
}

func (d *DB) DeleteTorrentsByMovie(movieID uint) error {
	return d.Where("movie_id = ?", movieID).Delete(&models.Torrent{}).Error
}
