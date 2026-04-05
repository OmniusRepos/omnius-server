package database

import (
	"gorm.io/gorm"

	"torrent-server/models"
)

func (d *DB) ListHomeSections(includeInactive bool) ([]models.HomeSection, error) {
	query := d.Model(&models.HomeSection{})
	if !includeInactive {
		query = query.Where("is_active = ?", true)
	}
	var sections []models.HomeSection
	err := query.Order("display_order ASC, id ASC").Find(&sections).Error
	return sections, err
}

func (d *DB) GetHomeSection(id uint) (*models.HomeSection, error) {
	var s models.HomeSection
	if err := d.First(&s, id).Error; err != nil {
		return nil, err
	}
	return &s, nil
}

func (d *DB) CreateHomeSection(s *models.HomeSection) error {
	if s.DisplayType == "" {
		s.DisplayType = "carousel"
	}
	return d.Create(s).Error
}

func (d *DB) UpdateHomeSection(s *models.HomeSection) error {
	if s.DisplayType == "" {
		s.DisplayType = "carousel"
	}
	return d.Save(s).Error
}

func (d *DB) DeleteHomeSection(id uint) error {
	return d.Delete(&models.HomeSection{}, id).Error
}

func (d *DB) ReorderHomeSections(ids []uint) error {
	return d.Transaction(func(tx *gorm.DB) error {
		for i, id := range ids {
			if err := tx.Model(&models.HomeSection{}).Where("id = ?", id).Update("display_order", i).Error; err != nil {
				return err
			}
		}
		return nil
	})
}
