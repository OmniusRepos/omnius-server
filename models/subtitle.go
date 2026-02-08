package models

type StoredSubtitle struct {
	ID              uint   `json:"id"`
	ImdbCode        string `json:"imdb_code"`
	Language        string `json:"language"`
	LanguageName    string `json:"language_name"`
	ReleaseName     string `json:"release_name"`
	HearingImpaired bool   `json:"hearing_impaired"`
	Source          string `json:"source"`
	VTTContent      string `json:"-"`
	VTTPath         string `json:"-"`
	CreatedAt       string `json:"created_at,omitempty"`
}
