package model

import "time"

// Profile represents a proxy configuration profile, holding name, unique slug, and client settings.
type Profile struct {
	ID               int64          `json:"id"`
	Name             string         `json:"name"`
	Slug             string         `json:"slug"`
	Settings         map[string]any `json:"settings"` // Mihomo top-level settings: mixed-port, mode, log-level, etc.
	NodePoolCount    int            `json:"node_pool_count"`
	RuleSetCount     int            `json:"rule_set_count"`
	StrategyCount    int            `json:"strategy_count"`
	RoutingRuleCount int            `json:"routing_rule_count"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
}
