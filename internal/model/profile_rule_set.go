package model

import (
	"time"

	"github.com/Cluas/subhub/internal/profile"
)

// ProfileRuleSet represents a rule set -- references an endpoint (SubHub-managed) or an external URL.
// Implements the profile.RuleSet interface.
type ProfileRuleSet struct {
	ID                int64          `json:"id"`
	ProfileID         int64          `json:"profile_id"`
	NameStr           string         `json:"name"`
	EndpointID        *int64         `json:"endpoint_id"`   // nullable: SubHub-managed endpoint
	EndpointSlugValue string         `json:"endpoint_slug"` // denormalized
	ExternalURL       string         `json:"url"`           // external URL (mutually exclusive with endpoint_id)
	MetadataJSON      map[string]any `json:"metadata"`      // behavior, format, etc.
	Interval          int            `json:"interval"`      // seconds, default 86400
	Position          int            `json:"position"`
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
}

// Name implements profile.RuleSet.
func (r *ProfileRuleSet) Name() string { return r.NameStr }

// EndpointSlug implements profile.RuleSet.
func (r *ProfileRuleSet) EndpointSlug() string { return r.EndpointSlugValue }

// URL implements profile.RuleSet.
func (r *ProfileRuleSet) URL() string { return r.ExternalURL }

// Metadata implements profile.RuleSet.
func (r *ProfileRuleSet) Metadata() map[string]any { return r.MetadataJSON }

// Compile-time interface assertion.
var _ profile.RuleSet = (*ProfileRuleSet)(nil)
