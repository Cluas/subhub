package model

import "time"

// Collection represents a local data collection (proxy collection or rule collection).
type Collection struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	ContentType string    `json:"content_type"` // "proxy" | "rule"
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
