package models

type Torrent struct {
	ID               uint   `json:"-" gorm:"primaryKey"`
	MovieID          uint   `json:"-" gorm:"index;not null"`
	URL              string `json:"url"`
	Hash             string `json:"hash" gorm:"index;not null"`
	Quality          string `json:"quality"`
	Type             string `json:"type" gorm:"default:'web'"`
	VideoCodec       string `json:"video_codec,omitempty"`
	Seeds            uint   `json:"seeds" gorm:"default:0"`
	Peers            uint   `json:"peers" gorm:"default:0"`
	Size             string `json:"size"`
	SizeBytes        uint64 `json:"size_bytes"`
	DateUploaded     string `json:"date_uploaded"`
	DateUploadedUnix int64  `json:"date_uploaded_unix"`
}

// MagnetURL generates a magnet link for this torrent
func (t *Torrent) MagnetURL() string {
	return "magnet:?xt=urn:btih:" + t.Hash
}
