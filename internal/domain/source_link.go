package domain

import "time"

type SourceLink struct {
	ID         string    `json:"id"`
	LinkID     string    `json:"link_id"`
	SourceName string    `json:"source_name"`
	Hash       string    `json:"hash"`
	IsActive   bool      `json:"is_active"`
	CreatedAt  time.Time `json:"created_at"`
}
