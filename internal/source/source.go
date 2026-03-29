package subscription

import (
	"context"

	"github.com/Cluas/subhub/internal/node"
	"github.com/Cluas/subhub/internal/rule"
)

// Source is the abstraction for all data sources.
// A Source can produce nodes, rules, or both.
type Source interface {
	ID() int64
	Name() string
	// Fetch retrieves the latest data from the source.
	Fetch(ctx context.Context) (*FetchResult, error)
	// IsRemote reports whether the source requires network fetching (vs local manual management).
	IsRemote() bool
}

// FetchResult holds the result of a fetch operation.
type FetchResult struct {
	Nodes         []proxy.Node
	Rules         []rule.Rule
	// NodeProviders contains proxy-provider definitions discovered in the subscription (remote references).
	NodeProviders []ProviderDef
	// RuleProviders contains rule-provider definitions discovered in the subscription (remote references).
	RuleProviders []ProviderDef
}

// ProviderDef represents a remote provider definition referenced in a subscription.
type ProviderDef struct {
	Name     string
	URL      string
	Metadata map[string]any // protocol-specific metadata (behavior, format, etc.)
}
