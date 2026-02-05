package database

import (
	"database/sql"
	"encoding/json"
	"strings"

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
	if filter.Limit <= 0 || filter.Limit > 100 {
		filter.Limit = 50
	}
	if filter.Page <= 0 {
		filter.Page = 1
	}

	// Build WHERE clause
	var conditions []string
	var args []interface{}
	argIndex := 1

	if filter.Country != "" {
		conditions = append(conditions, "country = $"+string('0'+argIndex))
		args = append(args, filter.Country)
		argIndex++
	}

	if filter.Category != "" {
		conditions = append(conditions, "categories LIKE $"+string('0'+argIndex))
		args = append(args, "%"+filter.Category+"%")
		argIndex++
	}

	if filter.QueryTerm != "" {
		conditions = append(conditions, "name LIKE $"+string('0'+argIndex))
		args = append(args, "%"+filter.QueryTerm+"%")
		argIndex++
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = " WHERE " + strings.Join(conditions, " AND ")
	}

	// Get total count
	var totalCount int
	countQuery := "SELECT COUNT(*) FROM channels" + whereClause
	if err := d.QueryRow(countQuery, args...).Scan(&totalCount); err != nil {
		return nil, 0, err
	}

	// Get channels
	offset := (filter.Page - 1) * filter.Limit
	query := `SELECT id, name, country, languages, categories, logo, stream_url
		FROM channels` + whereClause + ` ORDER BY name ASC LIMIT $` + string('0'+argIndex) + ` OFFSET $` + string('0'+argIndex+1)
	args = append(args, filter.Limit, offset)

	rows, err := d.Query(query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var channels []models.Channel
	for rows.Next() {
		var c models.Channel
		var languages, categories, logo, streamURL sql.NullString

		err := rows.Scan(&c.ID, &c.Name, &c.Country, &languages, &categories, &logo, &streamURL)
		if err != nil {
			continue
		}

		c.Logo = logo.String
		c.StreamURL = streamURL.String

		// Parse JSON arrays
		if languages.String != "" {
			json.Unmarshal([]byte(languages.String), &c.Languages)
		}
		if categories.String != "" {
			json.Unmarshal([]byte(categories.String), &c.Categories)
		}

		channels = append(channels, c)
	}

	return channels, totalCount, nil
}

func (d *DB) GetChannel(id string) (*models.Channel, error) {
	var c models.Channel
	var languages, categories, logo, streamURL sql.NullString

	err := d.QueryRow(`
		SELECT id, name, country, languages, categories, logo, stream_url
		FROM channels WHERE id = $1
	`, id).Scan(&c.ID, &c.Name, &c.Country, &languages, &categories, &logo, &streamURL)
	if err != nil {
		return nil, err
	}

	c.Logo = logo.String
	c.StreamURL = streamURL.String

	if languages.String != "" {
		json.Unmarshal([]byte(languages.String), &c.Languages)
	}
	if categories.String != "" {
		json.Unmarshal([]byte(categories.String), &c.Categories)
	}

	return &c, nil
}

func (d *DB) ListChannelCountries() ([]models.ChannelCountry, error) {
	rows, err := d.Query(`
		SELECT code, name, COALESCE(flag, '') FROM channel_countries ORDER BY name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var countries []models.ChannelCountry
	for rows.Next() {
		var c models.ChannelCountry
		if err := rows.Scan(&c.Code, &c.Name, &c.Flag); err != nil {
			continue
		}
		countries = append(countries, c)
	}

	return countries, nil
}

func (d *DB) ListChannelCategories() ([]models.ChannelCategory, error) {
	rows, err := d.Query(`SELECT id, name FROM channel_categories ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var categories []models.ChannelCategory
	for rows.Next() {
		var c models.ChannelCategory
		if err := rows.Scan(&c.ID, &c.Name); err != nil {
			continue
		}
		categories = append(categories, c)
	}

	return categories, nil
}

func (d *DB) GetChannelsByCountry(countryCode string, limit int) ([]models.Channel, error) {
	if limit <= 0 {
		limit = 50
	}

	rows, err := d.Query(`
		SELECT id, name, country, languages, categories, logo, stream_url
		FROM channels WHERE country = $1 ORDER BY name LIMIT $2
	`, countryCode, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var channels []models.Channel
	for rows.Next() {
		var c models.Channel
		var languages, categories, logo, streamURL sql.NullString

		err := rows.Scan(&c.ID, &c.Name, &c.Country, &languages, &categories, &logo, &streamURL)
		if err != nil {
			continue
		}

		c.Logo = logo.String
		c.StreamURL = streamURL.String

		if languages.String != "" {
			json.Unmarshal([]byte(languages.String), &c.Languages)
		}
		if categories.String != "" {
			json.Unmarshal([]byte(categories.String), &c.Categories)
		}

		channels = append(channels, c)
	}

	return channels, nil
}
