package rule

// Rule represents a single routing rule.
type Rule interface {
	ID() int64
	Match() RuleMatch
	Target() string
}

// RuleMatch holds the match condition of a rule.
type RuleMatch struct {
	Type  string // domain, domain-suffix, ip-cidr, geosite, geoip, match, rule-set...
	Value string // match value
}
