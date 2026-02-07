package models

type Channel struct {
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	Country    string   `json:"country,omitempty"`
	Languages  []string `json:"languages,omitempty"`
	Categories []string `json:"categories,omitempty"`
	Logo       string   `json:"logo,omitempty"`
	StreamURL  string   `json:"stream_url,omitempty"`
	IsNSFW     bool     `json:"is_nsfw,omitempty"`
	Website    string   `json:"website,omitempty"`
}

type ChannelCountry struct {
	Code         string `json:"code"`
	Name         string `json:"name"`
	Flag         string `json:"flag,omitempty"`
	ChannelCount int    `json:"channel_count,omitempty"`
}

type ChannelCategory struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	ChannelCount int    `json:"channel_count,omitempty"`
}

type ChannelListData struct {
	ChannelCount int       `json:"channel_count"`
	Limit        int       `json:"limit"`
	PageNumber   int       `json:"page_number"`
	Channels     []Channel `json:"channels"`
}

type ChannelEPG struct {
	ID          uint   `json:"id"`
	ChannelID   string `json:"channel_id"`
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	StartTime   string `json:"start_time"`
	EndTime     string `json:"end_time"`
}
