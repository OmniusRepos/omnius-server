package database

import (
	"torrent-server/models"
)

func (d *DB) ListServices() ([]models.ServiceConfig, error) {
	rows, err := d.Query("SELECT id, label, enabled, COALESCE(icon, ''), display_order FROM service_config ORDER BY display_order")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var services []models.ServiceConfig
	for rows.Next() {
		var s models.ServiceConfig
		var enabled int
		if err := rows.Scan(&s.ID, &s.Label, &enabled, &s.Icon, &s.DisplayOrder); err != nil {
			continue
		}
		s.Enabled = enabled == 1
		services = append(services, s)
	}
	return services, nil
}

func (d *DB) UpdateService(s *models.ServiceConfig) error {
	enabled := 0
	if s.Enabled {
		enabled = 1
	}
	_, err := d.Exec(
		"UPDATE service_config SET label = ?, enabled = ?, icon = ?, display_order = ? WHERE id = ?",
		s.Label, enabled, s.Icon, s.DisplayOrder, s.ID,
	)
	return err
}

func (d *DB) CreateService(s *models.ServiceConfig) error {
	enabled := 0
	if s.Enabled {
		enabled = 1
	}
	_, err := d.Exec(
		`INSERT INTO service_config (id, label, enabled, icon, display_order) VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET label = excluded.label, enabled = excluded.enabled, icon = excluded.icon, display_order = excluded.display_order`,
		s.ID, s.Label, enabled, s.Icon, s.DisplayOrder,
	)
	return err
}

func (d *DB) DeleteService(id string) error {
	_, err := d.Exec("DELETE FROM service_config WHERE id = ?", id)
	return err
}
