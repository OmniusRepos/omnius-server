package providers

import (
	"sort"
	"sync"
)

// AggregatorResult wraps TorrentResult with parsed season/episode for series
type AggregatorResult struct {
	TorrentResult
	Season  int `json:"season,omitempty"`
	Episode int `json:"episode,omitempty"`
}

// Aggregator searches multiple providers in parallel and merges results
type Aggregator struct {
	providers []TorrentProvider
	tpb       *TPBProvider
}

func NewAggregator() *Aggregator {
	return &Aggregator{
		providers: []TorrentProvider{
			NewYTSProvider(),
			NewEZTVProvider(),
			NewL337xProvider(),
		},
		tpb: NewTPBProvider(),
	}
}

// SearchMovie searches all providers for movie torrents
func (a *Aggregator) SearchMovie(title string, year int) []AggregatorResult {
	var mu sync.Mutex
	var allResults []AggregatorResult
	var wg sync.WaitGroup

	// Search standard providers
	for _, p := range a.providers {
		wg.Add(1)
		go func(provider TorrentProvider) {
			defer wg.Done()
			results, err := provider.SearchMovie(title, year)
			if err != nil {
				return
			}
			mu.Lock()
			for _, r := range results {
				allResults = append(allResults, AggregatorResult{TorrentResult: r})
			}
			mu.Unlock()
		}(p)
	}

	// Search TPB
	wg.Add(1)
	go func() {
		defer wg.Done()
		results, err := a.tpb.SearchMovie(title, year)
		if err != nil {
			return
		}
		mu.Lock()
		for _, r := range results {
			allResults = append(allResults, AggregatorResult{TorrentResult: r})
		}
		mu.Unlock()
	}()

	wg.Wait()
	return rankResults(allResults)
}

// SearchSeries searches all providers for series torrents
func (a *Aggregator) SearchSeries(title string, season, episode int) []AggregatorResult {
	var mu sync.Mutex
	var allResults []AggregatorResult
	var wg sync.WaitGroup

	// Search standard providers
	for _, p := range a.providers {
		wg.Add(1)
		go func(provider TorrentProvider) {
			defer wg.Done()
			results, err := provider.SearchSeries(title, season, episode)
			if err != nil {
				return
			}
			mu.Lock()
			for _, r := range results {
				s, e := ParseSeasonEpisode(r.Title)
				allResults = append(allResults, AggregatorResult{
					TorrentResult: r,
					Season:        s,
					Episode:       e,
				})
			}
			mu.Unlock()
		}(p)
	}

	// Search TPB
	wg.Add(1)
	go func() {
		defer wg.Done()
		results, err := a.tpb.SearchSeries(title, season, episode)
		if err != nil {
			return
		}
		mu.Lock()
		for _, r := range results {
			s, e := ParseSeasonEpisode(r.Title)
			allResults = append(allResults, AggregatorResult{
				TorrentResult: r,
				Season:        s,
				Episode:       e,
			})
		}
		mu.Unlock()
	}()

	wg.Wait()
	return rankResults(allResults)
}

// SearchByIMDB searches EZTV by IMDB ID + TPB by query for comprehensive series results
func (a *Aggregator) SearchSeriesByIMDB(imdbID, title string) []AggregatorResult {
	var mu sync.Mutex
	var allResults []AggregatorResult
	var wg sync.WaitGroup

	// EZTV by IMDB
	wg.Add(1)
	go func() {
		defer wg.Done()
		results, err := FetchEZTVTorrents(imdbID)
		if err != nil {
			return
		}
		mu.Lock()
		for _, r := range results {
			allResults = append(allResults, AggregatorResult{
				TorrentResult: TorrentResult{
					Title:     r.Title,
					Hash:      r.Hash,
					MagnetURL: r.MagnetURL,
					Quality:   r.Quality,
					Type:      "hdtv",
					Seeds:     uint(r.Seeds),
					Peers:     uint(r.Peers),
					Size:      r.Size,
					SizeBytes: r.SizeBytes,
					Source:    "EZTV",
				},
				Season:  r.Season,
				Episode: r.Episode,
			})
		}
		mu.Unlock()
	}()

	// TPB search by title
	if title != "" {
		wg.Add(1)
		go func() {
			defer wg.Done()
			results, err := a.tpb.SearchSeries(title, 0, 0)
			if err != nil {
				return
			}
			mu.Lock()
			for _, r := range results {
				s, e := ParseSeasonEpisode(r.Title)
				allResults = append(allResults, AggregatorResult{
					TorrentResult: r,
					Season:        s,
					Episode:       e,
				})
			}
			mu.Unlock()
		}()

		// 1337x search
		wg.Add(1)
		go func() {
			defer wg.Done()
			l337x := NewL337xProvider()
			results, err := l337x.SearchSeries(title, 0, 0)
			if err != nil {
				return
			}
			mu.Lock()
			for _, r := range results {
				s, e := ParseSeasonEpisode(r.Title)
				allResults = append(allResults, AggregatorResult{
					TorrentResult: r,
					Season:        s,
					Episode:       e,
				})
			}
			mu.Unlock()
		}()
	}

	wg.Wait()
	return rankResults(allResults)
}

// rankResults deduplicates by hash and sorts by seeds descending
func rankResults(results []AggregatorResult) []AggregatorResult {
	// Deduplicate by hash, keep the one with more seeds
	seen := make(map[string]int) // hash -> index in deduped
	var deduped []AggregatorResult

	for _, r := range results {
		if idx, ok := seen[r.Hash]; ok {
			// Keep the one with more seeds
			if r.Seeds > deduped[idx].Seeds {
				deduped[idx] = r
			}
		} else {
			seen[r.Hash] = len(deduped)
			deduped = append(deduped, r)
		}
	}

	// Sort by seeds descending
	sort.Slice(deduped, func(i, j int) bool {
		return deduped[i].Seeds > deduped[j].Seeds
	})

	return deduped
}
