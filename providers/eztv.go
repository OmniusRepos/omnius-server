package providers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

type EZTVProvider struct {
	baseURL string
}

type eztvResponse struct {
	ImdbID       string        `json:"imdb_id"`
	TorrentsCount int          `json:"torrents_count"`
	Torrents     []eztvTorrent `json:"torrents"`
}

type eztvTorrent struct {
	ID            int    `json:"id"`
	Hash          string `json:"hash"`
	Filename      string `json:"filename"`
	Title         string `json:"title"`
	Season        string `json:"season"`
	Episode       string `json:"episode"`
	Seeds         int    `json:"seeds"`
	Peers         int    `json:"peers"`
	SizeBytes     string `json:"size_bytes"`
	MagnetURL     string `json:"magnet_url"`
	SmallScreenshot string `json:"small_screenshot"`
}

func NewEZTVProvider() *EZTVProvider {
	return &EZTVProvider{
		baseURL: "https://eztvx.to/api",
	}
}

func (p *EZTVProvider) Name() string {
	return "EZTV"
}

func (p *EZTVProvider) SearchMovie(title string, year int) ([]TorrentResult, error) {
	// EZTV is for TV shows only
	return nil, fmt.Errorf("EZTV does not support movies")
}

func (p *EZTVProvider) SearchSeries(title string, season, episode int) ([]TorrentResult, error) {
	params := url.Values{}
	params.Set("limit", "100")

	// EZTV requires IMDB ID, so we search by title pattern
	resp, err := http.Get(p.baseURL + "/get-torrents?" + params.Encode())
	if err != nil {
		return nil, fmt.Errorf("EZTV request failed: %w", err)
	}
	defer resp.Body.Close()

	var eztvResp eztvResponse
	if err := json.NewDecoder(resp.Body).Decode(&eztvResp); err != nil {
		return nil, fmt.Errorf("EZTV decode failed: %w", err)
	}

	var results []TorrentResult
	searchPattern := fmt.Sprintf("S%02dE%02d", season, episode)
	titleLower := strings.ToLower(title)

	for _, t := range eztvResp.Torrents {
		// Filter by title and episode
		if !strings.Contains(strings.ToLower(t.Title), titleLower) {
			continue
		}
		if !strings.Contains(strings.ToUpper(t.Filename), searchPattern) {
			continue
		}

		quality := detectQuality(t.Filename)
		sizeBytes, _ := strconv.ParseUint(t.SizeBytes, 10, 64)

		results = append(results, TorrentResult{
			Title:     t.Title,
			Hash:      t.Hash,
			MagnetURL: t.MagnetURL,
			Quality:   quality,
			Type:      "hdtv",
			Seeds:     uint(t.Seeds),
			Peers:     uint(t.Peers),
			Size:      formatSize(sizeBytes),
			SizeBytes: sizeBytes,
			Source:    "EZTV",
		})
	}

	return results, nil
}

func (p *EZTVProvider) SearchByIMDB(imdbID string, season, episode int) ([]TorrentResult, error) {
	// Remove 'tt' prefix if present
	imdbID = strings.TrimPrefix(imdbID, "tt")

	params := url.Values{}
	params.Set("imdb_id", imdbID)
	params.Set("limit", "100")

	resp, err := http.Get(p.baseURL + "/get-torrents?" + params.Encode())
	if err != nil {
		return nil, fmt.Errorf("EZTV request failed: %w", err)
	}
	defer resp.Body.Close()

	var eztvResp eztvResponse
	if err := json.NewDecoder(resp.Body).Decode(&eztvResp); err != nil {
		return nil, fmt.Errorf("EZTV decode failed: %w", err)
	}

	var results []TorrentResult
	searchPattern := fmt.Sprintf("S%02dE%02d", season, episode)

	for _, t := range eztvResp.Torrents {
		// Filter by episode
		if season > 0 && episode > 0 {
			if !strings.Contains(strings.ToUpper(t.Filename), searchPattern) {
				continue
			}
		}

		quality := detectQuality(t.Filename)
		sizeBytes, _ := strconv.ParseUint(t.SizeBytes, 10, 64)

		results = append(results, TorrentResult{
			Title:     t.Title,
			Hash:      t.Hash,
			MagnetURL: t.MagnetURL,
			Quality:   quality,
			Type:      "hdtv",
			Seeds:     uint(t.Seeds),
			Peers:     uint(t.Peers),
			Size:      formatSize(sizeBytes),
			SizeBytes: sizeBytes,
			Source:    "EZTV",
		})
	}

	return results, nil
}

func detectQuality(filename string) string {
	filename = strings.ToLower(filename)

	if strings.Contains(filename, "2160p") || strings.Contains(filename, "4k") {
		return "2160p"
	}
	if strings.Contains(filename, "1080p") {
		return "1080p"
	}
	if strings.Contains(filename, "720p") {
		return "720p"
	}
	if strings.Contains(filename, "480p") {
		return "480p"
	}

	return "720p" // Default
}

func formatSize(bytes uint64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := uint64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// ParseSeasonEpisode extracts season and episode numbers from a filename
func ParseSeasonEpisode(filename string) (season, episode int) {
	re := regexp.MustCompile(`[Ss](\d{1,2})[Ee](\d{1,2})`)
	matches := re.FindStringSubmatch(filename)
	if len(matches) >= 3 {
		season, _ = strconv.Atoi(matches[1])
		episode, _ = strconv.Atoi(matches[2])
	}
	return
}

// EZTVSeriesResult contains torrent info with season/episode
type EZTVSeriesResult struct {
	Title     string
	Hash      string
	MagnetURL string
	Quality   string
	Season    int
	Episode   int
	Seeds     int
	Peers     int
	Size      string
	SizeBytes uint64
}

// FetchEZTVTorrents fetches all torrents for a series by IMDB ID
func FetchEZTVTorrents(imdbID string) ([]EZTVSeriesResult, error) {
	// Remove 'tt' prefix if present
	imdbID = strings.TrimPrefix(imdbID, "tt")

	params := url.Values{}
	params.Set("imdb_id", imdbID)
	params.Set("limit", "100")
	params.Set("page", "1")

	var allResults []EZTVSeriesResult

	// Fetch multiple pages
	for page := 1; page <= 5; page++ {
		params.Set("page", strconv.Itoa(page))

		resp, err := http.Get("https://eztvx.to/api/get-torrents?" + params.Encode())
		if err != nil {
			return nil, fmt.Errorf("EZTV request failed: %w", err)
		}
		defer resp.Body.Close()

		var eztvResp eztvResponse
		if err := json.NewDecoder(resp.Body).Decode(&eztvResp); err != nil {
			return nil, fmt.Errorf("EZTV decode failed: %w", err)
		}

		if len(eztvResp.Torrents) == 0 {
			break
		}

		for _, t := range eztvResp.Torrents {
			season, episode := ParseSeasonEpisode(t.Filename)
			if season == 0 && t.Season != "" {
				season, _ = strconv.Atoi(t.Season)
			}
			if episode == 0 && t.Episode != "" {
				episode, _ = strconv.Atoi(t.Episode)
			}

			quality := detectQuality(t.Filename)
			sizeBytes, _ := strconv.ParseUint(t.SizeBytes, 10, 64)

			allResults = append(allResults, EZTVSeriesResult{
				Title:     t.Title,
				Hash:      t.Hash,
				MagnetURL: t.MagnetURL,
				Quality:   quality,
				Season:    season,
				Episode:   episode,
				Seeds:     t.Seeds,
				Peers:     t.Peers,
				Size:      formatSize(sizeBytes),
				SizeBytes: sizeBytes,
			})
		}

		// If we got less than limit, no more pages
		if len(eztvResp.Torrents) < 100 {
			break
		}
	}

	return allResults, nil
}
