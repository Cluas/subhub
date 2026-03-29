package model

import "time"

// ProviderDefinition represents the original proxy-provider / rule-provider definition from a subscription.
type ProviderDefinition struct {
	ID             int64     `json:"id"`
	SubscriptionID int64     `json:"subscription_id"`
	Name           string    `json:"name"`           // provider name, e.g. "staging", "prod"
	Kind           string    `json:"kind"`            // "proxy" | "rule"
	Type           string    `json:"type"`            // "http" | "file"
	Behavior       string    `json:"behavior"`        // "domain" | "ipcidr" | "classical" (rule only)
	URL            string    `json:"url"`             // remote URL
	Interval       int       `json:"interval"`        // refresh interval (seconds)
	Path           string    `json:"path"`            // local path
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}
