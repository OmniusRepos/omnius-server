package models

type Torrent struct {
	ID               uint   `json:"-"`
	MovieID          uint   `json:"-"`
	URL              string `json:"url"`
	Hash             string `json:"hash"`
	Quality          string `json:"quality"`
	Type             string `json:"type"`
	VideoCodec       string `json:"video_codec,omitempty"`
	Seeds            uint   `json:"seeds"`
	Peers            uint   `json:"peers"`
	Size             string `json:"size"`
	SizeBytes        uint64 `json:"size_bytes"`
	DateUploaded     string `json:"date_uploaded"`
	DateUploadedUnix int64  `json:"date_uploaded_unix"`
}

// MagnetURL generates a magnet link for this torrent
func (t *Torrent) MagnetURL() string {
	return "magnet:?xt=urn:btih:" + t.Hash
}
