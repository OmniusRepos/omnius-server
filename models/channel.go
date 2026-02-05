package models

type Channel struct {
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	Country    string   `json:"country,omitempty"`
	Languages  []string `json:"languages,omitempty"`
	Categories []string `json:"categories,omitempty"`
	Logo       string   `json:"logo,omitempty"`
	StreamURL  string   `json:"stream_url,omitempty"`
}

type ChannelCountry struct {
	Code string `json:"code"`
	Name string `json:"name"`
	Flag string `json:"flag,omitempty"`
}

type ChannelCategory struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type ChannelListData struct {
	ChannelCount int       `json:"channel_count"`
	Limit        int       `json:"limit"`
	PageNumber   int       `json:"page_number"`
	Channels     []Channel `json:"channels"`
}
