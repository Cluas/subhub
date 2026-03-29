package model

import (
	"time"

	"github.com/Cluas/subhub/internal/profile"
)

// ProfileStrategy represents a node selection strategy group.
// Implements the profile.Group interface.
type ProfileStrategy struct {
	ID         int64          `json:"id"`
	ProfileID  int64          `json:"profile_id"`
	NameStr    string         `json:"name"`
	StrategyV  string         `json:"strategy"` // select, auto, fallback, load_balance
	PoolNames  []string       `json:"pools"`    // references ProfileNodePool.Name
	ProxyNames []string       `json:"proxies"`  // directly references node or group names
	ConfigJSON map[string]any `json:"config"`   // Mihomo-specific: filter, include-all, tolerance, url, interval
	Position   int            `json:"position"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
}

// Name implements profile.Group.
func (s *ProfileStrategy) Name() string { return s.NameStr }

// Strategy implements profile.Group.
func (s *ProfileStrategy) Strategy() string { return s.StrategyV }

// Pools implements profile.Group.
func (s *ProfileStrategy) Pools() []string { return s.PoolNames }

// Proxies implements profile.Group.
func (s *ProfileStrategy) Proxies() []string { return s.ProxyNames }

// Config implements profile.Group.
func (s *ProfileStrategy) Config() map[string]any { return s.ConfigJSON }

// Compile-time interface assertion.
var _ profile.Group = (*ProfileStrategy)(nil)
