package services

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/anacrolix/torrent"
	"github.com/anacrolix/torrent/storage"
)

type speedSnapshot struct {
	bytes int64
	time  time.Time
}

type TorrentService struct {
	client      *torrent.Client
	downloadDir string
	torrents    map[string]*torrent.Torrent
	lastSnap    map[string]speedSnapshot
	mu          sync.RWMutex
}

func NewTorrentService(downloadDir string) (*TorrentService, error) {
	if err := os.MkdirAll(downloadDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create download directory: %w", err)
	}

	cfg := torrent.NewDefaultClientConfig()
	cfg.DataDir = downloadDir
	cfg.DefaultStorage = storage.NewFileByInfoHash(downloadDir)
	cfg.Seed = true

	client, err := torrent.NewClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create torrent client: %w", err)
	}

	return &TorrentService{
		client:      client,
		downloadDir: downloadDir,
		torrents:    make(map[string]*torrent.Torrent),
		lastSnap:    make(map[string]speedSnapshot),
	}, nil
}

func (s *TorrentService) Close() {
	s.client.Close()
}

func (s *TorrentService) AddMagnet(magnetURI string) (*torrent.Torrent, error) {
	t, err := s.client.AddMagnet(magnetURI)
	if err != nil {
		return nil, err
	}

	<-t.GotInfo()

	s.mu.Lock()
	s.torrents[t.InfoHash().HexString()] = t
	s.mu.Unlock()

	return t, nil
}

func (s *TorrentService) AddTorrentFile(data []byte) (*torrent.Torrent, error) {
	// Write to temp file
	tmpFile, err := os.CreateTemp("", "torrent-*.torrent")
	if err != nil {
		return nil, err
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.Write(data); err != nil {
		return nil, err
	}
	tmpFile.Close()

	t, err := s.client.AddTorrentFromFile(tmpFile.Name())
	if err != nil {
		return nil, err
	}

	<-t.GotInfo()

	s.mu.Lock()
	s.torrents[t.InfoHash().HexString()] = t
	s.mu.Unlock()

	return t, nil
}

func (s *TorrentService) GetTorrent(infoHash string) (*torrent.Torrent, bool) {
	s.mu.RLock()
	t, ok := s.torrents[infoHash]
	s.mu.RUnlock()

	if ok {
		return t, true
	}

	// Try to add from magnet if not found
	magnetURI := "magnet:?xt=urn:btih:" + infoHash
	t, err := s.client.AddMagnet(magnetURI)
	if err != nil {
		return nil, false
	}

	// Wait for info with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	select {
	case <-t.GotInfo():
		s.mu.Lock()
		s.torrents[infoHash] = t
		s.mu.Unlock()
		return t, true
	case <-ctx.Done():
		t.Drop()
		return nil, false
	}
}

// GetSpeed returns download speed in bytes/sec for a torrent by comparing snapshots.
func (s *TorrentService) GetSpeed(infoHash string, currentBytes int64) int64 {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	snap, ok := s.lastSnap[infoHash]
	if !ok || now.Sub(snap.time) < 500*time.Millisecond {
		s.lastSnap[infoHash] = speedSnapshot{bytes: currentBytes, time: now}
		return 0
	}

	elapsed := now.Sub(snap.time).Seconds()
	delta := currentBytes - snap.bytes
	s.lastSnap[infoHash] = speedSnapshot{bytes: currentBytes, time: now}

	if delta <= 0 || elapsed <= 0 {
		return 0
	}
	return int64(float64(delta) / elapsed)
}

func (s *TorrentService) RemoveTorrent(infoHash string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if t, ok := s.torrents[infoHash]; ok {
		t.Drop()
		delete(s.torrents, infoHash)
		delete(s.lastSnap, infoHash)
	}
}

// FindLargestVideoFile finds the largest video file in a torrent
func (s *TorrentService) FindLargestVideoFile(t *torrent.Torrent) (int, *torrent.File) {
	var largestFile *torrent.File
	var largestIndex int
	var largestSize int64

	videoExts := []string{".mp4", ".mkv", ".avi", ".mov", ".wmv", ".flv", ".webm", ".m4v"}

	for i, f := range t.Files() {
		ext := strings.ToLower(filepath.Ext(f.Path()))
		isVideo := false
		for _, ve := range videoExts {
			if ext == ve {
				isVideo = true
				break
			}
		}

		if isVideo && f.Length() > largestSize {
			largestSize = f.Length()
			largestFile = f
			largestIndex = i
		}
	}

	return largestIndex, largestFile
}

// GetFileReader returns a reader for a specific file in the torrent
func (s *TorrentService) GetFileReader(t *torrent.Torrent, fileIndex int) (io.ReadSeeker, int64, error) {
	files := t.Files()
	if fileIndex < 0 || fileIndex >= len(files) {
		return nil, 0, fmt.Errorf("file index out of range")
	}

	f := files[fileIndex]

	// Start downloading this file with high priority
	f.SetPriority(torrent.PiecePriorityNow)

	reader := f.NewReader()
	reader.SetReadahead(5 * 1024 * 1024) // 5MB readahead
	reader.SetResponsive()

	return reader, f.Length(), nil
}

// GetContentType returns the content type based on file extension
func GetContentType(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".mp4":
		return "video/mp4"
	case ".mkv":
		return "video/x-matroska"
	case ".avi":
		return "video/x-msvideo"
	case ".mov":
		return "video/quicktime"
	case ".wmv":
		return "video/x-ms-wmv"
	case ".flv":
		return "video/x-flv"
	case ".webm":
		return "video/webm"
	case ".m4v":
		return "video/x-m4v"
	default:
		return "application/octet-stream"
	}
}

func (s *TorrentService) GetStats() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stats := make(map[string]interface{})
	stats["active_torrents"] = len(s.torrents)

	torrentsInfo := make([]map[string]interface{}, 0)
	for hash, t := range s.torrents {
		info := map[string]interface{}{
			"hash":        hash,
			"name":        t.Name(),
			"total_size":  t.Length(),
			"downloaded":  t.BytesCompleted(),
			"num_peers":   t.Stats().ActivePeers,
			"seeding":     t.Seeding(),
		}
		torrentsInfo = append(torrentsInfo, info)
	}
	stats["torrents"] = torrentsInfo

	return stats
}

func (s *TorrentService) StartDownload(infoHash string) error {
	t, ok := s.GetTorrent(infoHash)
	if !ok {
		return fmt.Errorf("torrent not found: %s", infoHash)
	}

	log.Printf("Starting download for torrent: %s (%s)", t.Name(), infoHash)
	t.DownloadAll()
	return nil
}
