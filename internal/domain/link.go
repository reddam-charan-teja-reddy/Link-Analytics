package domain

import "time"

type Link struct {
	ID           string       `json:"id"`
	UserID       string       `json:"user_id"`
	OriginalURL  string       `json:"original_url"`
	Hash         string       `json:"hash"`
	Title        string       `json:"title,omitempty"`
	IsActive     bool         `json:"is_active"`
	TotalClicks  int          `json:"total_clicks,omitempty"`
	LastClickedAt *time.Time  `json:"last_clicked_at,omitempty"`
	CreatedAt    time.Time    `json:"created_at"`
	UpdatedAt    time.Time    `json:"updated_at"`
	Sources      []SourceLink `json:"sources,omitempty"`
}
