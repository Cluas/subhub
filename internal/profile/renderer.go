package profile

// RenderInput is the complete data passed to a Renderer.
type RenderInput struct {
	Settings     map[string]any
	NodePools    []NodePool
	RuleSets     []RuleSet
	Groups       []Group
	RoutingRules []RoutingRule
	// BaseURL is the SubHub base URL used to construct endpoint URLs (e.g. http://localhost:9000).
	BaseURL string
}

// Renderer translates a Profile into a specific client configuration format.
type Renderer interface {
	// Name returns the renderer name, e.g. "mihomo", "surge", "singbox".
	Name() string
	// Render produces the full configuration output.
	Render(input *RenderInput) ([]byte, error)
	// ContentType returns the output MIME type.
	ContentType() string
}
