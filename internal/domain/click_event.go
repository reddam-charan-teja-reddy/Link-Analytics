package domain

import "time"

type ClickEvent struct {
	ID           int64     `json:"id"`
	Hash         string    `json:"hash"`
	LinkID       string    `json:"link_id"`
	SourceLinkID *string   `json:"source_link_id,omitempty"`
	SourceName   string    `json:"source_name,omitempty"`
	IPAddress    string    `json:"ip_address,omitempty"`
	UserAgent    string    `json:"user_agent,omitempty"`
	Referer      string    `json:"referer,omitempty"`
	Browser      string    `json:"browser,omitempty"`
	OS           string    `json:"os,omitempty"`
	Country      string    `json:"country,omitempty"`
	City         string    `json:"city,omitempty"`
	IsBot        bool      `json:"is_bot"`
	ClickedAt    time.Time `json:"clicked_at"`
}
