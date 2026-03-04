package domain

import "time"

type Group struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Name      string    `json:"name"`
	LinkCount int       `json:"link_count,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}
