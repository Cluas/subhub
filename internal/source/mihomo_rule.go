package subscription

import "github.com/Cluas/subhub/internal/rule"

// ruleResult implements rule.Rule for a single Mihomo rule string.
type ruleResult struct {
	id     int64
	match  rule.RuleMatch
	target string
}

var _ rule.Rule = (*ruleResult)(nil)

func (r *ruleResult) ID() int64              { return r.id }
func (r *ruleResult) Match() rule.RuleMatch  { return r.match }
func (r *ruleResult) Target() string         { return r.target }
