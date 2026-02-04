package models

// ApiResponse is the standard YTS-compatible API response wrapper
type ApiResponse[T any] struct {
	Status        string `json:"status"`
	StatusMessage string `json:"status_message"`
	Data          T      `json:"data"`
}

// NewSuccessResponse creates a successful API response
func NewSuccessResponse[T any](data T) ApiResponse[T] {
	return ApiResponse[T]{
		Status:        "ok",
		StatusMessage: "Query was successful",
		Data:          data,
	}
}

// NewErrorResponse creates an error API response
func NewErrorResponse(message string) ApiResponse[any] {
	return ApiResponse[any]{
		Status:        "error",
		StatusMessage: message,
		Data:          nil,
	}
}

// StremioManifest represents a Stremio addon manifest
type StremioManifest struct {
	ID          string           `json:"id"`
	Version     string           `json:"version"`
	Name        string           `json:"name"`
	Description string           `json:"description"`
	Resources   []string         `json:"resources"`
	Types       []string         `json:"types"`
	Catalogs    []StremioCatalog `json:"catalogs"`
	IDPrefixes  []string         `json:"idPrefixes,omitempty"`
}

type StremioCatalog struct {
	Type string `json:"type"`
	ID   string `json:"id"`
	Name string `json:"name"`
}

type StremioMeta struct {
	ID     string `json:"id"`
	Type   string `json:"type"`
	Name   string `json:"name"`
	Poster string `json:"poster,omitempty"`
	Year   uint   `json:"year,omitempty"`
}

type StremioStream struct {
	InfoHash  string `json:"infoHash,omitempty"`
	FileIdx   int    `json:"fileIdx,omitempty"`
	URL       string `json:"url,omitempty"`
	Title     string `json:"title,omitempty"`
	Name      string `json:"name,omitempty"`
}

type StremioCatalogResponse struct {
	Metas []StremioMeta `json:"metas"`
}

type StremioStreamResponse struct {
	Streams []StremioStream `json:"streams"`
}
