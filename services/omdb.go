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
	Title      string       `json:"Title"`
	Year       string       `json:"Year"`
	Rated      string       `json:"Rated"`
	Released   string       `json:"Released"`
	Runtime    string       `json:"Runtime"`
	Genre      string       `json:"Genre"`
	Director   string       `json:"Director"`
	Writer     string       `json:"Writer"`
	Actors     string       `json:"Actors"`
	Plot       string       `json:"Plot"`
	Language   string       `json:"Language"`
	Country    string       `json:"Country"`
	Awards     string       `json:"Awards"`
	Poster     string       `json:"Poster"`
	Ratings    []OMDBRating `json:"Ratings"`
	Metascore  string       `json:"Metascore"`
	ImdbRating string       `json:"imdbRating"`
	ImdbVotes  string       `json:"imdbVotes"`
	ImdbID     string       `json:"imdbID"`
	Type       string       `json:"Type"`
	BoxOffice  string       `json:"BoxOffice"`
	Response   string       `json:"Response"`
	Error      string       `json:"Error"`
}

type OMDBRating struct {
	Source string `json:"Source"`
	Value  string `json:"Value"`
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

// GetContentType returns the type of content (movie, series, episode)
func (s *OMDBService) GetContentType(imdbCode string) (string, error) {
	if !s.IsConfigured() {
		return "", fmt.Errorf("OMDB API key not configured")
	}

	params := url.Values{}
	params.Set("apikey", s.apiKey)
	params.Set("i", imdbCode)

	resp, err := http.Get(s.baseURL + "?" + params.Encode())
	if err != nil {
		return "", fmt.Errorf("failed to fetch from OMDB: %w", err)
	}
	defer resp.Body.Close()

	var omdb OMDBResponse
	if err := json.NewDecoder(resp.Body).Decode(&omdb); err != nil {
		return "", fmt.Errorf("failed to decode OMDB response: %w", err)
	}

	if omdb.Response == "False" {
		return "", fmt.Errorf("OMDB error: %s", omdb.Error)
	}

	return strings.ToLower(omdb.Type), nil
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

	// Check if it's actually a movie
	if strings.ToLower(omdb.Type) != "movie" {
		return nil, fmt.Errorf("not a movie: %s is a %s", imdbCode, omdb.Type)
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
		MpaRating:        omdb.Rated,
		SmallCoverImage:  omdb.Poster,
		MediumCoverImage: omdb.Poster,
		LargeCoverImage:  strings.Replace(omdb.Poster, "SX300", "SX500", 1),
		Director:         omdb.Director,
		Country:          omdb.Country,
		Awards:           omdb.Awards,
		BoxOfficeGross:   omdb.BoxOffice,
	}

	// Parse writers
	if omdb.Writer != "" {
		movie.Writers = strings.Split(omdb.Writer, ", ")
	}

	// Parse cast from actors
	if omdb.Actors != "" {
		actors := strings.Split(omdb.Actors, ", ")
		for _, actor := range actors {
			movie.Cast = append(movie.Cast, models.Cast{Name: actor})
		}
	}

	// Set IMDB rating
	if rating > 0 {
		r := float32(rating)
		movie.ImdbRating = &r
	}

	// Parse vote count
	if omdb.ImdbVotes != "" {
		movie.ImdbVotes = omdb.ImdbVotes
	}

	// Parse Metacritic from Metascore field
	if omdb.Metascore != "" && omdb.Metascore != "N/A" {
		if mc, err := strconv.Atoi(omdb.Metascore); err == nil {
			movie.Metacritic = &mc
		}
	}

	// Parse Rotten Tomatoes from Ratings array
	for _, r := range omdb.Ratings {
		if r.Source == "Rotten Tomatoes" {
			// Value is like "91%"
			rtStr := strings.TrimSuffix(r.Value, "%")
			if rt, err := strconv.Atoi(rtStr); err == nil {
				movie.RottenTomatoes = &rt
			}
		}
	}

	return movie
}

func parseRuntime(runtime string) int {
	// Runtime is like "142 min"
	runtime = strings.TrimSuffix(runtime, " min")
	mins, _ := strconv.Atoi(runtime)
	return mins
}
