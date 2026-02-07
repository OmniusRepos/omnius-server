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
	"strings"
	"time"

	"torrent-server/database"
	"torrent-server/models"
)

const (
	subdlAPIURL          = "https://api.subdl.com/api/v1/subtitles"
	opensubtitlesRestURL = "https://rest.opensubtitles.org/search"
)

type SubtitleService struct {
	client   *http.Client
	subdlKey string
	db       *database.DB
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

func NewSubtitleServiceWithDB(db *database.DB) *SubtitleService {
	s := NewSubtitleService()
	s.db = db
	return s
}

// SyncSubtitles downloads and stores subtitles for a given IMDB code.
// Returns the number of subtitles stored.
func (s *SubtitleService) SyncSubtitles(imdbCode string, languages string) (int, error) {
	if s.db == nil {
		return 0, fmt.Errorf("no database configured")
	}

	// Check if already synced
	count, _ := s.db.CountSubtitlesByIMDB(imdbCode)
	if count > 0 {
		log.Printf("[SubtitleSync] Already have %d subtitles for %s, skipping", count, imdbCode)
		return count, nil
	}

	log.Printf("[SubtitleSync] Syncing subtitles for %s (languages: %s)", imdbCode, languages)

	result, err := s.SearchByIMDB(imdbCode, languages)
	if err != nil {
		return 0, fmt.Errorf("failed to search subtitles: %w", err)
	}

	if len(result.Subtitles) == 0 {
		log.Printf("[SubtitleSync] No subtitles found for %s", imdbCode)
		return 0, nil
	}

	// Group by language, pick top 3 per language
	byLang := make(map[string][]Subtitle)
	for _, sub := range result.Subtitles {
		byLang[sub.Language] = append(byLang[sub.Language], sub)
	}

	stored := 0
	for lang, subs := range byLang {
		limit := 3
		if len(subs) < limit {
			limit = len(subs)
		}
		for i := 0; i < limit; i++ {
			sub := subs[i]
			vtt, err := s.DownloadSubtitle(sub.DownloadURL)
			if err != nil {
				log.Printf("[SubtitleSync] Failed to download %s subtitle for %s: %v", lang, imdbCode, err)
				continue
			}

			storedSub := &models.StoredSubtitle{
				ImdbCode:        imdbCode,
				Language:        sub.Language,
				LanguageName:    sub.LanguageName,
				ReleaseName:     sub.ReleaseName,
				HearingImpaired: sub.HearingImpaired,
				Source:          "subdl",
				VTTContent:      vtt,
			}
			if err := s.db.CreateSubtitle(storedSub); err != nil {
				log.Printf("[SubtitleSync] Failed to store subtitle: %v", err)
				continue
			}
			stored++

			// Rate limiting
			time.Sleep(500 * time.Millisecond)
		}
	}

	log.Printf("[SubtitleSync] Stored %d subtitles for %s", stored, imdbCode)
	return stored, nil
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

// SearchByIMDB searches subtitles by IMDB ID, tries SubDL then OpenSubtitles
func (s *SubtitleService) SearchByIMDB(imdbID string, languages string) (*SubtitleSearchResult, error) {
	imdb := strings.TrimPrefix(imdbID, "tt")
	log.Printf("[SubtitleService] Searching subtitles for IMDB: %s, languages: %s", imdb, languages)

	// Try SubDL first
	result, err := s.searchSubDL(imdb, languages)
	if err == nil && len(result.Subtitles) > 0 {
		return result, nil
	}
	if err != nil {
		log.Printf("[SubtitleService] SubDL error: %v, trying OpenSubtitles", err)
	} else {
		log.Printf("[SubtitleService] SubDL returned no results, trying OpenSubtitles")
	}

	// Fallback to OpenSubtitles REST API
	return s.searchOpenSubtitlesREST(imdb)
}

// SearchByFilename searches subtitles by release filename
func (s *SubtitleService) SearchByFilename(filename string, languages string) (*SubtitleSearchResult, error) {
	log.Printf("[SubtitleService] Searching subtitles by filename: %s", filename)

	encodedFilename := url.QueryEscape(filename)
	apiURL := fmt.Sprintf("%s?api_key=%s&file_name=%s", subdlAPIURL, s.subdlKey, encodedFilename)
	if languages != "" {
		apiURL += "&languages=" + url.QueryEscape(languages)
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

	req, err := http.NewRequest("GET", downloadURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "OmniusServer v1.0")

	resp, err := s.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to download subtitle: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to download subtitle: HTTP %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read subtitle bytes: %w", err)
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

	// Convert to VTT
	vtt := convertToVTT(content)
	log.Printf("[SubtitleService] Converted to VTT (%d chars)", len(vtt))
	return vtt, nil
}

func (s *SubtitleService) searchSubDL(imdbID string, languages string) (*SubtitleSearchResult, error) {
	apiURL := fmt.Sprintf("%s?api_key=%s&imdb_id=tt%s", subdlAPIURL, s.subdlKey, imdbID)
	if languages != "" {
		apiURL += "&languages=" + url.QueryEscape(languages)
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
		subtitles = append(subtitles, Subtitle{
			ID:              sub.URL,
			Language:        strings.ToLower(sub.Language),
			LanguageName:    sub.Lang,
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

func (s *SubtitleService) searchOpenSubtitlesREST(imdbID string) (*SubtitleSearchResult, error) {
	apiURL := fmt.Sprintf("%s/imdbid-%s", opensubtitlesRestURL, imdbID)
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

		subtitles = append(subtitles, Subtitle{
			ID:              sub.IDSubtitleFile,
			Language:        sub.SubLanguageID,
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
