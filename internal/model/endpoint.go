package model

import "time"

// Endpoint represents a proxy output endpoint, holding slug, format, and filter configuration.
type Endpoint struct {
	ID             int64           `json:"id"`
	Name           string          `json:"name"`
	Slug           string          `json:"slug"`
	SubscriptionID *int64          `json:"subscription_id"`
	CollectionID   *int64          `json:"collection_id"`
	OutputType     string          `json:"output_type"` // "proxy" | "rule"
	Format         string          `json:"format"`      // "clash" | "surge" | "shadowrocket"
	Filters        EndpointFilters `json:"filters"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
}

// EndpointFilters holds endpoint output filter criteria, serialized as a JSON blob in the DB.
type EndpointFilters struct {
	Regions      []string `json:"regions,omitempty"`
	Types        []string `json:"types,omitempty"`
	Groups       []string `json:"groups,omitempty"`
	NameContains string   `json:"name_contains,omitempty"`
	LatencyMax   int      `json:"latency_max,omitempty"`
	AliveOnly    bool     `json:"alive_only"`
	Target       string   `json:"target,omitempty"`
	Source       string   `json:"source,omitempty"`
	RuleType     string   `json:"rule_type,omitempty"`
	Keyword      string   `json:"keyword,omitempty"`
}
