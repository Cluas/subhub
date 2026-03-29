package subscription

import (
	"fmt"
	"strings"

	"github.com/tidwall/gjson"
	"sigs.k8s.io/yaml"

	"github.com/Cluas/subhub/internal/node"
	"github.com/Cluas/subhub/internal/rule"
)

// MihomoParser parses Mihomo/Clash YAML subscription data.
type MihomoParser struct{}

var _ Parser = (*MihomoParser)(nil)

// NewMihomoParser returns a new MihomoParser.
func NewMihomoParser() *MihomoParser {
	return &MihomoParser{}
}

// Detect returns true if data contains a Mihomo/Clash subscription —
// specifically if any of proxies, proxy-providers, or rule-providers
// keys are present at the top level.
func (p *MihomoParser) Detect(data []byte) bool {
	if len(data) == 0 {
		return false
	}
	jsonData, err := yaml.YAMLToJSON(data)
	if err != nil {
		return false
	}
	j := string(jsonData)
	return gjson.Get(j, "proxies").Exists() ||
		gjson.Get(j, "proxy-providers").Exists() ||
		gjson.Get(j, "rule-providers").Exists()
}

// Parse converts Mihomo/Clash YAML bytes into a *FetchResult.
// Returns an error if data cannot be parsed as YAML.
func (p *MihomoParser) Parse(data []byte) (*FetchResult, error) {
	if len(data) == 0 {
		return &FetchResult{}, nil
	}

	jsonData, err := yaml.YAMLToJSON(data)
	if err != nil {
		return nil, fmt.Errorf("mihomo: YAML→JSON: %w", err)
	}
	j := string(jsonData)

	// --- proxies ---
	var nodes []proxy.Node
	gjson.Get(j, "proxies").ForEach(func(_, v gjson.Result) bool {
		name := v.Get("name").String()
		if name == "" {
			return true // skip unnamed proxies
		}
		raw := make(map[string]any)
		for k, field := range v.Map() {
			raw[k] = field.Value()
		}
		nodes = append(nodes, &nodeResult{
			id:     int64(len(nodes)),
			name:   name,
			config: raw,
		})
		return true
	})

	// --- rules ---
	var rules []rule.Rule
	gjson.Get(j, "rules").ForEach(func(_, v gjson.Result) bool {
		r := parseRuleString(int64(len(rules)), v.String())
		if r != nil {
			rules = append(rules, r)
		}
		return true
	})

	// --- proxy-providers ---
	var nodeProviders []ProviderDef
	gjson.Get(j, "proxy-providers").ForEach(func(k, v gjson.Result) bool {
		meta := make(map[string]any)
		for key, field := range v.Map() {
			meta[key] = field.Value()
		}
		pd := ProviderDef{
			Name:     k.String(),
			URL:      v.Get("url").String(),
			Metadata: meta,
		}
		nodeProviders = append(nodeProviders, pd)
		return true
	})

	// --- rule-providers ---
	var ruleProviders []ProviderDef
	gjson.Get(j, "rule-providers").ForEach(func(k, v gjson.Result) bool {
		meta := make(map[string]any)
		for key, field := range v.Map() {
			meta[key] = field.Value()
		}
		pd := ProviderDef{
			Name:     k.String(),
			URL:      v.Get("url").String(),
			Metadata: meta,
		}
		ruleProviders = append(ruleProviders, pd)
		return true
	})

	return &FetchResult{
		Nodes:         nodes,
		Rules:         rules,
		NodeProviders: nodeProviders,
		RuleProviders: ruleProviders,
	}, nil
}

// parseRuleString parses a single Mihomo rule string into a *ruleResult.
// Supports:
//   - "TYPE,payload,TARGET"  (3+ parts)
//   - "TYPE,TARGET"          (2 parts, no payload)
//   - "MATCH,TARGET"         (MATCH rules have no payload)
//
// Returns nil for empty strings or strings with fewer than 2 parts.
func parseRuleString(id int64, s string) *ruleResult {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	if len(parts) < 2 {
		return nil
	}

	ruleType := parts[0]
	var payload, target string

	if ruleType == "MATCH" {
		// MATCH,TARGET — no payload
		target = parts[len(parts)-1]
		payload = ""
	} else if len(parts) >= 3 {
		// TYPE,<middle>,TARGET
		payload = strings.Join(parts[1:len(parts)-1], ",")
		target = parts[len(parts)-1]
	} else {
		// Exactly 2 parts: TYPE,TARGET
		target = parts[1]
		payload = ""
	}

	return &ruleResult{
		id: id,
		match: rule.RuleMatch{
			Type:  ruleType,
			Value: payload,
		},
		target: target,
	}
}
