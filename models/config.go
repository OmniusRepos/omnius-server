package models

type ServiceConfig struct {
	ID           string `json:"id"`
	Label        string `json:"label"`
	Enabled      bool   `json:"enabled"`
	Icon         string `json:"icon"`
	DisplayOrder int    `json:"display_order"`
}
