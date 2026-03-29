package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/Cluas/subhub/internal/model"

	"github.com/tidwall/gjson"
	"sigs.k8s.io/yaml"
)

// MergeClashConfig fetches and merges all proxy/rule providers from a subscription URL.
func MergeClashConfig(ctx context.Context, subscribeURL string) (*model.MergedConfig, error) {
	slog.Info("merging clash config", "url", subscribeURL)

	mainJSON, err := FetchClashConfig(ctx, subscribeURL)
	if err != nil {
		return nil, fmt.Errorf("fetch main config: %w", err)
	}

	visited := map[string]bool{subscribeURL: true}

	providerProxies, _ := ResolveProxyProviders(ctx, mainJSON, visited)
	providerRules, _ := ResolveRuleProviders(ctx, mainJSON, visited)

	merged := &model.MergedConfig{
		Proxies:     make([]map[string]interface{}, 0),
		ProxyGroups: make([]map[string]interface{}, 0),
		Rules:       make([]model.RuleWithSource, 0),
	}

	// Merge proxies (provider proxies + main config proxies; main config takes precedence)
	proxyMap := make(map[string]map[string]interface{})
	for _, p := range providerProxies {
		if n, ok := p["name"].(string); ok && n != "" {
			proxyMap[n] = p
		}
	}
	for _, proxy := range gjson.GetBytes(mainJSON, "proxies").Array() {
		var obj map[string]interface{}
		if err := json.Unmarshal([]byte(proxy.Raw), &obj); err != nil {
			continue
		}
		if n, ok := obj["name"].(string); ok && n != "" {
			proxyMap[n] = obj
		}
	}
	for _, p := range proxyMap {
		merged.Proxies = append(merged.Proxies, p)
	}

	// Merge proxy groups
	groupMap := make(map[string]map[string]interface{})
	for _, group := range gjson.GetBytes(mainJSON, "proxy-groups").Array() {
		var obj map[string]interface{}
		if err := json.Unmarshal([]byte(group.Raw), &obj); err != nil {
			continue
		}
		if n, ok := obj["name"].(string); ok && n != "" {
			groupMap[n] = obj
		}
	}
	for _, g := range groupMap {
		merged.ProxyGroups = append(merged.ProxyGroups, g)
	}
	ExpandAllProxyGroups(merged.ProxyGroups)

	// Build rule-provider map for RULE-SET expansion
	providerRulesMap := make(map[string][]model.RuleWithSource)
	for _, r := range providerRules {
		providerRulesMap[r.Source] = append(providerRulesMap[r.Source], r)
	}

	// Merge rules (provider rules + main config rules, expanding RULE-SET entries)
	ruleSet := make(map[string]bool)
	for _, r := range providerRules {
		if !ruleSet[r.Rule] {
			merged.Rules = append(merged.Rules, r)
			ruleSet[r.Rule] = true
		}
	}
	for _, rule := range gjson.GetBytes(mainJSON, "rules").Array() {
		ruleStr := rule.String()
		if strings.HasPrefix(ruleStr, "RULE-SET,") {
			for _, expanded := range ExpandRuleSet(ruleStr, providerRulesMap) {
				if !ruleSet[expanded.Rule] {
					merged.Rules = append(merged.Rules, expanded)
					ruleSet[expanded.Rule] = true
				}
			}
		} else {
			if !ruleSet[ruleStr] {
				merged.Rules = append(merged.Rules, model.RuleWithSource{Rule: ruleStr, Source: "main"})
				ruleSet[ruleStr] = true
			}
		}
	}

	slog.Info("config merged", "proxies", len(merged.Proxies), "groups", len(merged.ProxyGroups), "rules", len(merged.Rules))
	return merged, nil
}

// ConvertToRuleProvider converts a subscription into rule-provider YAML.
func ConvertToRuleProvider(ctx context.Context, subscribeURL string, filters []string) (string, error) {
	merged, err := MergeClashConfig(ctx, subscribeURL)
	if err != nil {
		return "", err
	}

	provider := &model.Provider{Payload: make([]string, 0)}

	for _, rws := range merged.Rules {
		if !matchesAnyFilter(rws, filters) {
			continue
		}
		parts := strings.Split(rws.Rule, ",")
		if len(parts) == 0 || parts[0] == "MATCH" {
			continue
		}
		// Strip the last element (target group)
		if len(parts) > 1 {
			parts = parts[:len(parts)-1]
		}
		provider.Payload = append(provider.Payload, strings.Join(parts, ","))
	}

	out, err := yaml.Marshal(provider)
	if err != nil {
		return "", fmt.Errorf("marshal yaml: %w", err)
	}
	return string(out), nil
}

// ConvertToProxyProvider converts a subscription into proxy-provider YAML.
func ConvertToProxyProvider(ctx context.Context, subscribeURL string, filters []string) (string, error) {
	merged, err := MergeClashConfig(ctx, subscribeURL)
	if err != nil {
		return "", err
	}

	provider := &model.Provider{Proxies: make([]map[string]interface{}, 0)}

	if len(filters) == 0 {
		provider.Proxies = append(provider.Proxies, merged.Proxies...)
	} else {
		// Filter by proxy group name (group name contains filter)
		proxyMap := make(map[string]map[string]interface{})
		for _, p := range merged.Proxies {
			if n, ok := p["name"].(string); ok {
				proxyMap[n] = p
			}
		}
		added := make(map[string]bool)

		for _, group := range merged.ProxyGroups {
			groupName, _ := group["name"].(string)
			if !containsAny(groupName, filters) {
				continue
			}
			proxies, _ := group["proxies"].([]interface{})
			for _, ref := range proxies {
				name, ok := ref.(string)
				if !ok {
					continue
				}
				if p, ok := proxyMap[name]; ok && !added[name] {
					provider.Proxies = append(provider.Proxies, p)
					added[name] = true
				}
			}
		}

		// Filter directly by proxy name (proxy name contains filter)
		for _, f := range filters {
			for name, p := range proxyMap {
				if strings.Contains(name, f) && !added[name] {
					provider.Proxies = append(provider.Proxies, p)
					added[name] = true
				}
			}
		}
	}

	out, err := yaml.Marshal(provider)
	if err != nil {
		return "", fmt.Errorf("marshal yaml: %w", err)
	}
	return string(out), nil
}

// DeduplicateProxies deduplicates proxies by server:port and returns a unique list.
func DeduplicateProxies(proxies []map[string]interface{}) []map[string]interface{} {
	seen := make(map[string]bool)
	result := make([]map[string]interface{}, 0, len(proxies))
	for _, p := range proxies {
		server, _ := p["server"].(string)
		port := fmt.Sprintf("%v", p["port"])
		key := server + ":" + port
		if !seen[key] {
			seen[key] = true
			result = append(result, p)
		}
	}
	return result
}

func matchesAnyFilter(rws model.RuleWithSource, filters []string) bool {
	if len(filters) == 0 {
		return true
	}
	for _, f := range filters {
		if f == "" {
			continue
		}
		if rws.Source == f || strings.Contains(rws.Rule, f) {
			return true
		}
	}
	return false
}

func containsAny(s string, filters []string) bool {
	for _, f := range filters {
		if strings.Contains(s, f) {
			return true
		}
	}
	return false
}
