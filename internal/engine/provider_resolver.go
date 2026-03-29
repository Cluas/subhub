package engine

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/Cluas/subhub/internal/model"

	"github.com/tidwall/gjson"
)

// ResolveProxyProviders recursively resolves all proxy-providers and returns a merged proxy list (deduplicated by name).
func ResolveProxyProviders(ctx context.Context, configJSON []byte, visited map[string]bool) ([]map[string]interface{}, error) {
	providers := gjson.GetBytes(configJSON, "proxy-providers").Map()
	proxyMap := make(map[string]map[string]interface{})

	for name, provider := range providers {
		if provider.Get("type").String() != "http" {
			continue
		}
		providerURL := provider.Get("url").String()
		if providerURL == "" {
			continue
		}
		if visited[providerURL] {
			slog.Warn("circular reference in proxy-provider", "name", name, "url", providerURL)
			continue
		}

		slog.Debug("resolving proxy-provider", "name", name, "url", providerURL)
		visited[providerURL] = true

		providerJSON, err := FetchClashConfig(ctx, providerURL)
		if err != nil {
			slog.Error("failed to fetch proxy-provider", "name", name, "err", err)
			delete(visited, providerURL)
			continue
		}

		// Recursively resolve nested providers
		nested, _ := ResolveProxyProviders(ctx, providerJSON, visited)
		for _, p := range nested {
			if n, ok := p["name"].(string); ok && n != "" {
				proxyMap[n] = p
			}
		}

		// Parse proxies from the current provider
		for _, proxy := range gjson.GetBytes(providerJSON, "proxies").Array() {
			var obj map[string]interface{}
			if err := json.Unmarshal([]byte(proxy.Raw), &obj); err != nil {
				continue
			}
			if n, ok := obj["name"].(string); ok && n != "" {
				proxyMap[n] = obj
			}
		}

		delete(visited, providerURL)
	}

	result := make([]map[string]interface{}, 0, len(proxyMap))
	for _, p := range proxyMap {
		result = append(result, p)
	}
	return result, nil
}

// ResolveRuleProviders recursively resolves all rule-providers and returns a deduplicated rule list with source info.
func ResolveRuleProviders(ctx context.Context, configJSON []byte, visited map[string]bool) ([]model.RuleWithSource, error) {
	providers := gjson.GetBytes(configJSON, "rule-providers").Map()
	var allRules []model.RuleWithSource
	ruleSet := make(map[string]bool)

	for name, provider := range providers {
		if provider.Get("type").String() != "http" {
			continue
		}
		providerURL := provider.Get("url").String()
		if providerURL == "" {
			continue
		}
		if visited[providerURL] {
			slog.Warn("circular reference in rule-provider", "name", name, "url", providerURL)
			continue
		}

		slog.Debug("resolving rule-provider", "name", name, "url", providerURL)
		visited[providerURL] = true

		providerJSON, err := FetchClashConfig(ctx, providerURL)
		if err != nil {
			slog.Error("failed to fetch rule-provider", "name", name, "err", err)
			delete(visited, providerURL)
			continue
		}

		// Recursively resolve nested providers (preserving their original source)
		nested, _ := ResolveRuleProviders(ctx, providerJSON, visited)
		for _, r := range nested {
			if !ruleSet[r.Rule] {
				allRules = append(allRules, r)
				ruleSet[r.Rule] = true
			}
		}

		// Parse rules from the current provider (payload format or rules format)
		payload := gjson.GetBytes(providerJSON, "payload").Array()
		if len(payload) == 0 {
			payload = gjson.GetBytes(providerJSON, "rules").Array()
		}
		for _, r := range payload {
			rule := r.String()
			if !ruleSet[rule] {
				allRules = append(allRules, model.RuleWithSource{Rule: rule, Source: name})
				ruleSet[rule] = true
			}
		}

		delete(visited, providerURL)
	}

	return allRules, nil
}
