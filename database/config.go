package database

import (
	"gorm.io/gorm/clause"

	"torrent-server/models"
)

func (d *DB) ListServices() ([]models.ServiceConfig, error) {
	var services []models.ServiceConfig
	err := d.Order("display_order").Find(&services).Error
	return services, err
}

func (d *DB) UpdateService(s *models.ServiceConfig) error {
	return d.Save(s).Error
}

func (d *DB) CreateService(s *models.ServiceConfig) error {
	return d.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		DoUpdates: clause.AssignmentColumns([]string{"label", "enabled", "icon", "display_order"}),
	}).Create(s).Error
}

func (d *DB) DeleteService(id string) error {
	return d.Where("id = ?", id).Delete(&models.ServiceConfig{}).Error
}
