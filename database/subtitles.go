package database

import (
	"fmt"
	"strings"

	"gorm.io/gorm/clause"

	"torrent-server/models"
)

func (d *DB) GetSubtitlesByIMDB(imdbCode, language string) ([]models.StoredSubtitle, error) {
	return d.GetSubtitlesByIMDBEpisode(imdbCode, language, 0, 0)
}

func (d *DB) GetSubtitlesByIMDBEpisode(imdbCode, language string, season, episode int) ([]models.StoredSubtitle, error) {
	query := d.Model(&models.StoredSubtitle{}).Where("imdb_code = ?", imdbCode)

	if season > 0 {
		query = query.Where("season_number = ?", season)
	}
	if episode > 0 {
		query = query.Where("episode_number = ?", episode)
	}
	if language != "" {
		langs := strings.Split(language, ",")
		trimmed := make([]string, len(langs))
		for i, l := range langs {
			trimmed[i] = strings.TrimSpace(l)
		}
		query = query.Where("language IN ?", trimmed)
	}

	var subtitles []models.StoredSubtitle
	err := query.
		Select("id, imdb_code, language, language_name, release_name, hearing_impaired, source, season_number, episode_number, created_at").
		Order("season_number, episode_number, created_at DESC").
		Find(&subtitles).Error
	if err != nil {
		return nil, fmt.Errorf("failed to query subtitles: %w", err)
	}
	return subtitles, nil
}

func (d *DB) GetSubtitleByID(id uint) (*models.StoredSubtitle, error) {
	var sub models.StoredSubtitle
	if err := d.First(&sub, id).Error; err != nil {
		return nil, fmt.Errorf("subtitle not found")
	}
	return &sub, nil
}

func (d *DB) CreateSubtitle(sub *models.StoredSubtitle) error {
	result := d.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "imdb_code"}, {Name: "language"}, {Name: "release_name"}},
		DoNothing: true,
	}).Create(sub)
	if result.Error != nil {
		return fmt.Errorf("failed to create subtitle: %w", result.Error)
	}
	return nil
}

func (d *DB) UpdateSubtitlePath(id uint, vttPath string) error {
	return d.Model(&models.StoredSubtitle{}).Where("id = ?", id).
		Updates(map[string]interface{}{"vtt_path": vttPath, "vtt_content": ""}).Error
}

func (d *DB) GetSubtitlesWithContent() ([]models.StoredSubtitle, error) {
	var subs []models.StoredSubtitle
	err := d.Select("id, imdb_code, vtt_content").
		Where("vtt_content != '' AND (vtt_path = '' OR vtt_path IS NULL)").
		Find(&subs).Error
	return subs, err
}

func (d *DB) DeleteSubtitle(id uint) error {
	return d.Delete(&models.StoredSubtitle{}, id).Error
}

func (d *DB) DeleteSubtitlesByIMDB(imdbCode string) error {
	return d.Where("imdb_code = ?", imdbCode).Delete(&models.StoredSubtitle{}).Error
}

func (d *DB) CountSubtitlesByIMDB(imdbCode string) (int, error) {
	var count int64
	err := d.Model(&models.StoredSubtitle{}).Where("imdb_code = ?", imdbCode).Count(&count).Error
	return int(count), err
}

func (d *DB) CountSubtitlesByIMDBEpisode(imdbCode string, season, episode int) (int, error) {
	var count int64
	err := d.Model(&models.StoredSubtitle{}).
		Where("imdb_code = ? AND season_number = ? AND episode_number = ?", imdbCode, season, episode).
		Count(&count).Error
	return int(count), err
}
