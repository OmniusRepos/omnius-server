package services

import (
	"archive/zip"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/anacrolix/torrent"

	"torrent-server/database"
	"torrent-server/models"
)

const (
	subdlAPIURL          = "https://api.subdl.com/api/v1/subtitles"
	opensubtitlesRestURL = "https://rest.opensubtitles.org/search"
)

type SubtitleService struct {
	client       *http.Client
	subdlKey     string
	db           *database.DB
	subtitlesDir string
}

func NewSubtitleService() *SubtitleService {
	apiKey := os.Getenv("SUBDL_API_KEY")
	if apiKey == "" {
		apiKey = "4bHkwDgMS95KS34bCXOo7y1LgkAKkK6P"
	}
	return &SubtitleService{
		client:   &http.Client{Timeout: 15 * time.Second},
		subdlKey: apiKey,
	}
}

func NewSubtitleServiceWithDB(db *database.DB, subtitlesDir string) *SubtitleService {
	s := NewSubtitleService()
	s.db = db
	s.subtitlesDir = subtitlesDir
	return s
}

// SyncEpisodeSubtitles downloads and stores subtitles for a specific episode.
func (s *SubtitleService) SyncEpisodeSubtitles(imdbCode string, languages string, season, episode int) (int, error) {
	if s.db == nil {
		return 0, fmt.Errorf("no database configured")
	}

	log.Printf("[SubtitleSync] Syncing subtitles for %s S%02dE%02d (languages: %s)", imdbCode, season, episode, languages)
	imdb := strings.TrimPrefix(imdbCode, "tt")

	stored := 0
	storedLangs := make(map[string]int)

	for _, lang := range strings.Split(languages, ",") {
		lang = strings.TrimSpace(lang)
		if lang == "" || storedLangs[lang] >= 2 {
			continue
		}
		subdlResult, err := s.searchSubDLEpisode(imdb, lang, season, episode)
		if err != nil {
			log.Printf("[SubtitleSync] SubDL error for %s S%02dE%02d %s: %v", imdbCode, season, episode, lang, err)
			continue
		}
		if len(subdlResult.Subtitles) == 0 {
			continue
		}
		count := s.syncDownloadEpisodeSubtitles(imdbCode, lang, subdlResult.Subtitles, 2-storedLangs[lang], season, episode)
		storedLangs[lang] += count
		stored += count
		time.Sleep(300 * time.Millisecond)
	}

	log.Printf("[SubtitleSync] Stored %d subtitles for %s S%02dE%02d", stored, imdbCode, season, episode)
	return stored, nil
}

// syncDownloadEpisodeSubtitles downloads and stores episode subtitles.
func (s *SubtitleService) syncDownloadEpisodeSubtitles(imdbCode, lang string, subs []Subtitle, limit, season, episode int) int {
	if limit <= 0 {
		return 0
	}
	stored := 0
	for i, sub := range subs {
		if i >= limit {
			break
		}
		vtt, err := s.DownloadSubtitle(sub.DownloadURL)
		if err != nil {
			log.Printf("[SubtitleSync] Failed to download %s subtitle for %s S%02dE%02d: %v", lang, imdbCode, season, episode, err)
			continue
		}

		source := "subdl"
		if strings.Contains(sub.DownloadURL, "opensubtitles.org") {
			source = "opensubtitles"
		}

		storedSub := &models.StoredSubtitle{
			ImdbCode:        imdbCode,
			Language:        sub.Language,
			LanguageName:    sub.LanguageName,
			ReleaseName:     sub.ReleaseName,
			HearingImpaired: sub.HearingImpaired,
			Source:          source,
			SeasonNumber:    season,
			EpisodeNumber:   episode,
		}
		if err := s.db.CreateSubtitle(storedSub); err != nil {
			continue
		}
		if storedSub.ID == 0 {
			continue
		}

		if vttPath, err := s.writeSubtitleFile(imdbCode, storedSub.ID, vtt); err != nil {
			log.Printf("[SubtitleSync] Failed to write VTT file: %v", err)
		} else {
			s.db.UpdateSubtitlePath(storedSub.ID, vttPath)
		}
		stored++
		time.Sleep(500 * time.Millisecond)
	}
	return stored
}

// SearchByIMDBEpisode searches subtitles for a specific episode.
func (s *SubtitleService) SearchByIMDBEpisode(imdbID string, languages string, season, episode int) (*SubtitleSearchResult, error) {
	imdb := strings.TrimPrefix(imdbID, "tt")
	log.Printf("[SubtitleService] Searching subtitles for IMDB: %s S%02dE%02d, languages: %s", imdb, season, episode, languages)

	var allSubtitles []Subtitle

	// OpenSubtitles with season/episode
	if languages != "" {
		for _, lang := range strings.Split(languages, ",") {
			lang = strings.TrimSpace(lang)
			osLang := iso2ToOSLang(lang)
			if osLang == "" {
				continue
			}
			osResult, err := s.searchOpenSubtitlesRESTEpisode(imdb, osLang, season, episode)
			if err == nil {
				allSubtitles = append(allSubtitles, osResult.Subtitles...)
			}
			time.Sleep(200 * time.Millisecond)
		}
	}

	// SubDL with season/episode
	subdlResult, err := s.searchSubDLEpisode(imdb, languages, season, episode)
	if err == nil {
		allSubtitles = append(allSubtitles, subdlResult.Subtitles...)
	}

	if len(allSubtitles) == 0 {
		return &SubtitleSearchResult{Subtitles: []Subtitle{}, TotalCount: 0}, nil
	}
	return &SubtitleSearchResult{Subtitles: allSubtitles, TotalCount: len(allSubtitles)}, nil
}

// SyncSubtitles downloads and stores subtitles for a given IMDB code.
// Downloads immediately after each search to avoid URL token expiration.
func (s *SubtitleService) SyncSubtitles(imdbCode string, languages string) (int, error) {
	if s.db == nil {
		return 0, fmt.Errorf("no database configured")
	}

	log.Printf("[SubtitleSync] Syncing subtitles for %s (languages: %s)", imdbCode, languages)
	imdb := strings.TrimPrefix(imdbCode, "tt")

	stored := 0
	storedLangs := make(map[string]int)

	// SubDL for sync downloads (OpenSubtitles blocks server IPs for downloads)
	// OpenSubtitles still used for live search — app downloads directly via Tauri
	for _, lang := range strings.Split(languages, ",") {
		lang = strings.TrimSpace(lang)
		if lang == "" || storedLangs[lang] >= 3 {
			continue
		}
		subdlResult, err := s.searchSubDL(imdb, lang)
		if err != nil {
			log.Printf("[SubtitleSync] SubDL error for %s: %v", lang, err)
			continue
		}
		if len(subdlResult.Subtitles) == 0 {
			log.Printf("[SubtitleSync] No SubDL results for %s", lang)
			continue
		}
		count := s.syncDownloadSubtitles(imdbCode, lang, subdlResult.Subtitles, 3-storedLangs[lang])
		storedLangs[lang] += count
		stored += count
		time.Sleep(300 * time.Millisecond)
	}

	log.Printf("[SubtitleSync] Stored %d subtitles for %s", stored, imdbCode)
	return stored, nil
}

// syncDownloadSubtitles downloads and stores up to `limit` subtitles from the list.
func (s *SubtitleService) syncDownloadSubtitles(imdbCode, lang string, subs []Subtitle, limit int) int {
	if limit <= 0 {
		return 0
	}
	stored := 0
	for i, sub := range subs {
		if i >= limit {
			break
		}
		vtt, err := s.DownloadSubtitle(sub.DownloadURL)
		if err != nil {
			log.Printf("[SubtitleSync] Failed to download %s subtitle for %s: %v", lang, imdbCode, err)
			continue
		}

		source := "subdl"
		if strings.Contains(sub.DownloadURL, "opensubtitles.org") {
			source = "opensubtitles"
		}

		storedSub := &models.StoredSubtitle{
			ImdbCode:        imdbCode,
			Language:        sub.Language,
			LanguageName:    sub.LanguageName,
			ReleaseName:     sub.ReleaseName,
			HearingImpaired: sub.HearingImpaired,
			Source:          source,
		}
		if err := s.db.CreateSubtitle(storedSub); err != nil {
			log.Printf("[SubtitleSync] Failed to store subtitle: %v", err)
			continue
		}
		if storedSub.ID == 0 {
			log.Printf("[SubtitleSync] Skipped duplicate: %s %s", lang, sub.ReleaseName)
			continue
		}

		if vttPath, err := s.writeSubtitleFile(imdbCode, storedSub.ID, vtt); err != nil {
			log.Printf("[SubtitleSync] Failed to write VTT file: %v", err)
		} else {
			s.db.UpdateSubtitlePath(storedSub.ID, vttPath)
		}
		stored++
		time.Sleep(500 * time.Millisecond)
	}
	return stored
}

// ExtractSubtitlesFromTorrent reads subtitle files (.srt, .ass, etc.) from a loaded torrent
// and stores them in the DB. Returns the number of subtitles extracted.
func (s *SubtitleService) ExtractSubtitlesFromTorrent(t *torrent.Torrent, ts *TorrentService, imdbCode string) int {
	if s.db == nil || t == nil || imdbCode == "" {
		return 0
	}

	subtitleExts := map[string]bool{
		".srt": true, ".vtt": true, ".ass": true, ".ssa": true, ".sub": true,
	}

	files := t.Files()
	extracted := 0

	for i, f := range files {
		ext := strings.ToLower(filepath.Ext(f.Path()))
		if !subtitleExts[ext] {
			continue
		}

		name := filepath.Base(f.Path())
		log.Printf("[SubtitleExtract] Found subtitle in torrent: %s (%d bytes)", name, f.Length())

		// Skip very large files (>5MB) - not a real subtitle
		if f.Length() > 5*1024*1024 {
			continue
		}

		reader, _, err := ts.GetFileReader(t, i)
		if err != nil {
			log.Printf("[SubtitleExtract] Failed to get reader for %s: %v", name, err)
			continue
		}

		data, err := io.ReadAll(reader)
		if err != nil {
			log.Printf("[SubtitleExtract] Failed to read %s: %v", name, err)
			continue
		}

		// Detect language from filename (e.g., "Movie.eng.srt", "Movie.English.srt")
		lang, langName := detectLanguageFromFilename(name)

		vtt := convertToVTT(toUTF8(string(data)))

		storedSub := &models.StoredSubtitle{
			ImdbCode:     imdbCode,
			Language:     lang,
			LanguageName: langName,
			ReleaseName:  name,
			Source:       "torrent",
		}
		if err := s.db.CreateSubtitle(storedSub); err != nil {
			log.Printf("[SubtitleExtract] Failed to store %s: %v", name, err)
			continue
		}

		// Write VTT to disk and update path
		if vttPath, err := s.writeSubtitleFile(imdbCode, storedSub.ID, vtt); err != nil {
			log.Printf("[SubtitleExtract] Failed to write VTT file: %v", err)
		} else {
			s.db.UpdateSubtitlePath(storedSub.ID, vttPath)
		}
		extracted++
		log.Printf("[SubtitleExtract] Stored subtitle: %s (lang=%s)", name, lang)
	}

	return extracted
}

// writeSubtitleFile writes VTT content to data/subtitles/{imdbCode}/{id}.vtt
func (s *SubtitleService) writeSubtitleFile(imdbCode string, id uint, vtt string) (string, error) {
	if s.subtitlesDir == "" {
		return "", fmt.Errorf("subtitles directory not configured")
	}
	dir := filepath.Join(s.subtitlesDir, imdbCode)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("failed to create subtitle dir: %w", err)
	}
	vttPath := filepath.Join(dir, fmt.Sprintf("%d.vtt", id))
	if err := os.WriteFile(vttPath, []byte(vtt), 0644); err != nil {
		return "", fmt.Errorf("failed to write VTT file: %w", err)
	}
	return vttPath, nil
}

// detectLanguageFromFilename tries to detect subtitle language from the filename.
func detectLanguageFromFilename(filename string) (string, string) {
	name := strings.ToLower(filename)
	// Remove extension
	name = strings.TrimSuffix(name, filepath.Ext(name))
	parts := strings.Split(name, ".")

	langMap := map[string][2]string{
		"eng": {"en", "English"}, "english": {"en", "English"},
		"spa": {"es", "Spanish"}, "spanish": {"es", "Spanish"},
		"fre": {"fr", "French"}, "french": {"fr", "French"},
		"ger": {"de", "German"}, "german": {"de", "German"},
		"ita": {"it", "Italian"}, "italian": {"it", "Italian"},
		"por": {"pt", "Portuguese"}, "portuguese": {"pt", "Portuguese"},
		"rus": {"ru", "Russian"}, "russian": {"ru", "Russian"},
		"chi": {"zh", "Chinese"}, "chinese": {"zh", "Chinese"},
		"jpn": {"ja", "Japanese"}, "japanese": {"ja", "Japanese"},
		"kor": {"ko", "Korean"}, "korean": {"ko", "Korean"},
		"ara": {"ar", "Arabic"}, "arabic": {"ar", "Arabic"},
		"dut": {"nl", "Dutch"}, "dutch": {"nl", "Dutch"},
		"pol": {"pl", "Polish"}, "polish": {"pl", "Polish"},
		"tur": {"tr", "Turkish"}, "turkish": {"tr", "Turkish"},
		"swe": {"sv", "Swedish"}, "swedish": {"sv", "Swedish"},
		"nor": {"no", "Norwegian"}, "norwegian": {"no", "Norwegian"},
		"dan": {"da", "Danish"}, "danish": {"da", "Danish"},
		"fin": {"fi", "Finnish"}, "finnish": {"fi", "Finnish"},
		"alb": {"sq", "Albanian"}, "albanian": {"sq", "Albanian"},
		"sqi": {"sq", "Albanian"},
		"en":  {"en", "English"}, "es": {"es", "Spanish"},
		"fr":  {"fr", "French"}, "de": {"de", "German"},
		"it":  {"it", "Italian"}, "pt": {"pt", "Portuguese"},
		"ru":  {"ru", "Russian"}, "sq": {"sq", "Albanian"},
	}

	// Check parts from the end (language tag is usually last before extension)
	for i := len(parts) - 1; i >= 0; i-- {
		if match, ok := langMap[parts[i]]; ok {
			return match[0], match[1]
		}
	}

	// Default to English (most common in torrents)
	return "en", "English"
}

// Subtitle represents a single subtitle result
type Subtitle struct {
	ID               string  `json:"id"`
	Language         string  `json:"language"`
	LanguageName     string  `json:"language_name"`
	DownloadURL      string  `json:"download_url"`
	ReleaseName      string  `json:"release_name,omitempty"`
	Uploader         string  `json:"uploader,omitempty"`
	DownloadCount    int64   `json:"download_count"`
	HearingImpaired  bool    `json:"hearing_impaired"`
	FPS              float64 `json:"fps,omitempty"`
}

// SubtitleSearchResult wraps subtitle search results
type SubtitleSearchResult struct {
	Subtitles  []Subtitle `json:"subtitles"`
	TotalCount int        `json:"total_count"`
}

// SearchByIMDB searches subtitles by IMDB ID, queries OpenSubtitles first, SubDL as fallback
func (s *SubtitleService) SearchByIMDB(imdbID string, languages string) (*SubtitleSearchResult, error) {
	imdb := strings.TrimPrefix(imdbID, "tt")
	log.Printf("[SubtitleService] Searching subtitles for IMDB: %s, languages: %s", imdb, languages)

	var allSubtitles []Subtitle

	// Query OpenSubtitles first (better quality matches)
	if languages != "" {
		for _, lang := range strings.Split(languages, ",") {
			lang = strings.TrimSpace(lang)
			osLang := iso2ToOSLang(lang)
			if osLang == "" {
				continue
			}
			osResult, err := s.searchOpenSubtitlesREST(imdb, osLang)
			if err != nil {
				log.Printf("[SubtitleService] OpenSubtitles error for %s: %v", lang, err)
				continue
			}
			allSubtitles = append(allSubtitles, osResult.Subtitles...)
			if len(osResult.Subtitles) > 0 {
				log.Printf("[SubtitleService] OpenSubtitles found %d %s subtitles", len(osResult.Subtitles), lang)
			}
			time.Sleep(200 * time.Millisecond) // rate limit
		}
	} else {
		osResult, err := s.searchOpenSubtitlesREST(imdb, "")
		if err != nil {
			log.Printf("[SubtitleService] OpenSubtitles error: %v", err)
		} else {
			allSubtitles = append(allSubtitles, osResult.Subtitles...)
			log.Printf("[SubtitleService] OpenSubtitles found %d subtitles", len(osResult.Subtitles))
		}
	}

	// Query SubDL as fallback (fills gaps for languages OpenSubtitles missed)
	subdlResult, err := s.searchSubDL(imdb, languages)
	if err != nil {
		log.Printf("[SubtitleService] SubDL error: %v", err)
	} else {
		allSubtitles = append(allSubtitles, subdlResult.Subtitles...)
		log.Printf("[SubtitleService] SubDL found %d subtitles", len(subdlResult.Subtitles))
	}

	if len(allSubtitles) == 0 {
		return &SubtitleSearchResult{Subtitles: []Subtitle{}, TotalCount: 0}, nil
	}

	return &SubtitleSearchResult{Subtitles: allSubtitles, TotalCount: len(allSubtitles)}, nil
}

// SearchByFilename searches subtitles by release filename
func (s *SubtitleService) SearchByFilename(filename string, languages string) (*SubtitleSearchResult, error) {
	log.Printf("[SubtitleService] Searching subtitles by filename: %s", filename)

	encodedFilename := url.QueryEscape(filename)
	apiURL := fmt.Sprintf("%s?api_key=%s&file_name=%s", subdlAPIURL, s.subdlKey, encodedFilename)
	if languages != "" {
		apiURL += "&languages=" + url.QueryEscape(strings.ToUpper(languages))
	}

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "OmniusServer v1.0")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch subtitles by filename: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("SubDL API error: %d", resp.StatusCode)
	}

	return s.parseSubDLResponse(resp.Body)
}

// DownloadSubtitle downloads a subtitle file, decompresses if needed, converts to VTT
func (s *SubtitleService) DownloadSubtitle(downloadURL string) (string, error) {
	log.Printf("[SubtitleService] Downloading subtitle from: %s", downloadURL)

	data, err := s.downloadWithHTTP(downloadURL)
	if err != nil {
		return "", err
	}

	log.Printf("[SubtitleService] Downloaded %d bytes", len(data))

	// Detect format and decompress
	var content string
	if len(data) >= 2 && data[0] == 0x1f && data[1] == 0x8b {
		// Gzip
		content, err = decompressGzip(data)
		if err != nil {
			return "", err
		}
	} else if len(data) >= 2 && data[0] == 0x50 && data[1] == 0x4B {
		// ZIP
		content, err = extractSubtitleFromZip(data)
		if err != nil {
			return "", err
		}
	} else {
		content = string(data)
	}

	// Ensure UTF-8 encoding
	content = toUTF8(content)

	// Convert to VTT
	vtt := convertToVTT(content)
	log.Printf("[SubtitleService] Converted to VTT (%d chars)", len(vtt))
	return vtt, nil
}

// downloadWithHTTP downloads a file using Go's HTTP client
func (s *SubtitleService) downloadWithHTTP(downloadURL string) ([]byte, error) {
	req, err := http.NewRequest("GET", downloadURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "OmniusServer v1.0")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to download subtitle: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to download subtitle: HTTP %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

func (s *SubtitleService) searchSubDLEpisode(imdbID string, languages string, season, episode int) (*SubtitleSearchResult, error) {
	apiURL := fmt.Sprintf("%s?api_key=%s&imdb_id=tt%s&season_number=%d&episode_number=%d", subdlAPIURL, s.subdlKey, imdbID, season, episode)
	if languages != "" {
		apiURL += "&languages=" + url.QueryEscape(strings.ToUpper(languages))
	}

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "OmniusServer v1.0")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch subtitles: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("SubDL API error: %d", resp.StatusCode)
	}

	return s.parseSubDLResponse(resp.Body)
}

func (s *SubtitleService) searchSubDL(imdbID string, languages string) (*SubtitleSearchResult, error) {
	apiURL := fmt.Sprintf("%s?api_key=%s&imdb_id=tt%s", subdlAPIURL, s.subdlKey, imdbID)
	if languages != "" {
		apiURL += "&languages=" + url.QueryEscape(strings.ToUpper(languages))
	}

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "OmniusServer v1.0")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch subtitles: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("SubDL API error: %d", resp.StatusCode)
	}

	return s.parseSubDLResponse(resp.Body)
}

func (s *SubtitleService) parseSubDLResponse(body io.Reader) (*SubtitleSearchResult, error) {
	var data subDLResponse
	if err := json.NewDecoder(body).Decode(&data); err != nil {
		return nil, fmt.Errorf("failed to parse SubDL response: %w", err)
	}

	if !data.Status {
		return &SubtitleSearchResult{Subtitles: []Subtitle{}, TotalCount: 0}, nil
	}

	subtitles := make([]Subtitle, 0, len(data.Subtitles))
	for _, sub := range data.Subtitles {
		langCode := subdlLangToISO2(strings.ToLower(sub.Language))
		langName := sub.Lang
		if langName == "" {
			langName = sub.Language
		}
		subtitles = append(subtitles, Subtitle{
			ID:              sub.URL,
			Language:        langCode,
			LanguageName:    langName,
			DownloadURL:     "https://dl.subdl.com" + sub.URL,
			ReleaseName:     sub.ReleaseName,
			Uploader:        sub.Author,
			DownloadCount:   0,
			HearingImpaired: sub.HI,
		})
	}

	log.Printf("[SubtitleService] SubDL found %d subtitles", len(subtitles))
	return &SubtitleSearchResult{Subtitles: subtitles, TotalCount: len(subtitles)}, nil
}

func (s *SubtitleService) searchOpenSubtitlesRESTEpisode(imdbID string, osLang string, season, episode int) (*SubtitleSearchResult, error) {
	apiURL := fmt.Sprintf("%s/imdbid-%s/season-%d/episode-%d", opensubtitlesRestURL, imdbID, season, episode)
	if osLang != "" {
		apiURL += "/sublanguageid-" + osLang
	}
	return s.doOpenSubtitlesSearch(apiURL)
}

func (s *SubtitleService) searchOpenSubtitlesREST(imdbID string, osLang string) (*SubtitleSearchResult, error) {
	apiURL := fmt.Sprintf("%s/imdbid-%s", opensubtitlesRestURL, imdbID)
	if osLang != "" {
		apiURL += "/sublanguageid-" + osLang
	}
	return s.doOpenSubtitlesSearch(apiURL)
}

func (s *SubtitleService) doOpenSubtitlesSearch(apiURL string) (*SubtitleSearchResult, error) {
	log.Printf("[SubtitleService] Fetching: %s", apiURL)

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "OmniusServer v1.0")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch subtitles: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("[SubtitleService] OpenSubtitles API error: %d", resp.StatusCode)
		return &SubtitleSearchResult{Subtitles: []Subtitle{}, TotalCount: 0}, nil
	}

	var data []openSubtitlesResult
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("failed to parse subtitle response: %w", err)
	}

	subtitles := make([]Subtitle, 0, len(data))
	for _, sub := range data {
		var fps float64
		fmt.Sscanf(sub.MovieFPS, "%f", &fps)
		var dlCount int64
		fmt.Sscanf(sub.SubDownloadsCnt, "%d", &dlCount)

		// Normalize 3-letter to 2-letter language code
		langCode := osLangToISO2(sub.SubLanguageID)

		subtitles = append(subtitles, Subtitle{
			ID:              sub.IDSubtitleFile,
			Language:        langCode,
			LanguageName:    sub.LanguageName,
			DownloadURL:     sub.SubDownloadLink,
			ReleaseName:     sub.MovieReleaseName,
			Uploader:        sub.UserNickName,
			DownloadCount:   dlCount,
			HearingImpaired: sub.SubHearingImpaired == "1",
			FPS:             fps,
		})
	}

	log.Printf("[SubtitleService] Found %d subtitles from OpenSubtitles", len(subtitles))
	return &SubtitleSearchResult{Subtitles: subtitles, TotalCount: len(subtitles)}, nil
}

// --- Encoding detection ---

// toUTF8 converts text to UTF-8 if it's not already valid UTF-8.
// Handles Windows-1252/Latin-1 encoded subtitles (common for European languages).
func toUTF8(s string) string {
	if utf8.ValidString(s) {
		return s
	}
	// Check for UTF-8 BOM and strip it
	if len(s) >= 3 && s[0] == 0xEF && s[1] == 0xBB && s[2] == 0xBF {
		s = s[3:]
		if utf8.ValidString(s) {
			return s
		}
	}
	// Assume Windows-1252 (superset of Latin-1, most common for European subs)
	var buf strings.Builder
	buf.Grow(len(s) * 2)
	for i := 0; i < len(s); i++ {
		b := s[i]
		if b < 0x80 {
			buf.WriteByte(b)
		} else {
			// Windows-1252 bytes 0x80-0xFF map to specific Unicode code points
			buf.WriteRune(windows1252ToRune(b))
		}
	}
	return buf.String()
}

// windows1252ToRune converts a Windows-1252 byte (0x80-0xFF) to its Unicode rune.
func windows1252ToRune(b byte) rune {
	// 0x80-0x9F range has special mappings in Windows-1252 (differs from Latin-1)
	if b >= 0xA0 {
		return rune(b) // Latin-1 supplement: direct mapping
	}
	// Windows-1252 special range 0x80-0x9F
	table := [32]rune{
		0x20AC, 0x0081, 0x201A, 0x0192, 0x201E, 0x2026, 0x2020, 0x2021, // 80-87
		0x02C6, 0x2030, 0x0160, 0x2039, 0x0152, 0x008D, 0x017D, 0x008F, // 88-8F
		0x0090, 0x2018, 0x2019, 0x201C, 0x201D, 0x2022, 0x2013, 0x2014, // 90-97
		0x02DC, 0x2122, 0x0161, 0x203A, 0x0153, 0x009D, 0x017E, 0x0178, // 98-9F
	}
	return table[b-0x80]
}

// --- Subtitle format conversion ---

func convertToVTT(content string) string {
	trimmed := strings.TrimSpace(content)
	if strings.HasPrefix(trimmed, "WEBVTT") {
		return content
	}
	if strings.Contains(content, "[Script Info]") || strings.Contains(content, "[V4+ Styles]") {
		return assToVTT(content)
	}
	return srtToVTT(content)
}

func srtToVTT(srt string) string {
	var vtt strings.Builder
	vtt.WriteString("WEBVTT\n\n")
	for _, line := range strings.Split(srt, "\n") {
		if strings.Contains(line, " --> ") {
			vtt.WriteString(strings.ReplaceAll(line, ",", "."))
		} else {
			vtt.WriteString(line)
		}
		vtt.WriteByte('\n')
	}
	return vtt.String()
}

func assToVTT(ass string) string {
	var vtt strings.Builder
	vtt.WriteString("WEBVTT\n\n")
	cueNumber := 1

	for _, line := range strings.Split(ass, "\n") {
		if !strings.HasPrefix(line, "Dialogue:") {
			continue
		}
		parts := strings.SplitN(line, ",", 10)
		if len(parts) < 10 {
			continue
		}

		start := convertASSTime(parts[1])
		end := convertASSTime(parts[2])
		if start == "" || end == "" {
			continue
		}

		text := parts[9]
		text = strings.ReplaceAll(text, "\\N", "\n")
		text = strings.ReplaceAll(text, "\\n", "\n")
		// Strip style tags
		for _, tag := range []string{"{\\i1}", "{\\i0}", "{\\b1}", "{\\b0}"} {
			text = strings.ReplaceAll(text, tag, "")
		}

		fmt.Fprintf(&vtt, "%d\n%s --> %s\n%s\n\n", cueNumber, start, end, strings.TrimSpace(text))
		cueNumber++
	}
	return vtt.String()
}

func convertASSTime(t string) string {
	t = strings.TrimSpace(t)
	parts := strings.Split(t, ":")
	if len(parts) != 3 {
		return ""
	}
	var hours, minutes int
	fmt.Sscanf(parts[0], "%d", &hours)
	fmt.Sscanf(parts[1], "%d", &minutes)

	secParts := strings.Split(parts[2], ".")
	if len(secParts) != 2 {
		return ""
	}
	var seconds, centiseconds int
	fmt.Sscanf(secParts[0], "%d", &seconds)
	fmt.Sscanf(secParts[1], "%d", &centiseconds)

	return fmt.Sprintf("%02d:%02d:%02d.%03d", hours, minutes, seconds, centiseconds*10)
}

// --- Decompression helpers ---

func decompressGzip(data []byte) (string, error) {
	reader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return "", fmt.Errorf("failed to decompress gzip: %w", err)
	}
	defer reader.Close()

	result, err := io.ReadAll(reader)
	if err != nil {
		return "", fmt.Errorf("failed to read gzip data: %w", err)
	}
	return string(result), nil
}

func extractSubtitleFromZip(data []byte) (string, error) {
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return "", fmt.Errorf("failed to open ZIP archive: %w", err)
	}

	log.Printf("[SubtitleService] ZIP contains %d files", len(reader.File))

	subtitleExts := []string{".srt", ".ass", ".ssa", ".sub", ".vtt"}
	for _, f := range reader.File {
		name := strings.ToLower(f.Name)
		for _, ext := range subtitleExts {
			if strings.HasSuffix(name, ext) {
				log.Printf("[SubtitleService] Found subtitle file: %s", f.Name)
				rc, err := f.Open()
				if err != nil {
					return "", fmt.Errorf("failed to read ZIP entry: %w", err)
				}
				defer rc.Close()
				content, err := io.ReadAll(rc)
				if err != nil {
					return "", fmt.Errorf("failed to read subtitle content: %w", err)
				}
				return string(content), nil
			}
		}
	}

	return "", fmt.Errorf("no subtitle file found in ZIP archive")
}

// --- API response types ---

type subDLResponse struct {
	Status    bool            `json:"status"`
	Subtitles []subDLSubtitle `json:"subtitles"`
}

type subDLSubtitle struct {
	ReleaseName string `json:"release_name"`
	Lang        string `json:"lang"`
	Language    string `json:"language"`
	URL         string `json:"url"`
	HI          bool   `json:"hi"`
	Author      string `json:"author"`
}

type openSubtitlesResult struct {
	IDSubtitleFile     string `json:"IDSubtitleFile"`
	SubLanguageID      string `json:"SubLanguageID"`
	LanguageName       string `json:"LanguageName"`
	SubDownloadLink    string `json:"SubDownloadLink"`
	MovieReleaseName   string `json:"MovieReleaseName"`
	UserNickName       string `json:"UserNickName"`
	SubDownloadsCnt    string `json:"SubDownloadsCnt"`
	SubHearingImpaired string `json:"SubHearingImpaired"`
	MovieFPS           string `json:"MovieFPS"`
}

// osLangToISO2 converts OpenSubtitles 3-letter language codes to ISO 639-1 2-letter codes
func osLangToISO2(code string) string {
	m := map[string]string{
		"eng": "en", "spa": "es", "fre": "fr", "ger": "de", "ita": "it",
		"por": "pt", "rus": "ru", "chi": "zh", "jpn": "ja", "kor": "ko",
		"ara": "ar", "dut": "nl", "pol": "pl", "tur": "tr", "swe": "sv",
		"nor": "no", "dan": "da", "fin": "fi", "gre": "el", "heb": "he",
		"hin": "hi", "tha": "th", "vie": "vi", "ind": "id", "cze": "cs",
		"hun": "hu", "rum": "ro", "bul": "bg", "ukr": "uk", "hrv": "hr",
		"srp": "sr", "slo": "sk", "slv": "sl", "alb": "sq", "per": "fa",
		"may": "ms", "est": "et", "lav": "lv", "lit": "lt", "cat": "ca",
		"bos": "bs", "mac": "mk", "ice": "is", "geo": "ka", "arm": "hy",
	}
	if iso2, ok := m[code]; ok {
		return iso2
	}
	return code
}

// iso2ToOSLang converts ISO 639-1 2-letter codes to OpenSubtitles 3-letter codes
func iso2ToOSLang(code string) string {
	m := map[string]string{
		"en": "eng", "es": "spa", "fr": "fre", "de": "ger", "it": "ita",
		"pt": "por", "ru": "rus", "zh": "chi", "ja": "jpn", "ko": "kor",
		"ar": "ara", "nl": "dut", "pl": "pol", "tr": "tur", "sv": "swe",
		"no": "nor", "da": "dan", "fi": "fin", "el": "gre", "he": "heb",
		"hi": "hin", "th": "tha", "vi": "vie", "id": "ind", "cs": "cze",
		"hu": "hun", "ro": "rum", "bg": "bul", "uk": "ukr", "hr": "hrv",
		"sr": "srp", "sk": "slo", "sl": "slv", "sq": "alb", "fa": "per",
		"ms": "may", "et": "est", "lv": "lav", "lt": "lit", "ca": "cat",
		"bs": "bos", "mk": "mac", "is": "ice", "ka": "geo", "hy": "arm",
	}
	if os3, ok := m[code]; ok {
		return os3
	}
	return ""
}

// subdlLangToISO2 converts SubDL full language names to ISO 639-1 2-letter codes
func subdlLangToISO2(lang string) string {
	m := map[string]string{
		"english": "en", "spanish": "es", "french": "fr", "german": "de",
		"italian": "it", "portuguese": "pt", "russian": "ru", "chinese": "zh",
		"japanese": "ja", "korean": "ko", "arabic": "ar", "dutch": "nl",
		"polish": "pl", "turkish": "tr", "swedish": "sv", "norwegian": "no",
		"danish": "da", "finnish": "fi", "greek": "el", "hebrew": "he",
		"hindi": "hi", "thai": "th", "vietnamese": "vi", "indonesian": "id",
		"czech": "cs", "hungarian": "hu", "romanian": "ro", "bulgarian": "bg",
		"ukrainian": "uk", "croatian": "hr", "serbian": "sr", "slovak": "sk",
		"slovenian": "sl", "albanian": "sq", "persian": "fa", "farsi": "fa",
		"malay": "ms", "estonian": "et", "latvian": "lv", "lithuanian": "lt",
		"catalan": "ca", "bosnian": "bs", "macedonian": "mk", "icelandic": "is",
		"georgian": "ka", "armenian": "hy", "bengali": "bn", "urdu": "ur",
		"tagalog": "tl", "filipino": "tl", "swahili": "sw",
		"big 5 code": "zh", "brazillian portuguese": "pt",
	}
	if code, ok := m[lang]; ok {
		return code
	}
	// If already a 2-letter code, return as-is
	if len(lang) == 2 {
		return lang
	}
	return lang
}

// GetSubtitleLanguages returns the static list of supported subtitle languages
func GetSubtitleLanguages() []SubtitleLanguage {
	return []SubtitleLanguage{
		{Code: "en", Name: "English"},
		{Code: "es", Name: "Spanish"},
		{Code: "fr", Name: "French"},
		{Code: "de", Name: "German"},
		{Code: "it", Name: "Italian"},
		{Code: "pt", Name: "Portuguese"},
		{Code: "ru", Name: "Russian"},
		{Code: "zh", Name: "Chinese"},
		{Code: "ja", Name: "Japanese"},
		{Code: "ko", Name: "Korean"},
		{Code: "ar", Name: "Arabic"},
		{Code: "nl", Name: "Dutch"},
		{Code: "pl", Name: "Polish"},
		{Code: "tr", Name: "Turkish"},
		{Code: "sv", Name: "Swedish"},
		{Code: "no", Name: "Norwegian"},
		{Code: "da", Name: "Danish"},
		{Code: "fi", Name: "Finnish"},
		{Code: "el", Name: "Greek"},
		{Code: "he", Name: "Hebrew"},
		{Code: "hi", Name: "Hindi"},
		{Code: "th", Name: "Thai"},
		{Code: "vi", Name: "Vietnamese"},
		{Code: "id", Name: "Indonesian"},
		{Code: "cs", Name: "Czech"},
		{Code: "hu", Name: "Hungarian"},
		{Code: "ro", Name: "Romanian"},
		{Code: "bg", Name: "Bulgarian"},
		{Code: "uk", Name: "Ukrainian"},
		{Code: "hr", Name: "Croatian"},
		{Code: "sr", Name: "Serbian"},
		{Code: "sk", Name: "Slovak"},
		{Code: "sl", Name: "Slovenian"},
		{Code: "sq", Name: "Albanian"},
	}
}

type SubtitleLanguage struct {
	Code string `json:"code"`
	Name string `json:"name"`
}
