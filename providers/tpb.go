package providers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// TPB categories
const (
	TPBCatMovies   = "201"
	TPBCatTV       = "205"
	TPBCatHDMovies = "207"
	TPBCatHDTV     = "208"
	TPBCat4KMovies = "211"
	TPBCat4KTV     = "212"
)

type TPBProvider struct {
	baseURL string
	client  *http.Client
}

type tpbResult struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	InfoHash string `json:"info_hash"`
	Leechers string `json:"leechers"`
	Seeders  string `json:"seeders"`
	Size     string `json:"size"`
	Added    string `json:"added"`
	Category string `json:"category"`
	IMDB     string `json:"imdb"`
}

func NewTPBProvider() *TPBProvider {
	return &TPBProvider{
		baseURL: "https://apibay.org",
		client:  &http.Client{Timeout: 10 * time.Second},
	}
}

func (p *TPBProvider) Name() string {
	return "TPB"
}

func (p *TPBProvider) SearchMovie(title string, year int) ([]TorrentResult, error) {
	query := title
	if year > 0 {
		query = fmt.Sprintf("%s %d", title, year)
	}

	// Search HD movies + regular movies
	results, err := p.search(query, TPBCatHDMovies)
	if err != nil {
		// Fallback to all movies
		results, err = p.search(query, TPBCatMovies)
		if err != nil {
			return nil, err
		}
	}

	// Also search 4K
	results4k, _ := p.search(query, TPBCat4KMovies)
	results = append(results, results4k...)

	return dedupeResults(results), nil
}

func (p *TPBProvider) SearchSeries(title string, season, episode int) ([]TorrentResult, error) {
	query := title
	if season > 0 && episode > 0 {
		query = fmt.Sprintf("%s S%02dE%02d", title, season, episode)
	} else if season > 0 {
		query = fmt.Sprintf("%s S%02d", title, season)
	}

	results, err := p.search(query, TPBCatHDTV)
	if err != nil {
		results, err = p.search(query, TPBCatTV)
		if err != nil {
			return nil, err
		}
	}

	// Also search 4K TV
	results4k, _ := p.search(query, TPBCat4KTV)
	results = append(results, results4k...)

	return dedupeResults(results), nil
}

// SearchAll searches without category filter for the aggregator
func (p *TPBProvider) SearchAll(query string) ([]TorrentResult, error) {
	return p.search(query, "0")
}

func (p *TPBProvider) search(query, category string) ([]TorrentResult, error) {
	apiURL := fmt.Sprintf("%s/q.php?q=%s&cat=%s", p.baseURL, url.QueryEscape(query), category)

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("TPB request failed: %w", err)
	}
	defer resp.Body.Close()

	var tpbResults []tpbResult
	if err := json.NewDecoder(resp.Body).Decode(&tpbResults); err != nil {
		return nil, fmt.Errorf("TPB decode failed: %w", err)
	}

	var results []TorrentResult
	for _, t := range tpbResults {
		// Skip the "no results" placeholder
		if t.ID == "0" || t.InfoHash == "0000000000000000000000000000000000000000" {
			continue
		}

		seeds, _ := strconv.ParseUint(t.Seeders, 10, 32)
		peers, _ := strconv.ParseUint(t.Leechers, 10, 32)
		sizeBytes, _ := strconv.ParseUint(t.Size, 10, 64)

		hash := strings.ToUpper(t.InfoHash)
		magnet := fmt.Sprintf("magnet:?xt=urn:btih:%s&dn=%s", hash, url.QueryEscape(t.Name))

		results = append(results, TorrentResult{
			Title:     t.Name,
			Hash:      hash,
			MagnetURL: magnet,
			Quality:   detectQuality(t.Name),
			Type:      detectType(t.Name),
			Seeds:     uint(seeds),
			Peers:     uint(peers),
			Size:      formatSize(sizeBytes),
			SizeBytes: sizeBytes,
			Source:    "TPB",
		})
	}

	return results, nil
}

func dedupeResults(results []TorrentResult) []TorrentResult {
	seen := make(map[string]bool)
	var deduped []TorrentResult
	for _, r := range results {
		if !seen[r.Hash] {
			seen[r.Hash] = true
			deduped = append(deduped, r)
		}
	}
	return deduped
}
