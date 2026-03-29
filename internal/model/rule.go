package model

// Rule represents a rule entry.
type Rule struct {
	ID             int64  `json:"id"`
	SubscriptionID *int64 `json:"subscription_id"`
	CollectionID   *int64 `json:"collection_id"`
	ProviderName   string `json:"provider_name"` // source rule-provider name
	Type           string `json:"type"`          // DOMAIN, DOMAIN-SUFFIX, IP-CIDR, GEOIP...
	Payload        string `json:"payload"`       // rule content
	Target         string `json:"target"`        // target proxy group
}

// RuleWithSource is an internal rule with source info (used during parsing).
type RuleWithSource struct {
	Rule   string // full rule string, e.g. "DOMAIN,example.com,DIRECT"
	Source string // rule-provider name; "main" for the main config
}
