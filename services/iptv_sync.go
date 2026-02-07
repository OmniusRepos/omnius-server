package services

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"torrent-server/database"
	"torrent-server/models"
)

// Default M3U source — configurable at runtime via admin settings
var IPTVM3UURL = "https://iptv-org.github.io/iptv/index.m3u"
var IPTVAPIBaseURL = "https://iptv-org.github.io/api"

type IPTVSyncService struct {
	db     *database.DB
	client *http.Client
	mu     sync.Mutex
	status IPTVSyncStatus
}

type IPTVSyncStatus struct {
	Running    bool   `json:"running"`
	Phase      string `json:"phase"`
	Progress   int    `json:"progress"`
	Total      int    `json:"total"`
	LastSync   string `json:"last_sync,omitempty"`
	LastError  string `json:"last_error,omitempty"`
	Channels   int    `json:"channels"`
	Countries  int    `json:"countries"`
	Categories int    `json:"categories"`
	M3UURL     string `json:"m3u_url"`
}

func NewIPTVSyncService(db *database.DB) *IPTVSyncService {
	return &IPTVSyncService{
		db:     db,
		client: &http.Client{Timeout: 120 * time.Second},
	}
}

func (s *IPTVSyncService) GetStatus() IPTVSyncStatus {
	s.mu.Lock()
	defer s.mu.Unlock()
	st := s.status
	st.M3UURL = IPTVM3UURL
	return st
}

// SyncFromM3U fetches an M3U URL and imports all channels
func (s *IPTVSyncService) SyncFromM3U(m3uURL string) error {
	s.mu.Lock()
	if s.status.Running {
		s.mu.Unlock()
		return fmt.Errorf("sync already in progress")
	}
	s.status = IPTVSyncStatus{Running: true, Phase: "starting"}
	s.mu.Unlock()

	if m3uURL != "" {
		IPTVM3UURL = m3uURL
	}

	go func() {
		err := s.doM3USync(IPTVM3UURL)
		s.mu.Lock()
		s.status.Running = false
		s.status.LastSync = time.Now().Format(time.RFC3339)
		if err != nil {
			s.status.LastError = err.Error()
			s.status.Phase = "error: " + err.Error()
		} else {
			s.status.Phase = "completed"
			s.status.LastError = ""
		}
		s.mu.Unlock()
	}()

	return nil
}

func (s *IPTVSyncService) doM3USync(m3uURL string) error {
	// 1. Fetch reference data from iptv-org API (countries/categories with names & flags)
	s.setPhase("fetching reference data", 0, 0)
	s.syncCountriesFromAPI()
	s.syncCategoriesFromAPI()

	// 2. Fetch and parse M3U
	s.setPhase("downloading M3U", 0, 0)
	log.Printf("[IPTV Sync] Fetching M3U from %s", m3uURL)

	resp, err := s.client.Get(m3uURL)
	if err != nil {
		return fmt.Errorf("failed to fetch M3U: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("M3U fetch returned HTTP %d", resp.StatusCode)
	}

	channels, err := parseM3U(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to parse M3U: %w", err)
	}

	log.Printf("[IPTV Sync] Parsed %d channels from M3U", len(channels))

	// 3. Store channels
	total := len(channels)
	stored := 0
	countrySeen := make(map[string]bool)
	categorySeen := make(map[string]bool)

	for i, ch := range channels {
		if i%500 == 0 {
			s.setPhase("storing channels", i, total)
		}

		if err := s.db.UpsertChannel(&ch); err == nil {
			stored++
		}

		// Track unique countries/categories
		if ch.Country != "" {
			countrySeen[ch.Country] = true
		}
		for _, cat := range ch.Categories {
			categorySeen[cat] = true
		}
	}

	// Auto-create any countries/categories not in reference data
	for code := range countrySeen {
		s.db.UpsertChannelCountry(&models.ChannelCountry{
			Code: strings.ToUpper(code),
			Name: strings.ToUpper(code), // will be overwritten if API data exists
		})
	}
	for cat := range categorySeen {
		s.db.UpsertChannelCategory(&models.ChannelCategory{
			ID:   strings.ToLower(cat),
			Name: cat,
		})
	}

	s.mu.Lock()
	s.status.Channels = stored
	s.status.Countries = len(countrySeen)
	s.status.Categories = len(categorySeen)
	s.mu.Unlock()

	log.Printf("[IPTV Sync] Stored %d channels, %d countries, %d categories",
		stored, len(countrySeen), len(categorySeen))

	return nil
}

func (s *IPTVSyncService) setPhase(phase string, progress, total int) {
	s.mu.Lock()
	s.status.Phase = phase
	s.status.Progress = progress
	s.status.Total = total
	s.mu.Unlock()
}

// --- M3U Parser ---
// Parses standard M3U/M3U8 with EXTINF metadata:
// #EXTINF:-1 tvg-id="ID" tvg-country="US" tvg-language="English" tvg-logo="URL" group-title="Category",Channel Name
// http://stream-url

func parseM3U(reader io.Reader) ([]models.Channel, error) {
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024) // 1MB buffer for long lines

	var channels []models.Channel
	var current *m3uEntry

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if line == "" || line == "#EXTM3U" {
			continue
		}

		if strings.HasPrefix(line, "#EXTINF:") {
			current = parseEXTINF(line)
			continue
		}

		// Skip other directives
		if strings.HasPrefix(line, "#") {
			continue
		}

		// This is a URL line
		if current != nil && (strings.HasPrefix(line, "http://") || strings.HasPrefix(line, "https://")) {
			id := current.tvgID
			if id == "" {
				// Generate ID from name
				id = strings.ReplaceAll(current.name, " ", "") + "." + strings.ToLower(current.country)
			}

			ch := models.Channel{
				ID:        id,
				Name:      current.name,
				Country:   strings.ToUpper(current.country),
				Logo:      current.logo,
				StreamURL: line,
			}

			if current.language != "" {
				ch.Languages = []string{current.language}
			}
			if current.group != "" {
				ch.Categories = []string{current.group}
			}

			channels = append(channels, ch)
			current = nil
		}
	}

	if err := scanner.Err(); err != nil {
		return channels, fmt.Errorf("scanner error: %w", err)
	}

	return channels, nil
}

type m3uEntry struct {
	tvgID    string
	name     string
	country  string
	language string
	logo     string
	group    string
}

func parseEXTINF(line string) *m3uEntry {
	entry := &m3uEntry{}

	// Extract name (after the last comma)
	if idx := strings.LastIndex(line, ","); idx >= 0 {
		entry.name = strings.TrimSpace(line[idx+1:])
		line = line[:idx]
	}

	// Extract attributes
	entry.tvgID = extractAttr(line, "tvg-id")
	entry.country = extractAttr(line, "tvg-country")
	entry.language = extractAttr(line, "tvg-language")
	entry.logo = extractAttr(line, "tvg-logo")
	entry.group = extractAttr(line, "group-title")

	return entry
}

func extractAttr(line, attr string) string {
	key := attr + `="`
	idx := strings.Index(line, key)
	if idx < 0 {
		return ""
	}
	start := idx + len(key)
	end := strings.Index(line[start:], `"`)
	if end < 0 {
		return ""
	}
	return line[start : start+end]
}

// --- iptv-org API reference data (for country names/flags and category names) ---

func (s *IPTVSyncService) syncCountriesFromAPI() {
	var countries []struct {
		Code string `json:"code"`
		Name string `json:"name"`
		Flag string `json:"flag"`
	}
	if err := s.fetchJSON("/countries.json", &countries); err != nil {
		log.Printf("[IPTV Sync] Could not fetch countries reference data: %v", err)
		return
	}
	for _, c := range countries {
		s.db.UpsertChannelCountry(&models.ChannelCountry{
			Code: c.Code,
			Name: c.Name,
			Flag: c.Flag,
		})
	}
	log.Printf("[IPTV Sync] Loaded %d country references", len(countries))
}

func (s *IPTVSyncService) syncCategoriesFromAPI() {
	var categories []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	if err := s.fetchJSON("/categories.json", &categories); err != nil {
		log.Printf("[IPTV Sync] Could not fetch categories reference data: %v", err)
		return
	}
	for _, c := range categories {
		s.db.UpsertChannelCategory(&models.ChannelCategory{
			ID:   c.ID,
			Name: c.Name,
		})
	}
	log.Printf("[IPTV Sync] Loaded %d category references", len(categories))
}

func (s *IPTVSyncService) fetchJSON(endpoint string, target any) error {
	url := IPTVAPIBaseURL + endpoint

	resp, err := s.client.Get(url)
	if err != nil {
		return fmt.Errorf("failed to fetch %s: %w", endpoint, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API returned %d for %s", resp.StatusCode, endpoint)
	}

	return json.NewDecoder(resp.Body).Decode(target)
}
