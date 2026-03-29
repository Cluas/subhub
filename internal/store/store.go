package store

import (
	"context"
	"github.com/Cluas/subhub/internal/model"
)

// Store defines the interface for all persistence operations.
type Store interface {
	// Subscription management
	CreateSubscription(ctx context.Context, s *model.Subscription) error
	GetSubscription(ctx context.Context, id int64) (*model.Subscription, error)
	ListSubscriptions(ctx context.Context) ([]*model.Subscription, error)
	UpdateSubscription(ctx context.Context, s *model.Subscription) error
	DeleteSubscription(ctx context.Context, id int64) error

	// Proxy node management
	UpsertProxies(ctx context.Context, subscriptionID int64, proxies []*model.Proxy) error
	ListProxies(ctx context.Context, filter ProxyFilter) ([]*model.Proxy, error)
	GetProxy(ctx context.Context, id int64) (*model.Proxy, error)
	UpdateProxyHealth(ctx context.Context, id int64, alive bool, latencyMs int) error
	DeleteProxiesBySubscription(ctx context.Context, subscriptionID int64) error

	// Self-managed proxy CRUD
	CreateProxy(ctx context.Context, p *model.Proxy) error
	UpdateProxy(ctx context.Context, p *model.Proxy) error
	DeleteProxy(ctx context.Context, id int64) error

	// Rule management
	UpsertRules(ctx context.Context, subscriptionID int64, rules []*model.Rule) error
	ListRules(ctx context.Context, filter RuleFilter) ([]*model.Rule, error)
	DeleteRulesBySubscription(ctx context.Context, subscriptionID int64) error

	// Self-managed rule CRUD
	GetRule(ctx context.Context, id int64) (*model.Rule, error)
	CreateRule(ctx context.Context, r *model.Rule) error
	UpdateRule(ctx context.Context, r *model.Rule) error
	DeleteRule(ctx context.Context, id int64) error

	// Provider definition management (original proxy-provider / rule-provider from subscriptions)
	UpsertProviderDefs(ctx context.Context, subscriptionID int64, defs []*model.ProviderDefinition) error
	ListProviderDefs(ctx context.Context, subscriptionID int64, kind string) ([]*model.ProviderDefinition, error)
	DeleteProviderDefsBySubscription(ctx context.Context, subscriptionID int64) error

	// Collection management
	CreateCollection(ctx context.Context, c *model.Collection) error
	GetCollection(ctx context.Context, id int64) (*model.Collection, error)
	ListCollections(ctx context.Context) ([]*model.Collection, error)
	UpdateCollection(ctx context.Context, c *model.Collection) error
	DeleteCollection(ctx context.Context, id int64) error

	// Endpoint management
	CreateEndpoint(ctx context.Context, e *model.Endpoint) error
	GetEndpoint(ctx context.Context, id int64) (*model.Endpoint, error)
	GetEndpointBySlug(ctx context.Context, slug string) (*model.Endpoint, error)
	ListEndpoints(ctx context.Context) ([]*model.Endpoint, error)
	UpdateEndpoint(ctx context.Context, e *model.Endpoint) error
	DeleteEndpoint(ctx context.Context, id int64) error

	// Profile management
	CreateProfile(ctx context.Context, p *model.Profile) error
	GetProfile(ctx context.Context, id int64) (*model.Profile, error)
	GetProfileBySlug(ctx context.Context, slug string) (*model.Profile, error)
	ListProfiles(ctx context.Context) ([]*model.Profile, error)
	UpdateProfile(ctx context.Context, p *model.Profile) error
	DeleteProfile(ctx context.Context, id int64) error

	// Profile node pool management
	CreateProfileNodePool(ctx context.Context, np *model.ProfileNodePool) error
	GetProfileNodePool(ctx context.Context, id int64) (*model.ProfileNodePool, error)
	ListProfileNodePools(ctx context.Context, profileID int64) ([]*model.ProfileNodePool, error)
	UpdateProfileNodePool(ctx context.Context, np *model.ProfileNodePool) error
	DeleteProfileNodePool(ctx context.Context, id int64) error

	// Profile rule set management
	CreateProfileRuleSet(ctx context.Context, rs *model.ProfileRuleSet) error
	GetProfileRuleSet(ctx context.Context, id int64) (*model.ProfileRuleSet, error)
	ListProfileRuleSets(ctx context.Context, profileID int64) ([]*model.ProfileRuleSet, error)
	UpdateProfileRuleSet(ctx context.Context, rs *model.ProfileRuleSet) error
	DeleteProfileRuleSet(ctx context.Context, id int64) error

	// Profile strategy management
	CreateProfileStrategy(ctx context.Context, st *model.ProfileStrategy) error
	GetProfileStrategy(ctx context.Context, id int64) (*model.ProfileStrategy, error)
	ListProfileStrategies(ctx context.Context, profileID int64) ([]*model.ProfileStrategy, error)
	UpdateProfileStrategy(ctx context.Context, st *model.ProfileStrategy) error
	DeleteProfileStrategy(ctx context.Context, id int64) error

	// Profile routing rule management
	CreateProfileRoutingRule(ctx context.Context, rr *model.ProfileRoutingRule) error
	GetProfileRoutingRule(ctx context.Context, id int64) (*model.ProfileRoutingRule, error)
	ListProfileRoutingRules(ctx context.Context, profileID int64) ([]*model.ProfileRoutingRule, error)
	UpdateProfileRoutingRule(ctx context.Context, rr *model.ProfileRoutingRule) error
	DeleteProfileRoutingRule(ctx context.Context, id int64) error

	// System settings
	GetSystemSetting(ctx context.Context, key, fallback string) string
	SetSystemSetting(ctx context.Context, key, value string) error

	Close() error
}

// ProxyFilter defines filter criteria for proxy queries.
type ProxyFilter struct {
	SubscriptionID int64
	CollectionID   int64    // filter by collection
	Region         string
	Type           string   // filter by proxy type (ss/vmess/trojan etc.), empty means no filter
	LatencyMax     int      // max latency in ms, 0 = unlimited
	Alive          *bool
	SortByLatency  bool
	NameContains   string   // fuzzy match on node name (LIKE %name%), empty means no filter
	Names          []string // exact match list for node names (IN), empty means no filter
	// Groups is a list of proxy-group names to filter by. Since proxy-group
	// membership is not stored per row, the filter semantics are: node name
	// contains any of the group strings (OR LIKE). Empty means no filter.
	Groups []string
}

// RuleFilter defines filter criteria for rule queries.
type RuleFilter struct {
	SubscriptionID int64
	CollectionID   int64  // filter by collection
	ProviderName   string
	Type           string
	Target         string // exact match on target proxy group, empty means no filter
	Keyword        string // fuzzy match on payload or type (LIKE %keyword%), empty means no filter
}
