package providers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

var ytsTrackers = []string{
	"udp://tracker.opentrackr.org:1337/announce",
	"udp://tracker.torrent.eu.org:451/announce",
	"udp://tracker.dler.org:6969/announce",
	"udp://open.stealth.si:80/announce",
	"udp://open.demonii.com:1337/announce",
	"https://tracker.moeblog.cn:443/announce",
	"udp://open.dstud.io:6969/announce",
	"udp://tracker.srv00.com:6969/announce",
	"https://tracker.zhuqiy.com:443/announce",
	"https://tracker.pmman.tech:443/announce",
}

type YTSProvider struct {
	baseURL string
}

type ytsResponse struct {
	Status        string `json:"status"`
	StatusMessage string `json:"status_message"`
	Data          struct {
		MovieCount int        `json:"movie_count"`
		Movies     []ytsMovie `json:"movies"`
	} `json:"data"`
}

type ytsMovie struct {
	ID       int    `json:"id"`
	Title    string `json:"title"`
	Year     int    `json:"year"`
	Torrents []struct {
		URL       string `json:"url"`
		Hash      string `json:"hash"`
		Quality   string `json:"quality"`
		Type      string `json:"type"`
		Seeds     int    `json:"seeds"`
		Peers     int    `json:"peers"`
		Size      string `json:"size"`
		SizeBytes int64  `json:"size_bytes"`
	} `json:"torrents"`
}

func NewYTSProvider() *YTSProvider {
	return &YTSProvider{
		baseURL: "https://yts.mx/api/v2",
	}
}

func (p *YTSProvider) Name() string {
	return "YTS"
}

func (p *YTSProvider) SearchMovie(title string, year int) ([]TorrentResult, error) {
	params := url.Values{}
	params.Set("query_term", title)
	if year > 0 {
		params.Set("year", strconv.Itoa(year))
	}
	params.Set("limit", "10")

	resp, err := http.Get(p.baseURL + "/list_movies.json?" + params.Encode())
	if err != nil {
		return nil, fmt.Errorf("YTS request failed: %w", err)
	}
	defer resp.Body.Close()

	var ytsResp ytsResponse
	if err := json.NewDecoder(resp.Body).Decode(&ytsResp); err != nil {
		return nil, fmt.Errorf("YTS decode failed: %w", err)
	}

	if ytsResp.Status != "ok" {
		return nil, fmt.Errorf("YTS error: %s", ytsResp.StatusMessage)
	}

	var results []TorrentResult
	for _, movie := range ytsResp.Data.Movies {
		for _, t := range movie.Torrents {
			results = append(results, TorrentResult{
				Title:     fmt.Sprintf("%s (%d) - %s", movie.Title, movie.Year, t.Quality),
				Hash:      t.Hash,
				MagnetURL: buildMagnetURL(t.Hash, fmt.Sprintf("%s (%d) [%s] [YTS.MX]", movie.Title, movie.Year, t.Quality)),
				Quality:   t.Quality,
				Type:      t.Type,
				Seeds:     uint(t.Seeds),
				Peers:     uint(t.Peers),
				Size:      t.Size,
				SizeBytes: uint64(t.SizeBytes),
				Source:    "YTS",
			})
		}
	}

	return results, nil
}

func (p *YTSProvider) SearchSeries(title string, season, episode int) ([]TorrentResult, error) {
	// YTS doesn't support series
	return nil, fmt.Errorf("YTS does not support series")
}

func buildMagnetURL(hash, name string) string {
	var b strings.Builder
	b.WriteString("magnet:?xt=urn:btih:")
	b.WriteString(hash)
	b.WriteString("&dn=")
	b.WriteString(url.QueryEscape(name))
	for _, tracker := range ytsTrackers {
		b.WriteString("&tr=")
		b.WriteString(url.QueryEscape(tracker))
	}
	return b.String()
}
