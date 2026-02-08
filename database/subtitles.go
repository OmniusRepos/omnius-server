package database

import (
	"database/sql"
	"fmt"
	"strings"

	"torrent-server/models"
)

func (d *DB) GetSubtitlesByIMDB(imdbCode, language string) ([]models.StoredSubtitle, error) {
	return d.GetSubtitlesByIMDBEpisode(imdbCode, language, 0, 0)
}

func (d *DB) GetSubtitlesByIMDBEpisode(imdbCode, language string, season, episode int) ([]models.StoredSubtitle, error) {
	query := "SELECT id, imdb_code, language, language_name, release_name, hearing_impaired, source, season_number, episode_number, created_at FROM subtitles WHERE imdb_code = ?"
	args := []interface{}{imdbCode}

	if season > 0 {
		query += " AND season_number = ?"
		args = append(args, season)
	}
	if episode > 0 {
		query += " AND episode_number = ?"
		args = append(args, episode)
	}

	if language != "" {
		langs := strings.Split(language, ",")
		placeholders := make([]string, len(langs))
		for i, l := range langs {
			placeholders[i] = "?"
			args = append(args, strings.TrimSpace(l))
		}
		query += " AND language IN (" + strings.Join(placeholders, ",") + ")"
	}

	query += " ORDER BY season_number, episode_number, created_at DESC"

	rows, err := d.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query subtitles: %w", err)
	}
	defer rows.Close()

	var subtitles []models.StoredSubtitle
	for rows.Next() {
		var sub models.StoredSubtitle
		var hi int
		if err := rows.Scan(&sub.ID, &sub.ImdbCode, &sub.Language, &sub.LanguageName, &sub.ReleaseName, &hi, &sub.Source, &sub.SeasonNumber, &sub.EpisodeNumber, &sub.CreatedAt); err != nil {
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
	var vttPath sql.NullString
	err := d.QueryRow(
		"SELECT id, imdb_code, language, language_name, release_name, hearing_impaired, source, vtt_content, vtt_path, created_at FROM subtitles WHERE id = ?",
		id,
	).Scan(&sub.ID, &sub.ImdbCode, &sub.Language, &sub.LanguageName, &sub.ReleaseName, &hi, &sub.Source, &sub.VTTContent, &vttPath, &sub.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("subtitle not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get subtitle: %w", err)
	}
	sub.HearingImpaired = hi == 1
	if vttPath.Valid {
		sub.VTTPath = vttPath.String
	}
	return &sub, nil
}

func (d *DB) CreateSubtitle(sub *models.StoredSubtitle) error {
	hi := 0
	if sub.HearingImpaired {
		hi = 1
	}
	result, err := d.Exec(
		`INSERT INTO subtitles (imdb_code, language, language_name, release_name, hearing_impaired, source, vtt_content, vtt_path, season_number, episode_number)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(imdb_code, language, release_name) DO NOTHING`,
		sub.ImdbCode, sub.Language, sub.LanguageName, sub.ReleaseName, hi, sub.Source, sub.VTTContent, sub.VTTPath, sub.SeasonNumber, sub.EpisodeNumber,
	)
	if err != nil {
		return fmt.Errorf("failed to create subtitle: %w", err)
	}
	id, _ := result.LastInsertId()
	sub.ID = uint(id)
	return nil
}

func (d *DB) UpdateSubtitlePath(id uint, vttPath string) error {
	_, err := d.Exec("UPDATE subtitles SET vtt_path = ?, vtt_content = '' WHERE id = ?", vttPath, id)
	return err
}

// GetSubtitlesWithContent returns subtitles that still have vtt_content in DB (for migration).
func (d *DB) GetSubtitlesWithContent() ([]models.StoredSubtitle, error) {
	rows, err := d.Query("SELECT id, imdb_code, vtt_content FROM subtitles WHERE vtt_content != '' AND (vtt_path = '' OR vtt_path IS NULL)")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var subs []models.StoredSubtitle
	for rows.Next() {
		var sub models.StoredSubtitle
		if err := rows.Scan(&sub.ID, &sub.ImdbCode, &sub.VTTContent); err != nil {
			return nil, err
		}
		subs = append(subs, sub)
	}
	return subs, nil
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

func (d *DB) CountSubtitlesByIMDBEpisode(imdbCode string, season, episode int) (int, error) {
	var count int
	err := d.QueryRow("SELECT COUNT(*) FROM subtitles WHERE imdb_code = ? AND season_number = ? AND episode_number = ?", imdbCode, season, episode).Scan(&count)
	return count, err
}
