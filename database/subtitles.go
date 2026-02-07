package database

import (
	"database/sql"
	"fmt"

	"torrent-server/models"
)

func (d *DB) GetSubtitlesByIMDB(imdbCode, language string) ([]models.StoredSubtitle, error) {
	query := "SELECT id, imdb_code, language, language_name, release_name, hearing_impaired, source, created_at FROM subtitles WHERE imdb_code = ?"
	args := []interface{}{imdbCode}

	if language != "" {
		query += " AND language = ?"
		args = append(args, language)
	}

	query += " ORDER BY created_at DESC"

	rows, err := d.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query subtitles: %w", err)
	}
	defer rows.Close()

	var subtitles []models.StoredSubtitle
	for rows.Next() {
		var sub models.StoredSubtitle
		var hi int
		if err := rows.Scan(&sub.ID, &sub.ImdbCode, &sub.Language, &sub.LanguageName, &sub.ReleaseName, &hi, &sub.Source, &sub.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan subtitle: %w", err)
		}
		sub.HearingImpaired = hi == 1
		subtitles = append(subtitles, sub)
	}
	return subtitles, nil
}

func (d *DB) GetSubtitleByID(id uint) (*models.StoredSubtitle, error) {
	var sub models.StoredSubtitle
	var hi int
	err := d.QueryRow(
		"SELECT id, imdb_code, language, language_name, release_name, hearing_impaired, source, vtt_content, created_at FROM subtitles WHERE id = ?",
		id,
	).Scan(&sub.ID, &sub.ImdbCode, &sub.Language, &sub.LanguageName, &sub.ReleaseName, &hi, &sub.Source, &sub.VTTContent, &sub.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("subtitle not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get subtitle: %w", err)
	}
	sub.HearingImpaired = hi == 1
	return &sub, nil
}

func (d *DB) CreateSubtitle(sub *models.StoredSubtitle) error {
	hi := 0
	if sub.HearingImpaired {
		hi = 1
	}
	result, err := d.Exec(
		`INSERT INTO subtitles (imdb_code, language, language_name, release_name, hearing_impaired, source, vtt_content)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(imdb_code, language, release_name) DO NOTHING`,
		sub.ImdbCode, sub.Language, sub.LanguageName, sub.ReleaseName, hi, sub.Source, sub.VTTContent,
	)
	if err != nil {
		return fmt.Errorf("failed to create subtitle: %w", err)
	}
	id, _ := result.LastInsertId()
	sub.ID = uint(id)
	return nil
}

func (d *DB) DeleteSubtitle(id uint) error {
	_, err := d.Exec("DELETE FROM subtitles WHERE id = ?", id)
	return err
}

func (d *DB) DeleteSubtitlesByIMDB(imdbCode string) error {
	_, err := d.Exec("DELETE FROM subtitles WHERE imdb_code = ?", imdbCode)
	return err
}

func (d *DB) CountSubtitlesByIMDB(imdbCode string) (int, error) {
	var count int
	err := d.QueryRow("SELECT COUNT(*) FROM subtitles WHERE imdb_code = ?", imdbCode).Scan(&count)
	return count, err
}
