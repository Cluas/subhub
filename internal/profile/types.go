package profile

import "github.com/Cluas/subhub/internal/rule"

// Profile is the abstraction of a configuration scheme -- a description that composes everything together.
type Profile interface {
	ID() int64
	Name() string
	Settings() map[string]any
	NodePools() []NodePool
	RuleSets() []RuleSet
	Groups() []Group
	RoutingRules() []RoutingRule
}

// NodePool represents a node pool -- "a set of nodes obtained from a source after applying filters".
type NodePool interface {
	Name() string
	// EndpointSlug returns the output endpoint slug (used for URL generation).
	EndpointSlug() string
	// HealthCheck returns the health check configuration.
	HealthCheck() HealthCheckConfig
}

// HealthCheckConfig holds health check parameters.
type HealthCheckConfig struct {
	URL      string
	Interval int // seconds
}

// RuleSet represents a rule set -- "a set of rules obtained from a source".
type RuleSet interface {
	Name() string
	// EndpointSlug returns the slug when managed by SubHub, or empty for external URLs.
	EndpointSlug() string
	// URL is used for direct external references.
	URL() string
	// Metadata holds protocol-specific metadata (behavior, format, etc.).
	Metadata() map[string]any
}

// Group represents a node selection strategy -- "pick one node from a set using a given strategy".
type Group interface {
	Name() string
	// Strategy is the selection strategy: select, auto, fallback, load_balance.
	Strategy() string
	// Pools returns the list of referenced NodePool names.
	Pools() []string
	// Proxies returns directly referenced node or group names.
	Proxies() []string
	// Config holds client-specific configuration (filter, include-all, tolerance, etc.).
	// Read by the Renderer; not parsed by the core system.
	Config() map[string]any
}

// RoutingRule represents a routing rule entry -- "when traffic matches X, route via Y".
type RoutingRule interface {
	Match() rule.RuleMatch
	Target() string // references Group.Name
	Position() int  // priority order
	// NoResolve reports whether the no-resolve flag should be appended for IP-based rules.
	NoResolve() bool
}
