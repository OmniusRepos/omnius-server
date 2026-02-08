package services

import (
	"fmt"
	"log"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"torrent-server/database"
)

type ChannelHealthService struct {
	db     *database.DB
	mu     sync.Mutex
	status HealthCheckStatus
}

type HealthCheckStatus struct {
	Running     bool   `json:"running"`
	Phase       string `json:"phase"`
	Total       int    `json:"total"`
	Checked     int    `json:"checked"`
	Removed     int    `json:"removed"`
	StartedAt   string `json:"started_at,omitempty"`
	CompletedAt string `json:"completed_at,omitempty"`
	LastError   string `json:"last_error,omitempty"`
}

func NewChannelHealthService(db *database.DB) *ChannelHealthService {
	return &ChannelHealthService{db: db}
}

func (s *ChannelHealthService) GetStatus() HealthCheckStatus {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.status
}

func (s *ChannelHealthService) RunHealthCheck() error {
	s.mu.Lock()
	if s.status.Running {
		s.mu.Unlock()
		return fmt.Errorf("health check already in progress")
	}
	s.status = HealthCheckStatus{
		Running:   true,
		Phase:     "starting",
		StartedAt: time.Now().Format(time.RFC3339),
	}
	s.mu.Unlock()

	go func() {
		err := s.doHealthCheck()
		s.mu.Lock()
		s.status.Running = false
		s.status.CompletedAt = time.Now().Format(time.RFC3339)
		if err != nil {
			s.status.LastError = err.Error()
			s.status.Phase = "error: " + err.Error()
		} else {
			s.status.Phase = "completed"
			s.status.LastError = ""
		}
		s.mu.Unlock()
	}()

	return nil
}

func (s *ChannelHealthService) doHealthCheck() error {
	s.setPhase("fetching channels")

	channels, err := s.db.GetAllChannelsWithStreams()
	if err != nil {
		return fmt.Errorf("failed to fetch channels: %w", err)
	}

	total := len(channels)
	log.Printf("[Health Check] Starting health check for %d channels", total)

	s.mu.Lock()
	s.status.Total = total
	s.mu.Unlock()

	s.setPhase("checking streams")

	var checked atomic.Int64
	var removed atomic.Int64

	client := &http.Client{
		Timeout: 5 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 3 {
				return fmt.Errorf("too many redirects")
			}
			return nil
		},
	}

	// Semaphore for 50 concurrent workers
	sem := make(chan struct{}, 50)
	var wg sync.WaitGroup

	for _, ch := range channels {
		wg.Add(1)
		sem <- struct{}{} // acquire

		go func(id, streamURL string) {
			defer wg.Done()
			defer func() { <-sem }() // release

			alive := checkStream(client, streamURL)

			currentChecked := checked.Add(1)
			if !alive {
				s.db.AddToBlocklist(id, "dead_stream")
				if err := s.db.DeleteChannel(id); err == nil {
					removed.Add(1)
				}
			}

			// Log progress every 500 channels
			if currentChecked%500 == 0 {
				s.mu.Lock()
				s.status.Checked = int(currentChecked)
				s.status.Removed = int(removed.Load())
				s.mu.Unlock()
				log.Printf("[Health Check] Progress: %d/%d checked, %d removed",
					currentChecked, total, removed.Load())
			}
		}(ch.ID, ch.StreamURL)
	}

	wg.Wait()

	finalChecked := int(checked.Load())
	finalRemoved := int(removed.Load())

	s.mu.Lock()
	s.status.Checked = finalChecked
	s.status.Removed = finalRemoved
	s.mu.Unlock()

	log.Printf("[Health Check] Completed: %d/%d checked, %d removed", finalChecked, total, finalRemoved)
	return nil
}

func (s *ChannelHealthService) setPhase(phase string) {
	s.mu.Lock()
	s.status.Phase = phase
	s.mu.Unlock()
}

func checkStream(client *http.Client, url string) bool {
	resp, err := client.Head(url)
	if err != nil {
		return false
	}
	resp.Body.Close()

	// Accept 200 and 206 (partial content, common for streams)
	return resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusPartialContent
}
