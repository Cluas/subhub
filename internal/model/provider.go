package model

// Provider represents the output format (proxy-provider / rule-provider YAML).
type Provider struct {
	Payload []string                 `json:"payload,omitempty" yaml:"payload,omitempty"`
	Proxies []map[string]interface{} `json:"proxies,omitempty" yaml:"proxies,omitempty"`
}

// MergedConfig holds the merged configuration (used internally during parsing).
type MergedConfig struct {
	Proxies     []map[string]interface{}
	ProxyGroups []map[string]interface{}
	Rules       []RuleWithSource
}
