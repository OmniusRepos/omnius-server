package models

type ServiceConfig struct {
	ID           string `json:"id" gorm:"primaryKey"`
	Label        string `json:"label" gorm:"not null"`
	Enabled      bool   `json:"enabled" gorm:"default:true"`
	Icon         string `json:"icon"`
	DisplayOrder int    `json:"display_order" gorm:"default:0"`
}

func (ServiceConfig) TableName() string { return "service_config" }
