package database

import (
	"database/sql"
	"time"

	"torrent-server/models"
)

// --- License CRUD ---

func (d *DB) CreateLicense(l *models.License) (int64, error) {
	res, err := d.Exec(`
		INSERT INTO licenses (license_key, plan, owner_email, owner_name, max_deployments, is_active, notes, features, expires_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		l.LicenseKey, l.Plan, l.OwnerEmail, l.OwnerName, l.MaxDeployments, l.IsActive, l.Notes, l.Features, l.ExpiresAt,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (d *DB) GetLicenseByKey(key string) (*models.License, error) {
	l := &models.License{}
	var expiresAt, revokedAt sql.NullTime
	var isActive int
	var features sql.NullString
	err := d.QueryRow(`
		SELECT id, license_key, plan, owner_email, owner_name, max_deployments, is_active, notes, features, created_at, expires_at, revoked_at
		FROM licenses WHERE license_key = ?`, key,
	).Scan(&l.ID, &l.LicenseKey, &l.Plan, &l.OwnerEmail, &l.OwnerName, &l.MaxDeployments, &isActive, &l.Notes, &features, &l.CreatedAt, &expiresAt, &revokedAt)
	if err != nil {
		return nil, err
	}
	l.IsActive = isActive == 1
	l.Features = features.String
	if expiresAt.Valid {
		l.ExpiresAt = &expiresAt.Time
	}
	if revokedAt.Valid {
		l.RevokedAt = &revokedAt.Time
	}
	return l, nil
}

func (d *DB) GetLicenseByID(id int64) (*models.License, error) {
	l := &models.License{}
	var expiresAt, revokedAt sql.NullTime
	var isActive int
	var features sql.NullString
	err := d.QueryRow(`
		SELECT id, license_key, plan, owner_email, owner_name, max_deployments, is_active, notes, features, created_at, expires_at, revoked_at
		FROM licenses WHERE id = ?`, id,
	).Scan(&l.ID, &l.LicenseKey, &l.Plan, &l.OwnerEmail, &l.OwnerName, &l.MaxDeployments, &isActive, &l.Notes, &features, &l.CreatedAt, &expiresAt, &revokedAt)
	if err != nil {
		return nil, err
	}
	l.IsActive = isActive == 1
	l.Features = features.String
	if expiresAt.Valid {
		l.ExpiresAt = &expiresAt.Time
	}
	if revokedAt.Valid {
		l.RevokedAt = &revokedAt.Time
	}
	return l, nil
}

func (d *DB) ListLicenses() ([]models.License, error) {
	rows, err := d.Query(`
		SELECT l.id, l.license_key, l.plan, l.owner_email, l.owner_name, l.max_deployments, l.is_active, l.notes, l.features, l.created_at, l.expires_at, l.revoked_at,
		       COALESCE((SELECT COUNT(*) FROM license_deployments WHERE license_id = l.id AND is_active = 1), 0) AS active_deployments
		FROM licenses l ORDER BY l.created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var licenses []models.License
	for rows.Next() {
		var l models.License
		var expiresAt, revokedAt sql.NullTime
		var isActive int
		var features sql.NullString
		if err := rows.Scan(&l.ID, &l.LicenseKey, &l.Plan, &l.OwnerEmail, &l.OwnerName, &l.MaxDeployments, &isActive, &l.Notes, &features, &l.CreatedAt, &expiresAt, &revokedAt, &l.ActiveDeployments); err != nil {
			continue
		}
		l.IsActive = isActive == 1
		l.Features = features.String
		if expiresAt.Valid {
			l.ExpiresAt = &expiresAt.Time
		}
		if revokedAt.Valid {
			l.RevokedAt = &revokedAt.Time
		}
		licenses = append(licenses, l)
	}
	return licenses, nil
}

func (d *DB) UpdateLicense(id int64, req *models.AdminUpdateLicenseRequest) error {
	if req.Plan != nil {
		d.Exec("UPDATE licenses SET plan = ? WHERE id = ?", *req.Plan, id)
	}
	if req.OwnerEmail != nil {
		d.Exec("UPDATE licenses SET owner_email = ? WHERE id = ?", *req.OwnerEmail, id)
	}
	if req.OwnerName != nil {
		d.Exec("UPDATE licenses SET owner_name = ? WHERE id = ?", *req.OwnerName, id)
	}
	if req.MaxDeployments != nil {
		d.Exec("UPDATE licenses SET max_deployments = ? WHERE id = ?", *req.MaxDeployments, id)
	}
	if req.IsActive != nil {
		active := 0
		if *req.IsActive {
			active = 1
		}
		d.Exec("UPDATE licenses SET is_active = ? WHERE id = ?", active, id)
		if !*req.IsActive {
			now := time.Now()
			d.Exec("UPDATE licenses SET revoked_at = ? WHERE id = ?", now, id)
		} else {
			d.Exec("UPDATE licenses SET revoked_at = NULL WHERE id = ?", id)
		}
	}
	if req.Notes != nil {
		d.Exec("UPDATE licenses SET notes = ? WHERE id = ?", *req.Notes, id)
	}
	if req.ExpiresAt != nil {
		if *req.ExpiresAt == "" {
			d.Exec("UPDATE licenses SET expires_at = NULL WHERE id = ?", id)
		} else {
			d.Exec("UPDATE licenses SET expires_at = ? WHERE id = ?", *req.ExpiresAt, id)
		}
	}
	return nil
}

// GetLicenseByPaddleTransaction finds a license by Paddle transaction ID in the notes field (for idempotency)
func (d *DB) GetLicenseByPaddleTransaction(txnID string) (*models.License, error) {
	l := &models.License{}
	var expiresAt, revokedAt sql.NullTime
	var isActive int
	var features sql.NullString
	err := d.QueryRow(`
		SELECT id, license_key, plan, owner_email, owner_name, max_deployments, is_active, notes, features, created_at, expires_at, revoked_at
		FROM licenses WHERE notes LIKE ?`, "%Paddle transaction: "+txnID+"%",
	).Scan(&l.ID, &l.LicenseKey, &l.Plan, &l.OwnerEmail, &l.OwnerName, &l.MaxDeployments, &isActive, &l.Notes, &features, &l.CreatedAt, &expiresAt, &revokedAt)
	if err != nil {
		return nil, err
	}
	l.IsActive = isActive == 1
	l.Features = features.String
	if expiresAt.Valid {
		l.ExpiresAt = &expiresAt.Time
	}
	if revokedAt.Valid {
		l.RevokedAt = &revokedAt.Time
	}
	return l, nil
}

// GetLicenseByEmail returns the most recent active license for a given email
func (d *DB) GetLicenseByEmail(email string) (*models.License, error) {
	l := &models.License{}
	var expiresAt, revokedAt sql.NullTime
	var isActive int
	var features sql.NullString
	err := d.QueryRow(`
		SELECT id, license_key, plan, owner_email, owner_name, max_deployments, is_active, notes, features, created_at, expires_at, revoked_at
		FROM licenses WHERE owner_email = ? AND is_active = 1 ORDER BY created_at DESC LIMIT 1`, email,
	).Scan(&l.ID, &l.LicenseKey, &l.Plan, &l.OwnerEmail, &l.OwnerName, &l.MaxDeployments, &isActive, &l.Notes, &features, &l.CreatedAt, &expiresAt, &revokedAt)
	if err != nil {
		return nil, err
	}
	l.IsActive = isActive == 1
	l.Features = features.String
	if expiresAt.Valid {
		l.ExpiresAt = &expiresAt.Time
	}
	if revokedAt.Valid {
		l.RevokedAt = &revokedAt.Time
	}
	return l, nil
}

func (d *DB) DeleteLicense(id int64) error {
	_, err := d.Exec("DELETE FROM licenses WHERE id = ?", id)
	return err
}

// --- Deployment CRUD ---

func (d *DB) UpsertDeployment(licenseID int64, fingerprint, label, ip, version string) (*models.LicenseDeployment, error) {
	_, err := d.Exec(`
		INSERT INTO license_deployments (license_id, machine_fingerprint, machine_label, ip_address, server_version)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(license_id, machine_fingerprint) DO UPDATE SET
			machine_label = excluded.machine_label,
			ip_address = excluded.ip_address,
			server_version = excluded.server_version,
			last_heartbeat = CURRENT_TIMESTAMP,
			is_active = 1`,
		licenseID, fingerprint, label, ip, version,
	)
	if err != nil {
		return nil, err
	}

	dep := &models.LicenseDeployment{}
	var isActive int
	err = d.QueryRow(`
		SELECT id, license_id, machine_fingerprint, machine_label, ip_address, server_version, first_seen, last_heartbeat, is_active
		FROM license_deployments WHERE license_id = ? AND machine_fingerprint = ?`,
		licenseID, fingerprint,
	).Scan(&dep.ID, &dep.LicenseID, &dep.MachineFingerprint, &dep.MachineLabel, &dep.IPAddress, &dep.ServerVersion, &dep.FirstSeen, &dep.LastHeartbeat, &isActive)
	if err != nil {
		return nil, err
	}
	dep.IsActive = isActive == 1
	return dep, nil
}

func (d *DB) CountActiveDeployments(licenseID int64) (int, error) {
	var count int
	err := d.QueryRow("SELECT COUNT(*) FROM license_deployments WHERE license_id = ? AND is_active = 1", licenseID).Scan(&count)
	return count, err
}

func (d *DB) GetDeploymentsByLicense(licenseID int64) ([]models.LicenseDeployment, error) {
	rows, err := d.Query(`
		SELECT id, license_id, machine_fingerprint, machine_label, ip_address, server_version, first_seen, last_heartbeat, is_active
		FROM license_deployments WHERE license_id = ? ORDER BY last_heartbeat DESC`, licenseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var deps []models.LicenseDeployment
	for rows.Next() {
		var dep models.LicenseDeployment
		var isActive int
		if err := rows.Scan(&dep.ID, &dep.LicenseID, &dep.MachineFingerprint, &dep.MachineLabel, &dep.IPAddress, &dep.ServerVersion, &dep.FirstSeen, &dep.LastHeartbeat, &isActive); err != nil {
			continue
		}
		dep.IsActive = isActive == 1
		deps = append(deps, dep)
	}
	return deps, nil
}

func (d *DB) UpdateDeploymentHeartbeat(licenseID int64, fingerprint, ip, version string) error {
	_, err := d.Exec(`
		UPDATE license_deployments SET last_heartbeat = CURRENT_TIMESTAMP, ip_address = ?, server_version = ?, is_active = 1
		WHERE license_id = ? AND machine_fingerprint = ?`,
		ip, version, licenseID, fingerprint)
	return err
}

func (d *DB) DeactivateDeployment(licenseID int64, fingerprint string) error {
	_, err := d.Exec("UPDATE license_deployments SET is_active = 0 WHERE license_id = ? AND machine_fingerprint = ?", licenseID, fingerprint)
	return err
}

func (d *DB) DeactivateDeploymentByID(deploymentID int64) error {
	_, err := d.Exec("UPDATE license_deployments SET is_active = 0 WHERE id = ?", deploymentID)
	return err
}

// MarkStaleDeployments marks deployments as inactive if no heartbeat in the given duration.
func (d *DB) MarkStaleDeployments(staleDuration time.Duration) (int64, error) {
	cutoff := time.Now().Add(-staleDuration)
	res, err := d.Exec("UPDATE license_deployments SET is_active = 0 WHERE is_active = 1 AND last_heartbeat < ?", cutoff)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// IsDeploymentActive checks if a specific deployment exists and is active
func (d *DB) IsDeploymentActive(licenseID int64, fingerprint string) (bool, error) {
	var isActive int
	err := d.QueryRow("SELECT is_active FROM license_deployments WHERE license_id = ? AND machine_fingerprint = ?", licenseID, fingerprint).Scan(&isActive)
	if err != nil {
		return false, err
	}
	return isActive == 1, nil
}

// --- Events ---

func (d *DB) LogLicenseEvent(licenseID int64, eventType, fingerprint, ip, details string) error {
	_, err := d.Exec(`
		INSERT INTO license_events (license_id, event_type, machine_fingerprint, ip_address, details)
		VALUES (?, ?, ?, ?, ?)`,
		licenseID, eventType, fingerprint, ip, details)
	return err
}

func (d *DB) GetLicenseEvents(licenseID int64, limit int) ([]models.LicenseEvent, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := d.Query(`
		SELECT id, license_id, event_type, machine_fingerprint, ip_address, details, created_at
		FROM license_events WHERE license_id = ? ORDER BY created_at DESC LIMIT ?`, licenseID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []models.LicenseEvent
	for rows.Next() {
		var e models.LicenseEvent
		var fingerprint, ip, details sql.NullString
		if err := rows.Scan(&e.ID, &e.LicenseID, &e.EventType, &fingerprint, &ip, &details, &e.CreatedAt); err != nil {
			continue
		}
		e.MachineFingerprint = fingerprint.String
		e.IPAddress = ip.String
		e.Details = details.String
		events = append(events, e)
	}
	return events, nil
}
