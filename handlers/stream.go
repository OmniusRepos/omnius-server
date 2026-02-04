package handlers

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"torrent-server/services"
)

type StreamHandler struct {
	torrentService *services.TorrentService
}

func NewStreamHandler(ts *services.TorrentService) *StreamHandler {
	return &StreamHandler{torrentService: ts}
}

// Stream handles GET /stream/{infoHash}/{fileIndex}
func (h *StreamHandler) Stream(w http.ResponseWriter, r *http.Request) {
	infoHash := chi.URLParam(r, "infoHash")
	fileIndexStr := chi.URLParam(r, "fileIndex")

	fileIndex, err := strconv.Atoi(fileIndexStr)
	if err != nil {
		fileIndex = -1 // Will auto-select largest video file
	}

	t, ok := h.torrentService.GetTorrent(infoHash)
	if !ok {
		http.Error(w, "Torrent not found or failed to load", http.StatusNotFound)
		return
	}

	// If fileIndex is -1, find the largest video file
	if fileIndex < 0 {
		fileIndex, _ = h.torrentService.FindLargestVideoFile(t)
	}

	files := t.Files()
	if fileIndex >= len(files) {
		http.Error(w, "File index out of range", http.StatusBadRequest)
		return
	}

	file := files[fileIndex]
	fileSize := file.Length()
	fileName := filepath.Base(file.Path())

	reader, _, err := h.torrentService.GetFileReader(t, fileIndex)
	if err != nil {
		http.Error(w, "Failed to get file reader: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Set content type based on extension
	contentType := services.GetContentType(fileName)
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Accept-Ranges", "bytes")

	// Parse Range header
	rangeHeader := r.Header.Get("Range")
	if rangeHeader == "" {
		// No range request, serve entire file
		w.Header().Set("Content-Length", strconv.FormatInt(fileSize, 10))
		w.WriteHeader(http.StatusOK)
		io.Copy(w, reader)
		return
	}

	// Parse range header: "bytes=start-end"
	start, end := parseRangeHeader(rangeHeader, fileSize)
	if start < 0 {
		http.Error(w, "Invalid range", http.StatusRequestedRangeNotSatisfiable)
		return
	}

	// Seek to start position
	if _, err := reader.Seek(start, io.SeekStart); err != nil {
		http.Error(w, "Seek failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	contentLength := end - start + 1
	w.Header().Set("Content-Length", strconv.FormatInt(contentLength, 10))
	w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, fileSize))
	w.WriteHeader(http.StatusPartialContent)

	// Copy only the requested range
	io.CopyN(w, reader, contentLength)

	log.Printf("Streamed %s bytes %d-%d/%d", fileName, start, end, fileSize)
}

// parseRangeHeader parses a Range header like "bytes=0-1023" or "bytes=0-"
func parseRangeHeader(rangeHeader string, fileSize int64) (int64, int64) {
	if !strings.HasPrefix(rangeHeader, "bytes=") {
		return -1, -1
	}

	rangeSpec := strings.TrimPrefix(rangeHeader, "bytes=")
	parts := strings.Split(rangeSpec, "-")
	if len(parts) != 2 {
		return -1, -1
	}

	var start, end int64
	var err error

	if parts[0] == "" {
		// Suffix range: "-500" means last 500 bytes
		end = fileSize - 1
		suffixLen, err := strconv.ParseInt(parts[1], 10, 64)
		if err != nil {
			return -1, -1
		}
		start = fileSize - suffixLen
		if start < 0 {
			start = 0
		}
	} else {
		start, err = strconv.ParseInt(parts[0], 10, 64)
		if err != nil {
			return -1, -1
		}

		if parts[1] == "" {
			// Open-ended range: "0-" means from start to end
			end = fileSize - 1
		} else {
			end, err = strconv.ParseInt(parts[1], 10, 64)
			if err != nil {
				return -1, -1
			}
		}
	}

	// Validate range
	if start > end || start >= fileSize {
		return -1, -1
	}

	if end >= fileSize {
		end = fileSize - 1
	}

	return start, end
}

// Health handles GET /health
func (h *StreamHandler) Health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"ok"}`))
}

// Stats handles GET /stats
func (h *StreamHandler) Stats(w http.ResponseWriter, r *http.Request) {
	stats := h.torrentService.GetStats()

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"status":"ok","data":%v}`, stats)
}
