package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
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
	if filter.Limit <= 0 || filter.Limit > 50000 {
		filter.Limit = 50
	}
	if filter.Page <= 0 {
		filter.Page = 1
	}

	var conditions []string
	var args []any

	if filter.Country != "" {
		conditions = append(conditions, "country = ?")
		args = append(args, filter.Country)
	}

	if filter.Category != "" {
		conditions = append(conditions, "categories LIKE ?")
		args = append(args, "%"+filter.Category+"%")
	}

	if filter.QueryTerm != "" {
		conditions = append(conditions, "name LIKE ?")
		args = append(args, "%"+filter.QueryTerm+"%")
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = " WHERE " + strings.Join(conditions, " AND ")
	}

	var totalCount int
	countQuery := "SELECT COUNT(*) FROM channels" + whereClause
	if err := d.QueryRow(countQuery, args...).Scan(&totalCount); err != nil {
		return nil, 0, err
	}

	offset := (filter.Page - 1) * filter.Limit
	query := `SELECT id, name, country, languages, categories, logo, stream_url
		FROM channels` + whereClause + ` ORDER BY name ASC LIMIT ? OFFSET ?`
	args = append(args, filter.Limit, offset)

	rows, err := d.Query(query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var channels []models.Channel
	for rows.Next() {
		c := scanChannel(rows)
		if c != nil {
			channels = append(channels, *c)
		}
	}

	return channels, totalCount, nil
}

func (d *DB) GetChannel(id string) (*models.Channel, error) {
	var c models.Channel
	var languages, categories, logo, streamURL sql.NullString

	err := d.QueryRow(`
		SELECT id, name, country, languages, categories, logo, stream_url
		FROM channels WHERE id = ?
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
		SELECT cc.code, cc.name, COALESCE(cc.flag, ''), COUNT(ch.id) as channel_count
		FROM channel_countries cc
		LEFT JOIN channels ch ON ch.country = cc.code
		GROUP BY cc.code, cc.name, cc.flag
		HAVING channel_count > 0
		ORDER BY cc.name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var countries []models.ChannelCountry
	for rows.Next() {
		var c models.ChannelCountry
		if err := rows.Scan(&c.Code, &c.Name, &c.Flag, &c.ChannelCount); err != nil {
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
		FROM channels WHERE country = ? ORDER BY name LIMIT ?
	`, countryCode, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var channels []models.Channel
	for rows.Next() {
		c := scanChannel(rows)
		if c != nil {
			channels = append(channels, *c)
		}
	}

	return channels, nil
}

// --- Upsert methods for IPTV sync ---

func (d *DB) UpsertChannel(ch *models.Channel) error {
	languagesJSON, _ := json.Marshal(ch.Languages)
	categoriesJSON, _ := json.Marshal(ch.Categories)
	nsfw := 0
	if ch.IsNSFW {
		nsfw = 1
	}

	_, err := d.Exec(`
		INSERT INTO channels (id, name, country, languages, categories, logo, stream_url, is_nsfw, website, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(id) DO UPDATE SET
			name = excluded.name,
			country = excluded.country,
			languages = excluded.languages,
			categories = excluded.categories,
			logo = excluded.logo,
			stream_url = excluded.stream_url,
			is_nsfw = excluded.is_nsfw,
			website = excluded.website,
			updated_at = CURRENT_TIMESTAMP
	`, ch.ID, ch.Name, ch.Country, string(languagesJSON), string(categoriesJSON), ch.Logo, ch.StreamURL, nsfw, ch.Website)
	return err
}

func (d *DB) UpsertChannelCountry(c *models.ChannelCountry) error {
	_, err := d.Exec(`
		INSERT INTO channel_countries (code, name, flag)
		VALUES (?, ?, ?)
		ON CONFLICT(code) DO UPDATE SET name = excluded.name, flag = excluded.flag
	`, c.Code, c.Name, c.Flag)
	return err
}

func (d *DB) UpsertChannelCategory(c *models.ChannelCategory) error {
	_, err := d.Exec(`
		INSERT INTO channel_categories (id, name)
		VALUES (?, ?)
		ON CONFLICT(id) DO UPDATE SET name = excluded.name
	`, c.ID, c.Name)
	return err
}

func (d *DB) ClearChannels() error {
	_, err := d.Exec("DELETE FROM channels")
	return err
}

func (d *DB) CountChannels() (int, error) {
	var count int
	err := d.QueryRow("SELECT COUNT(*) FROM channels").Scan(&count)
	return count, err
}

func (d *DB) DeleteChannel(id string) error {
	_, err := d.Exec("DELETE FROM channels WHERE id = ?", id)
	return err
}

// --- EPG methods ---

func (d *DB) UpsertEPG(epg *models.ChannelEPG) error {
	_, err := d.Exec(`
		INSERT INTO channel_epg (channel_id, title, description, start_time, end_time)
		VALUES (?, ?, ?, ?, ?)
	`, epg.ChannelID, epg.Title, epg.Description, epg.StartTime, epg.EndTime)
	return err
}

func (d *DB) GetEPG(channelID string) ([]models.ChannelEPG, error) {
	rows, err := d.Query(`
		SELECT id, channel_id, title, COALESCE(description, ''), start_time, end_time
		FROM channel_epg
		WHERE channel_id = ? AND end_time >= datetime('now')
		ORDER BY start_time ASC
		LIMIT 50
	`, channelID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var epgs []models.ChannelEPG
	for rows.Next() {
		var e models.ChannelEPG
		if err := rows.Scan(&e.ID, &e.ChannelID, &e.Title, &e.Description, &e.StartTime, &e.EndTime); err != nil {
			continue
		}
		epgs = append(epgs, e)
	}
	return epgs, nil
}

func (d *DB) ClearEPG() error {
	_, err := d.Exec("DELETE FROM channel_epg")
	return err
}

func (d *DB) GetChannelStats() (map[string]int, error) {
	stats := make(map[string]int)
	var count int

	d.QueryRow("SELECT COUNT(*) FROM channels").Scan(&count)
	stats["channels"] = count
	d.QueryRow("SELECT COUNT(DISTINCT country) FROM channels WHERE country != ''").Scan(&count)
	stats["countries"] = count
	d.QueryRow("SELECT COUNT(*) FROM channel_categories").Scan(&count)
	stats["categories"] = count
	d.QueryRow("SELECT COUNT(*) FROM channels WHERE stream_url != '' AND stream_url IS NOT NULL").Scan(&count)
	stats["with_streams"] = count
	d.QueryRow("SELECT COUNT(*) FROM channel_blocklist").Scan(&count)
	stats["blocklisted"] = count

	return stats, nil
}

// GetAllChannelsWithStreams returns all channels that have a non-empty stream URL
func (d *DB) GetAllChannelsWithStreams() ([]models.Channel, error) {
	rows, err := d.Query(`
		SELECT id, name, country, languages, categories, logo, stream_url
		FROM channels
		WHERE stream_url IS NOT NULL AND stream_url != ''
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var channels []models.Channel
	for rows.Next() {
		c := scanChannel(rows)
		if c != nil {
			channels = append(channels, *c)
		}
	}
	return channels, nil
}

// --- Helper ---

func scanChannel(rows *sql.Rows) *models.Channel {
	var c models.Channel
	var languages, categories, logo, streamURL sql.NullString

	err := rows.Scan(&c.ID, &c.Name, &c.Country, &languages, &categories, &logo, &streamURL)
	if err != nil {
		return nil
	}

	c.Logo = logo.String
	c.StreamURL = streamURL.String
	if languages.String != "" {
		json.Unmarshal([]byte(languages.String), &c.Languages)
	}
	if categories.String != "" {
		json.Unmarshal([]byte(categories.String), &c.Categories)
	}

	return &c
}

// UpdateChannelStream updates only the stream_url for a channel
func (d *DB) UpdateChannelStream(channelID, streamURL string) error {
	_, err := d.Exec("UPDATE channels SET stream_url = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?", streamURL, channelID)
	return err
}

// --- Blocklist methods ---

// AddToBlocklist adds a channel ID to the blocklist
func (d *DB) AddToBlocklist(channelID, reason string) error {
	_, err := d.Exec(`
		INSERT INTO channel_blocklist (channel_id, reason, blocked_at)
		VALUES (?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(channel_id) DO UPDATE SET reason = excluded.reason, blocked_at = CURRENT_TIMESTAMP
	`, channelID, reason)
	return err
}

// IsBlocklisted checks if a channel ID is in the blocklist
func (d *DB) IsBlocklisted(channelID string) bool {
	var count int
	d.QueryRow("SELECT COUNT(*) FROM channel_blocklist WHERE channel_id = ?", channelID).Scan(&count)
	return count > 0
}

// GetBlocklistCount returns the number of blocklisted channels
func (d *DB) GetBlocklistCount() int {
	var count int
	d.QueryRow("SELECT COUNT(*) FROM channel_blocklist").Scan(&count)
	return count
}

// ClearBlocklist removes all entries from the blocklist
func (d *DB) ClearBlocklist() error {
	_, err := d.Exec("DELETE FROM channel_blocklist")
	return err
}

// GetBlocklistedIDs returns all blocklisted channel IDs as a map for fast lookup
func (d *DB) GetBlocklistedIDs() map[string]bool {
	result := make(map[string]bool)
	rows, err := d.Query("SELECT channel_id FROM channel_blocklist")
	if err != nil {
		return result
	}
	defer rows.Close()

	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err == nil {
			result[id] = true
		}
	}
	return result
}

// GetChannelCountByCategory returns the count of channels per category
func (d *DB) GetChannelCountByCategory() ([]models.ChannelCategory, error) {
	// First get all categories
	cats, err := d.ListChannelCategories()
	if err != nil {
		return nil, err
	}

	for i, cat := range cats {
		var count int
		d.QueryRow("SELECT COUNT(*) FROM channels WHERE categories LIKE ?", fmt.Sprintf("%%%s%%", cat.ID)).Scan(&count)
		cats[i].ChannelCount = count
	}

	return cats, nil
}
