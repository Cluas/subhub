package engine

import (
	"log/slog"
	"strings"

	"github.com/Cluas/subhub/internal/model"
)

// ExpandRuleSet expands "RULE-SET,provider-name,target-group" into the actual rule list.
func ExpandRuleSet(ruleStr string, providerRulesMap map[string][]model.RuleWithSource) []model.RuleWithSource {
	parts := strings.Split(ruleStr, ",")
	if len(parts) < 3 || parts[0] != "RULE-SET" {
		return nil
	}

	providerName := strings.TrimSpace(parts[1])
	targetGroup := strings.TrimSpace(parts[2])

	providerRules, ok := providerRulesMap[providerName]
	if !ok {
		slog.Warn("RULE-SET references unknown provider", "provider", providerName)
		return nil
	}

	result := make([]model.RuleWithSource, 0, len(providerRules))
	for _, rws := range providerRules {
		ruleParts := strings.Split(rws.Rule, ",")
		if len(ruleParts) < 2 {
			continue
		}
		// Replace the target group: keep rule type and parameters, replace the last field (target group)
		typeAndParams := strings.Join(ruleParts[:len(ruleParts)-1], ",")
		result = append(result, model.RuleWithSource{
			Rule:   typeAndParams + "," + targetGroup,
			Source: providerName,
		})
	}

	slog.Debug("expanded RULE-SET", "provider", providerName, "count", len(result), "target", targetGroup)
	return result
}

// ExpandProxyGroup recursively expands proxy group references and returns all actual proxy names.
func ExpandProxyGroup(groupName string, groupMap map[string]map[string]interface{}, visited map[string]bool) []string {
	if visited[groupName] {
		slog.Warn("circular reference in proxy-group", "group", groupName)
		return nil
	}

	group, ok := groupMap[groupName]
	if !ok {
		return []string{groupName} // not a proxy group; it's a proxy name or special value (DIRECT/REJECT)
	}

	visited[groupName] = true
	defer delete(visited, groupName)

	proxies, ok := group["proxies"].([]interface{})
	if !ok {
		return nil
	}

	seen := make(map[string]bool)
	var result []string
	for _, ref := range proxies {
		name, ok := ref.(string)
		if !ok {
			continue
		}
		var expanded []string
		if _, isGroup := groupMap[name]; isGroup {
			expanded = ExpandProxyGroup(name, groupMap, visited)
		} else {
			expanded = []string{name}
		}
		for _, p := range expanded {
			if !seen[p] {
				result = append(result, p)
				seen[p] = true
			}
		}
	}
	return result
}

// ExpandAllProxyGroups expands nested proxy group references in all proxy groups (in-place).
func ExpandAllProxyGroups(groups []map[string]interface{}) {
	groupMap := make(map[string]map[string]interface{})
	for _, g := range groups {
		if name, ok := g["name"].(string); ok && name != "" {
			groupMap[name] = g
		}
	}

	for _, g := range groups {
		name, _ := g["name"].(string)
		if name == "" {
			continue
		}
		expanded := ExpandProxyGroup(name, groupMap, make(map[string]bool))
		proxyList := make([]interface{}, len(expanded))
		for i, p := range expanded {
			proxyList[i] = p
		}
		g["proxies"] = proxyList
		slog.Debug("expanded proxy-group", "name", name, "count", len(expanded))
	}
}
