package services

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"torrent-server/models"
)

const imdbAPIBaseURL = "https://api.imdbapi.dev"

type IMDBService struct {
	client *http.Client
}

func NewIMDBService() *IMDBService {
	return &IMDBService{
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

// Title response from /titles/{id}
type IMDBTitle struct {
	ID             string          `json:"id"`
	Type           string          `json:"type"`
	PrimaryTitle   string          `json:"primaryTitle"`
	OriginalTitle  string          `json:"originalTitle"`
	StartYear      int             `json:"startYear"`
	EndYear        *int            `json:"endYear"`
	RuntimeSeconds int             `json:"runtimeSeconds"`
	Genres         []string        `json:"genres"`
	Plot           string          `json:"plot"`
	Rating         *IMDBRating     `json:"rating"`
	Metacritic     *IMDBMetacritic `json:"metacritic"`
	PrimaryImage   *IMDBImage      `json:"primaryImage"`
	ContentRating  string          `json:"contentRating"`
	IsAdult        bool            `json:"isAdult"`
}

type IMDBMetacritic struct {
	Score       int `json:"score"`
	ReviewCount int `json:"reviewCount"`
}

type IMDBRating struct {
	AggregateRating float64 `json:"aggregateRating"`
	VoteCount       int     `json:"voteCount"`
}

type IMDBImage struct {
	URL    string `json:"url"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

// Credits response from /titles/{id}/credits
type IMDBCreditsResponse struct {
	ID      string       `json:"id"`
	Credits []IMDBCredit `json:"credits"`
}

type IMDBCredit struct {
	Category   string       `json:"category"`
	Name       *IMDBName    `json:"name"`
	Characters []string     `json:"characters"`
}

type IMDBName struct {
	ID          string     `json:"id"`
	DisplayName string     `json:"displayName"`
	PrimaryImage *IMDBImage `json:"primaryImage"`
}

// Images response from /titles/{id}/images
type IMDBImagesResponse struct {
	ID     string       `json:"id"`
	Images []IMDBImageItem `json:"images"`
}

type IMDBImageItem struct {
	ID     string `json:"id"`
	URL    string `json:"url"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
	Type   string `json:"type"`
}

// Videos response from /titles/{id}/videos
type IMDBVideosResponse struct {
	ID     string       `json:"id"`
	Videos []IMDBVideo  `json:"videos"`
}

type IMDBVideo struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Description  string `json:"description"`
	ThumbnailURL string `json:"thumbnailUrl"`
	Runtime      int    `json:"runtime"`
	ContentType  string `json:"contentType"`
	PrimaryTitle string `json:"primaryTitle"`
}

// BoxOffice response from /titles/{id}/boxOffice
type IMDBBoxOfficeResponse struct {
	ID        string          `json:"id"`
	BoxOffice *IMDBBoxOffice  `json:"boxOffice"`
}

type IMDBBoxOffice struct {
	Budget          *IMDBMoney `json:"budget"`
	OpeningWeekend  *IMDBMoney `json:"openingWeekendGross"`
	WorldwideGross  *IMDBMoney `json:"worldwideGross"`
	DomesticGross   *IMDBMoney `json:"domesticGross"`
}

type IMDBMoney struct {
	Amount   int64  `json:"amount"`
	Currency string `json:"currency"`
}

// RichMovieData combines data from multiple IMDB endpoints
type RichMovieData struct {
	Title             string
	OriginalTitle     string
	Year              int
	Runtime           int // minutes
	Genres            []string
	Plot              string
	Rating            float64
	VoteCount         int
	Metacritic        int
	ContentRating     string
	PosterURL         string
	BackgroundURL     string
	Directors         []string
	Writers           []string
	Cast              []models.Cast
	Budget            string
	BoxOfficeGross    string
	YouTubeTrailerID  string
	AllImages         []string
}

// FetchTitle gets basic title info
func (s *IMDBService) FetchTitle(imdbID string) (*IMDBTitle, error) {
	resp, err := s.client.Get(imdbAPIBaseURL + "/titles/" + imdbID)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("IMDB API returned %d", resp.StatusCode)
	}

	var title IMDBTitle
	if err := json.NewDecoder(resp.Body).Decode(&title); err != nil {
		return nil, err
	}
	return &title, nil
}

// FetchCredits gets cast and crew
func (s *IMDBService) FetchCredits(imdbID string) (*IMDBCreditsResponse, error) {
	resp, err := s.client.Get(imdbAPIBaseURL + "/titles/" + imdbID + "/credits")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("IMDB API returned %d", resp.StatusCode)
	}

	var credits IMDBCreditsResponse
	if err := json.NewDecoder(resp.Body).Decode(&credits); err != nil {
		return nil, err
	}
	return &credits, nil
}

// FetchImages gets all images for a title
func (s *IMDBService) FetchImages(imdbID string) (*IMDBImagesResponse, error) {
	resp, err := s.client.Get(imdbAPIBaseURL + "/titles/" + imdbID + "/images")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("IMDB API returned %d", resp.StatusCode)
	}

	var images IMDBImagesResponse
	if err := json.NewDecoder(resp.Body).Decode(&images); err != nil {
		return nil, err
	}
	return &images, nil
}

// FetchVideos gets trailers and videos
func (s *IMDBService) FetchVideos(imdbID string) (*IMDBVideosResponse, error) {
	resp, err := s.client.Get(imdbAPIBaseURL + "/titles/" + imdbID + "/videos")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("IMDB API returned %d", resp.StatusCode)
	}

	var videos IMDBVideosResponse
	if err := json.NewDecoder(resp.Body).Decode(&videos); err != nil {
		return nil, err
	}
	return &videos, nil
}

// FetchBoxOffice gets budget and gross
func (s *IMDBService) FetchBoxOffice(imdbID string) (*IMDBBoxOfficeResponse, error) {
	resp, err := s.client.Get(imdbAPIBaseURL + "/titles/" + imdbID + "/boxOffice")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("IMDB API returned %d", resp.StatusCode)
	}

	var boxOffice IMDBBoxOfficeResponse
	if err := json.NewDecoder(resp.Body).Decode(&boxOffice); err != nil {
		return nil, err
	}
	return &boxOffice, nil
}

// FetchRichData fetches all available data for a movie
func (s *IMDBService) FetchRichData(imdbID string) (*RichMovieData, error) {
	data := &RichMovieData{}

	// Fetch basic title info (required)
	title, err := s.FetchTitle(imdbID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch title: %w", err)
	}

	// Accept movies, TV movies, and shorts - reject TV series
	validTypes := map[string]bool{"movie": true, "tvMovie": true, "short": true, "video": true}
	if !validTypes[title.Type] {
		return nil, fmt.Errorf("not a movie: %s is a %s", imdbID, title.Type)
	}

	data.Title = title.PrimaryTitle
	data.OriginalTitle = title.OriginalTitle
	data.Year = title.StartYear
	data.Runtime = title.RuntimeSeconds / 60
	data.Genres = title.Genres
	data.Plot = title.Plot
	data.ContentRating = title.ContentRating

	if title.Rating != nil {
		data.Rating = title.Rating.AggregateRating
		data.VoteCount = title.Rating.VoteCount
	}

	if title.Metacritic != nil {
		data.Metacritic = title.Metacritic.Score
	}

	if title.PrimaryImage != nil {
		data.PosterURL = title.PrimaryImage.URL
	}

	// Fetch credits (optional - don't fail if unavailable)
	if credits, err := s.FetchCredits(imdbID); err == nil {
		data.Directors = []string{}
		data.Writers = []string{}
		data.Cast = []models.Cast{}

		for _, credit := range credits.Credits {
			if credit.Name == nil {
				continue
			}

			switch credit.Category {
			case "director":
				data.Directors = append(data.Directors, credit.Name.DisplayName)
			case "writer":
				data.Writers = append(data.Writers, credit.Name.DisplayName)
			case "actor", "actress":
				if len(data.Cast) < 10 { // Limit to top 10 actors
					cast := models.Cast{
						Name:     credit.Name.DisplayName,
						ImdbCode: credit.Name.ID,
					}
					if len(credit.Characters) > 0 {
						cast.CharacterName = credit.Characters[0]
					}
					if credit.Name.PrimaryImage != nil {
						cast.URLSmallImage = credit.Name.PrimaryImage.URL
					}
					data.Cast = append(data.Cast, cast)
				}
			}
		}
	}

	// Fetch images (optional)
	if images, err := s.FetchImages(imdbID); err == nil {
		data.AllImages = []string{}
		var posterFound, backgroundFound bool

		for _, img := range images.Images {
			data.AllImages = append(data.AllImages, img.URL)

			// Try to find a good poster (portrait)
			if !posterFound && img.Height > img.Width && data.PosterURL == "" {
				data.PosterURL = img.URL
				posterFound = true
			}

			// Try to find a good background (landscape)
			if !backgroundFound && img.Width > img.Height {
				data.BackgroundURL = img.URL
				backgroundFound = true
			}

			// Limit stored images
			if len(data.AllImages) >= 20 {
				break
			}
		}
	}

	// Fetch videos for trailer (optional)
	if videos, err := s.FetchVideos(imdbID); err == nil {
		for _, video := range videos.Videos {
			// Look for official trailer
			lowerName := strings.ToLower(video.Name)
			if strings.Contains(lowerName, "trailer") {
				// Extract YouTube ID if available (IMDB videos may have different format)
				// The video ID from IMDB isn't a YouTube ID, but we store whatever we get
				data.YouTubeTrailerID = video.ID
				break
			}
		}
	}

	// Fetch box office (optional)
	if boxOffice, err := s.FetchBoxOffice(imdbID); err == nil && boxOffice.BoxOffice != nil {
		if boxOffice.BoxOffice.Budget != nil {
			data.Budget = formatMoney(boxOffice.BoxOffice.Budget)
		}
		if boxOffice.BoxOffice.WorldwideGross != nil {
			data.BoxOfficeGross = formatMoney(boxOffice.BoxOffice.WorldwideGross)
		}
	}

	return data, nil
}

// ToMovie converts rich data to a Movie model
func (data *RichMovieData) ToMovie(imdbCode string) *models.Movie {
	movie := &models.Movie{
		ImdbCode:        imdbCode,
		Title:           data.Title,
		TitleEnglish:    data.Title,
		TitleLong:       fmt.Sprintf("%s (%d)", data.Title, data.Year),
		Slug:            strings.ToLower(strings.ReplaceAll(data.Title, " ", "-")),
		Year:            uint(data.Year),
		Runtime:         uint(data.Runtime),
		Genres:          data.Genres,
		Summary:         data.Plot,
		DescriptionFull: data.Plot,
		Synopsis:        data.Plot,
		Language:        "en",
		MpaRating:       data.ContentRating,
		Cast:            data.Cast,
	}

	// Set rating
	if data.Rating > 0 {
		rating := float32(data.Rating)
		movie.Rating = rating
		movie.ImdbRating = &rating
	}

	// Set vote count
	if data.VoteCount > 0 {
		movie.ImdbVotes = formatVotes(data.VoteCount)
	}

	// Set Metacritic
	if data.Metacritic > 0 {
		mc := data.Metacritic
		movie.Metacritic = &mc
	}

	// Set images
	if data.PosterURL != "" {
		movie.SmallCoverImage = data.PosterURL
		movie.MediumCoverImage = data.PosterURL
		movie.LargeCoverImage = data.PosterURL
	}
	if data.BackgroundURL != "" {
		movie.BackgroundImage = data.BackgroundURL
	}

	// Set trailer
	if data.YouTubeTrailerID != "" {
		movie.YtTrailerCode = data.YouTubeTrailerID
	}

	return movie
}

func formatMoney(money *IMDBMoney) string {
	if money == nil {
		return ""
	}
	if money.Amount >= 1_000_000_000 {
		return fmt.Sprintf("$%.1fB", float64(money.Amount)/1_000_000_000)
	}
	if money.Amount >= 1_000_000 {
		return fmt.Sprintf("$%.1fM", float64(money.Amount)/1_000_000)
	}
	return fmt.Sprintf("$%d", money.Amount)
}

func formatVotes(votes int) string {
	if votes >= 1_000_000 {
		return fmt.Sprintf("%.1fM", float64(votes)/1_000_000)
	}
	if votes >= 1_000 {
		return fmt.Sprintf("%.1fK", float64(votes)/1_000)
	}
	return fmt.Sprintf("%d", votes)
}

// GetContentType checks if an IMDB ID is a movie or TV series
func (s *IMDBService) GetContentType(imdbID string) (string, error) {
	title, err := s.FetchTitle(imdbID)
	if err != nil {
		return "", err
	}
	return title.Type, nil
}
