package database

import (
	"fmt"
	"time"

	"gorm.io/gorm/clause"

	"torrent-server/models"
)

type ChannelFilter struct {
	Limit     int
	Page      int
	Country   string
	Category  string
	QueryTerm string
}

func (d *DB) ListChannels(filter ChannelFilter) ([]models.Channel, int, error) {
	if filter.Limit <= 0 || filter.Limit > 50000 {
		filter.Limit = 50
	}
	if filter.Page <= 0 {
		filter.Page = 1
	}

	query := d.Model(&models.Channel{})
	if filter.Country != "" {
		query = query.Where("country = ?", filter.Country)
	}
	if filter.Category != "" {
		query = query.Where("categories LIKE ?", "%"+filter.Category+"%")
	}
	if filter.QueryTerm != "" {
		query = query.Where("name LIKE ?", "%"+filter.QueryTerm+"%")
	}

	var totalCount int64
	query.Count(&totalCount)

	offset := (filter.Page - 1) * filter.Limit
	var channels []models.Channel
	err := query.Order("name ASC").Limit(filter.Limit).Offset(offset).Find(&channels).Error
	return channels, int(totalCount), err
}

func (d *DB) GetChannel(id string) (*models.Channel, error) {
	var c models.Channel
	if err := d.First(&c, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &c, nil
}

func (d *DB) ListChannelCountries() ([]models.ChannelCountry, error) {
	var countries []models.ChannelCountry
	err := d.Model(&models.ChannelCountry{}).
		Select("channel_countries.code, channel_countries.name, COALESCE(channel_countries.flag, '') as flag, COUNT(channels.id) as channel_count").
		Joins("LEFT JOIN channels ON channels.country = channel_countries.code").
		Group("channel_countries.code, channel_countries.name, channel_countries.flag").
		Having("COUNT(channels.id) > 0").
		Order("channel_countries.name").
		Find(&countries).Error
	return countries, err
}

func (d *DB) ListChannelCategories() ([]models.ChannelCategory, error) {
	var categories []models.ChannelCategory
	err := d.Order("name").Find(&categories).Error
	return categories, err
}

func (d *DB) GetChannelsByCountry(countryCode string, limit int) ([]models.Channel, error) {
	if limit <= 0 {
		limit = 50
	}
	var channels []models.Channel
	err := d.Where("country = ?", countryCode).Order("name").Limit(limit).Find(&channels).Error
	return channels, err
}

func (d *DB) UpsertChannel(ch *models.Channel) error {
	return d.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		DoUpdates: clause.AssignmentColumns([]string{"name", "country", "languages", "categories", "logo", "stream_url", "is_nsfw", "website", "updated_at"}),
	}).Create(ch).Error
}

func (d *DB) UpsertChannelCountry(c *models.ChannelCountry) error {
	return d.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "code"}},
		DoUpdates: clause.AssignmentColumns([]string{"name", "flag"}),
	}).Create(c).Error
}

func (d *DB) UpsertChannelCategory(c *models.ChannelCategory) error {
	return d.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		DoUpdates: clause.AssignmentColumns([]string{"name"}),
	}).Create(c).Error
}

func (d *DB) ClearChannels() error {
	return d.Where("1 = 1").Delete(&models.Channel{}).Error
}

func (d *DB) CountChannels() (int, error) {
	var count int64
	err := d.Model(&models.Channel{}).Count(&count).Error
	return int(count), err
}

func (d *DB) DeleteChannel(id string) error {
	return d.Where("id = ?", id).Delete(&models.Channel{}).Error
}

func (d *DB) UpsertEPG(epg *models.ChannelEPG) error {
	return d.Create(epg).Error
}

func (d *DB) GetEPG(channelID string) ([]models.ChannelEPG, error) {
	var epgs []models.ChannelEPG
	err := d.Where("channel_id = ? AND end_time >= ?", channelID, time.Now().Format("2006-01-02 15:04:05")).
		Order("start_time ASC").Limit(50).Find(&epgs).Error
	return epgs, err
}

func (d *DB) ClearEPG() error {
	return d.Where("1 = 1").Delete(&models.ChannelEPG{}).Error
}

func (d *DB) GetChannelStats() (map[string]int, error) {
	stats := make(map[string]int)
	var count int64
	d.Model(&models.Channel{}).Count(&count)
	stats["channels"] = int(count)
	d.Model(&models.Channel{}).Where("country != ''").Distinct("country").Count(&count)
	stats["countries"] = int(count)
	d.Model(&models.ChannelCategory{}).Count(&count)
	stats["categories"] = int(count)
	d.Model(&models.Channel{}).Where("stream_url != '' AND stream_url IS NOT NULL").Count(&count)
	stats["with_streams"] = int(count)
	d.Model(&models.ChannelBlocklist{}).Count(&count)
	stats["blocklisted"] = int(count)
	return stats, nil
}

func (d *DB) GetAllChannelsWithStreams() ([]models.Channel, error) {
	var channels []models.Channel
	err := d.Where("stream_url IS NOT NULL AND stream_url != ''").Find(&channels).Error
	return channels, err
}

func (d *DB) UpdateChannelStream(channelID, streamURL string) error {
	return d.Model(&models.Channel{}).Where("id = ?", channelID).
		Updates(map[string]interface{}{"stream_url": streamURL, "updated_at": time.Now()}).Error
}

func (d *DB) AddToBlocklist(channelID, reason string) error {
	bl := models.ChannelBlocklist{ChannelID: channelID, Reason: reason}
	return d.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "channel_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"reason", "blocked_at"}),
	}).Create(&bl).Error
}

func (d *DB) IsBlocklisted(channelID string) bool {
	var count int64
	d.Model(&models.ChannelBlocklist{}).Where("channel_id = ?", channelID).Count(&count)
	return count > 0
}

func (d *DB) GetBlocklistCount() int {
	var count int64
	d.Model(&models.ChannelBlocklist{}).Count(&count)
	return int(count)
}

func (d *DB) ClearBlocklist() error {
	return d.Where("1 = 1").Delete(&models.ChannelBlocklist{}).Error
}

func (d *DB) GetBlocklistedIDs() map[string]bool {
	result := make(map[string]bool)
	var ids []string
	d.Model(&models.ChannelBlocklist{}).Pluck("channel_id", &ids)
	for _, id := range ids {
		result[id] = true
	}
	return result
}

func (d *DB) GetChannelCountByCategory() ([]models.ChannelCategory, error) {
	cats, err := d.ListChannelCategories()
	if err != nil {
		return nil, err
	}
	for i, cat := range cats {
		var count int64
		d.Model(&models.Channel{}).Where("categories LIKE ?", fmt.Sprintf("%%%s%%", cat.ID)).Count(&count)
		cats[i].ChannelCount = int(count)
	}
	return cats, nil
}
