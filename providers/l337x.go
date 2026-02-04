package providers

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

type L337xProvider struct {
	baseURL string
}

func NewL337xProvider() *L337xProvider {
	return &L337xProvider{
		baseURL: "https://1337x.to",
	}
}

func (p *L337xProvider) Name() string {
	return "1337x"
}

func (p *L337xProvider) SearchMovie(title string, year int) ([]TorrentResult, error) {
	query := title
	if year > 0 {
		query = fmt.Sprintf("%s %d", title, year)
	}

	return p.search(query, "Movies")
}

func (p *L337xProvider) SearchSeries(title string, season, episode int) ([]TorrentResult, error) {
	query := fmt.Sprintf("%s S%02dE%02d", title, season, episode)
	return p.search(query, "TV")
}

func (p *L337xProvider) search(query, category string) ([]TorrentResult, error) {
	searchURL := fmt.Sprintf("%s/category-search/%s/%s/1/", p.baseURL, url.PathEscape(query), category)

	req, err := http.NewRequest("GET", searchURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("1337x request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return p.parseSearchResults(string(body))
}

func (p *L337xProvider) parseSearchResults(html string) ([]TorrentResult, error) {
	var results []TorrentResult

	// Parse torrent links from HTML
	linkRe := regexp.MustCompile(`<a href="(/torrent/[^"]+)">([^<]+)</a>`)
	seedRe := regexp.MustCompile(`<td class="coll-2 seeds">(\d+)</td>`)
	leechRe := regexp.MustCompile(`<td class="coll-3 leeches">(\d+)</td>`)
	sizeRe := regexp.MustCompile(`<td class="coll-4 size[^"]*">([^<]+)<`)

	linkMatches := linkRe.FindAllStringSubmatch(html, -1)
	seedMatches := seedRe.FindAllStringSubmatch(html, -1)
	leechMatches := leechRe.FindAllStringSubmatch(html, -1)
	sizeMatches := sizeRe.FindAllStringSubmatch(html, -1)

	for i, match := range linkMatches {
		if len(match) < 3 {
			continue
		}

		torrentURL := p.baseURL + match[1]
		title := match[2]

		var seeds, peers uint
		if i < len(seedMatches) && len(seedMatches[i]) > 1 {
			s, _ := strconv.Atoi(seedMatches[i][1])
			seeds = uint(s)
		}
		if i < len(leechMatches) && len(leechMatches[i]) > 1 {
			pe, _ := strconv.Atoi(leechMatches[i][1])
			peers = uint(pe)
		}

		var size string
		if i < len(sizeMatches) && len(sizeMatches[i]) > 1 {
			size = strings.TrimSpace(sizeMatches[i][1])
		}

		// Get magnet link from torrent page
		hash, magnetURL := p.getMagnetLink(torrentURL)
		if hash == "" {
			continue
		}

		results = append(results, TorrentResult{
			Title:     title,
			Hash:      hash,
			MagnetURL: magnetURL,
			Quality:   detectQuality(title),
			Type:      detectType(title),
			Seeds:     seeds,
			Peers:     peers,
			Size:      size,
			SizeBytes: parseSize(size),
			Source:    "1337x",
		})

		// Limit results
		if len(results) >= 20 {
			break
		}
	}

	return results, nil
}

func (p *L337xProvider) getMagnetLink(torrentURL string) (string, string) {
	req, err := http.NewRequest("GET", torrentURL, nil)
	if err != nil {
		return "", ""
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", ""
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", ""
	}

	// Extract magnet link
	magnetRe := regexp.MustCompile(`href="(magnet:\?xt=urn:btih:([a-fA-F0-9]+)[^"]*)"`)
	match := magnetRe.FindStringSubmatch(string(body))
	if len(match) >= 3 {
		return strings.ToUpper(match[2]), match[1]
	}

	return "", ""
}

func detectType(title string) string {
	title = strings.ToLower(title)

	if strings.Contains(title, "bluray") || strings.Contains(title, "blu-ray") {
		return "bluray"
	}
	if strings.Contains(title, "webrip") || strings.Contains(title, "web-rip") {
		return "webrip"
	}
	if strings.Contains(title, "webdl") || strings.Contains(title, "web-dl") {
		return "web"
	}
	if strings.Contains(title, "hdtv") {
		return "hdtv"
	}
	if strings.Contains(title, "dvdrip") {
		return "dvdrip"
	}

	return "web"
}

func parseSize(size string) uint64 {
	size = strings.ToUpper(strings.TrimSpace(size))
	size = strings.ReplaceAll(size, ",", ".")

	re := regexp.MustCompile(`([\d.]+)\s*(GB|MB|KB|TB)`)
	match := re.FindStringSubmatch(size)
	if len(match) < 3 {
		return 0
	}

	value, _ := strconv.ParseFloat(match[1], 64)
	unit := match[2]

	switch unit {
	case "TB":
		return uint64(value * 1024 * 1024 * 1024 * 1024)
	case "GB":
		return uint64(value * 1024 * 1024 * 1024)
	case "MB":
		return uint64(value * 1024 * 1024)
	case "KB":
		return uint64(value * 1024)
	}

	return 0
}
