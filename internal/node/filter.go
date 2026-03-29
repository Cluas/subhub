package proxy

import "github.com/Cluas/subhub/internal/rule"

// Filter selects a subset from a data set.
type Filter interface {
	ApplyNodes(nodes []Node) []Node
	ApplyRules(rules []rule.Rule) []rule.Rule
}
