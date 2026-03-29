package model

import "time"

// Proxy represents a proxy node.
type Proxy struct {
	ID             int64          `json:"id"`
	SubscriptionID *int64         `json:"subscription_id"`
	CollectionID   *int64         `json:"collection_id"`
	Name           string         `json:"name"`
	Type           string         `json:"type"`    // ss, ssr, vmess, vless, trojan, hysteria2, socks5
	Server         string         `json:"server"`
	Port           int            `json:"port"`
	Config         map[string]any `json:"config"`  // full protocol-specific configuration
	Region         string         `json:"region"`
	Latency        *int           `json:"latency"`      // ms, nil = not tested
	Alive          *bool          `json:"alive"`        // nil = not tested
	LastCheckAt    *time.Time     `json:"last_check_at"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
}

// Int64Ptr returns a pointer to the given int64 value.
// Useful for setting SubscriptionID on Proxy, Rule, and Endpoint.
func Int64Ptr(v int64) *int64 {
	return &v
}
