package database

import (
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm/clause"

	"torrent-server/models"
)

type SeriesFilter struct {
	Limit         int
	Page          int
	MinimumRating float32
	QueryTerm     string
	Genre         string
	SortBy        string
	OrderBy       string
	Year          int
	Status        string
	Network       string
}

func (d *DB) ListSeries(filter SeriesFilter) ([]models.Series, int, error) {
	if filter.Limit <= 0 || filter.Limit > 50 {
		filter.Limit = 20
	}
	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.SortBy == "" {
		filter.SortBy = "date_added"
	}
	if filter.OrderBy == "" {
		filter.OrderBy = "desc"
	}

	query := d.Model(&models.Series{})

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
	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}
	if filter.Network != "" {
		query = query.Where("network = ?", filter.Network)
	}

	var totalCount int64
	if err := query.Count(&totalCount).Error; err != nil {
		return nil, 0, err
	}

	validSortColumns := map[string]string{
		"title": "title", "year": "year", "rating": "rating",
		"date_added": "date_added_unix",
	}
	sortCol, ok := validSortColumns[filter.SortBy]
	if !ok {
		sortCol = "date_added_unix"
	}

	orderDir := "DESC"
	if strings.ToLower(filter.OrderBy) == "asc" {
		orderDir = "ASC"
	}

	offset := (filter.Page - 1) * filter.Limit

	var seriesList []models.Series
	err := query.
		Order(fmt.Sprintf("%s %s", sortCol, orderDir)).
		Limit(filter.Limit).
		Offset(offset).
		Find(&seriesList).Error

	return seriesList, int(totalCount), err
}

func (d *DB) GetSeries(id uint) (*models.Series, error) {
	var s models.Series
	if err := d.First(&s, id).Error; err != nil {
		return nil, err
	}
	return &s, nil
}

func (d *DB) GetSeriesByIMDB(imdbCode string) (*models.Series, error) {
	var s models.Series
	if err := d.Where("imdb_code = ?", imdbCode).First(&s).Error; err != nil {
		return nil, err
	}
	return &s, nil
}

func (d *DB) CreateSeries(s *models.Series) error {
	now := time.Now()
	if s.DateAdded == "" {
		s.DateAdded = now.Format("2006-01-02 15:04:05")
	}
	if s.DateAddedUnix == 0 {
		s.DateAddedUnix = now.Unix()
	}
	return d.Create(s).Error
}

func (d *DB) UpdateSeries(s *models.Series) error {
	return d.Save(s).Error
}

func (d *DB) DeleteSeries(id uint) error {
	d.Where("episode_id IN (?)",
		d.DB.Model(&models.Episode{}).Select("id").Where("series_id = ?", id),
	).Delete(&models.EpisodeTorrent{})
	d.Where("series_id = ?", id).Delete(&models.Episode{})
	d.Where("series_id = ?", id).Delete(&models.SeasonPack{})
	return d.Delete(&models.Series{}, id).Error
}

func (d *DB) GetEpisodes(seriesID uint, season int) ([]models.Episode, error) {
	query := d.Where("series_id = ?", seriesID)
	if season > 0 {
		query = query.Where("season_number = ?", season)
	}

	var episodes []models.Episode
	err := query.Order("season_number, episode_number").Find(&episodes).Error
	if err != nil {
		return nil, err
	}

	for i := range episodes {
		d.DB.Where("episode_id = ?", episodes[i].ID).Find(&episodes[i].Torrents)
	}

	return episodes, nil
}

func (d *DB) CreateEpisode(e *models.Episode) error {
	return d.Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "series_id"}, {Name: "season_number"}, {Name: "episode_number"},
		},
		DoUpdates: clause.AssignmentColumns([]string{
			"title", "summary", "air_date", "runtime", "still_image",
		}),
	}).Create(e).Error
}

func (d *DB) GetEpisodeTorrents(episodeID uint) ([]models.EpisodeTorrent, error) {
	var torrents []models.EpisodeTorrent
	err := d.Where("episode_id = ?", episodeID).Find(&torrents).Error
	return torrents, err
}

func (d *DB) CreateEpisodeTorrent(t *models.EpisodeTorrent) error {
	return d.Create(t).Error
}

func (d *DB) GetSeasonPacks(seriesID uint) ([]models.SeasonPack, error) {
	var packs []models.SeasonPack
	err := d.Where("series_id = ?", seriesID).Order("season_number").Find(&packs).Error
	return packs, err
}

func (d *DB) GetSeasonPack(id uint) (*models.SeasonPack, error) {
	var p models.SeasonPack
	if err := d.First(&p, id).Error; err != nil {
		return nil, err
	}
	return &p, nil
}

func (d *DB) CreateSeasonPack(p *models.SeasonPack) error {
	var count int64
	d.Model(&models.SeasonPack{}).Where("hash = ?", p.Hash).Count(&count)
	if count > 0 {
		return nil
	}
	return d.Create(p).Error
}
