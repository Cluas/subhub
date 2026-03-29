package model

import (
	"time"

	"github.com/Cluas/subhub/internal/profile"
)

// ProfileNodePool represents a node pool -- references an output endpoint; the Renderer constructs a URL from the slug.
// Implements the profile.NodePool interface.
type ProfileNodePool struct {
	ID                  int64     `json:"id"`
	ProfileID           int64     `json:"profile_id"`
	NameStr             string    `json:"name"`
	EndpointID          *int64    `json:"endpoint_id"`          // nullable: references endpoints.id
	EndpointSlugValue   string    `json:"endpoint_slug"`         // denormalized to avoid JOIN
	HealthCheckURL      string    `json:"health_check_url"`
	HealthCheckInterval int       `json:"health_check_interval"` // seconds, default 300
	Position            int       `json:"position"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

// Name implements profile.NodePool.
func (p *ProfileNodePool) Name() string { return p.NameStr }

// EndpointSlug implements profile.NodePool.
func (p *ProfileNodePool) EndpointSlug() string { return p.EndpointSlugValue }

// HealthCheck implements profile.NodePool.
func (p *ProfileNodePool) HealthCheck() profile.HealthCheckConfig {
	return profile.HealthCheckConfig{
		URL:      p.HealthCheckURL,
		Interval: p.HealthCheckInterval,
	}
}

// Compile-time interface assertion.
var _ profile.NodePool = (*ProfileNodePool)(nil)
