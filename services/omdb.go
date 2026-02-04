package services

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"torrent-server/models"
)

type OMDBService struct {
	apiKey  string
	baseURL string
}

type OMDBResponse struct {
	Title      string `json:"Title"`
	Year       string `json:"Year"`
	Rated      string `json:"Rated"`
	Released   string `json:"Released"`
	Runtime    string `json:"Runtime"`
	Genre      string `json:"Genre"`
	Director   string `json:"Director"`
	Writer     string `json:"Writer"`
	Actors     string `json:"Actors"`
	Plot       string `json:"Plot"`
	Language   string `json:"Language"`
	Country    string `json:"Country"`
	Awards     string `json:"Awards"`
	Poster     string `json:"Poster"`
	ImdbRating string `json:"imdbRating"`
	ImdbVotes  string `json:"imdbVotes"`
	ImdbID     string `json:"imdbID"`
	Type       string `json:"Type"`
	Response   string `json:"Response"`
	Error      string `json:"Error"`
}

func NewOMDBService(apiKey string) *OMDBService {
	return &OMDBService{
		apiKey:  apiKey,
		baseURL: "https://www.omdbapi.com/",
	}
}

func (s *OMDBService) IsConfigured() bool {
	return s.apiKey != ""
}

func (s *OMDBService) FetchByIMDB(imdbCode string) (*models.Movie, error) {
	if !s.IsConfigured() {
		return nil, fmt.Errorf("OMDB API key not configured")
	}

	params := url.Values{}
	params.Set("apikey", s.apiKey)
	params.Set("i", imdbCode)
	params.Set("plot", "full")

	resp, err := http.Get(s.baseURL + "?" + params.Encode())
	if err != nil {
		return nil, fmt.Errorf("failed to fetch from OMDB: %w", err)
	}
	defer resp.Body.Close()

	var omdb OMDBResponse
	if err := json.NewDecoder(resp.Body).Decode(&omdb); err != nil {
		return nil, fmt.Errorf("failed to decode OMDB response: %w", err)
	}

	if omdb.Response == "False" {
		return nil, fmt.Errorf("OMDB error: %s", omdb.Error)
	}

	return s.convertToMovie(omdb), nil
}

func (s *OMDBService) SearchByTitle(title string, year int) (*models.Movie, error) {
	if !s.IsConfigured() {
		return nil, fmt.Errorf("OMDB API key not configured")
	}

	params := url.Values{}
	params.Set("apikey", s.apiKey)
	params.Set("t", title)
	params.Set("plot", "full")
	if year > 0 {
		params.Set("y", strconv.Itoa(year))
	}

	resp, err := http.Get(s.baseURL + "?" + params.Encode())
	if err != nil {
		return nil, fmt.Errorf("failed to fetch from OMDB: %w", err)
	}
	defer resp.Body.Close()

	var omdb OMDBResponse
	if err := json.NewDecoder(resp.Body).Decode(&omdb); err != nil {
		return nil, fmt.Errorf("failed to decode OMDB response: %w", err)
	}

	if omdb.Response == "False" {
		return nil, fmt.Errorf("OMDB error: %s", omdb.Error)
	}

	return s.convertToMovie(omdb), nil
}

func (s *OMDBService) convertToMovie(omdb OMDBResponse) *models.Movie {
	year, _ := strconv.Atoi(omdb.Year)
	rating, _ := strconv.ParseFloat(omdb.ImdbRating, 32)
	runtime := parseRuntime(omdb.Runtime)

	genres := strings.Split(omdb.Genre, ", ")

	movie := &models.Movie{
		ImdbCode:         omdb.ImdbID,
		Title:            omdb.Title,
		TitleEnglish:     omdb.Title,
		TitleLong:        fmt.Sprintf("%s (%s)", omdb.Title, omdb.Year),
		Slug:             strings.ToLower(strings.ReplaceAll(omdb.Title, " ", "-")),
		Year:             uint(year),
		Rating:           float32(rating),
		Runtime:          uint(runtime),
		Genres:           genres,
		Summary:          omdb.Plot,
		DescriptionFull:  omdb.Plot,
		Synopsis:         omdb.Plot,
		Language:         strings.Split(omdb.Language, ", ")[0],
		SmallCoverImage:  omdb.Poster,
		MediumCoverImage: omdb.Poster,
		LargeCoverImage:  strings.Replace(omdb.Poster, "SX300", "SX500", 1),
	}

	return movie
}

func parseRuntime(runtime string) int {
	// Runtime is like "142 min"
	runtime = strings.TrimSuffix(runtime, " min")
	mins, _ := strconv.Atoi(runtime)
	return mins
}
