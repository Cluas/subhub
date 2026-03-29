package model

import (
	"time"

	"github.com/Cluas/subhub/internal/profile"
	"github.com/Cluas/subhub/internal/rule"
)

// ProfileRoutingRule represents a routing rule entry -- "when traffic matches X, route via Y".
// Implements the core.RoutingRule interface.
type ProfileRoutingRule struct {
	ID        int64     `json:"id"`
	ProfileID int64     `json:"profile_id"`
	Type      string    `json:"type"`       // DOMAIN-SUFFIX, IP-CIDR, GEOIP, RULE-SET, MATCH...
	Payload   string    `json:"payload"`    // match value, empty for MATCH
	TargetStr string    `json:"target"`     // target strategy group name
	NoResolveV bool      `json:"no_resolve"` // no-resolve flag for IP-based rules
	PositionV int       `json:"position"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Match implements profile.RoutingRule.
func (r *ProfileRoutingRule) Match() rule.RuleMatch {
	return rule.RuleMatch{Type: r.Type, Value: r.Payload}
}

// Target implements profile.RoutingRule.
func (r *ProfileRoutingRule) Target() string { return r.TargetStr }

// Position implements profile.RoutingRule.
func (r *ProfileRoutingRule) Position() int { return r.PositionV }

// NoResolve implements profile.RoutingRule.
func (r *ProfileRoutingRule) NoResolve() bool { return r.NoResolveV }

// Compile-time interface assertion.
var _ profile.RoutingRule = (*ProfileRoutingRule)(nil)
