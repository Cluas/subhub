package engine

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/Cluas/subhub/internal/model"
	"github.com/Cluas/subhub/internal/store"
)

// FetchAndPersist fetches the Clash config for the given subscription, parses
// proxies and rules, persists them to the store, and updates the subscription
// status.  On any fetch/parse error the subscription is updated with
// Status="error" and ErrorMsg set before returning.
func FetchAndPersist(ctx context.Context, st store.Store, sub *model.Subscription) error {
	merged, err := MergeClashConfig(ctx, sub.URL)
	if err != nil {
		sub.Status = "error"
		sub.ErrorMsg = err.Error()
		_ = st.UpdateSubscription(ctx, sub)
		slog.Error("fetch failed", "subscription_id", sub.ID, "error", err)
		return err
	}

	proxies := convertProxies(sub.ID, merged.Proxies)
	rules := convertRules(sub.ID, merged.Rules)

	if err := st.UpsertProxies(ctx, sub.ID, proxies); err != nil {
		return err
	}
	if err := st.UpsertRules(ctx, sub.ID, rules); err != nil {
		return err
	}

	// Build and persist proxy-group membership data so that group-based
	// endpoint filters can resolve group names to proxy names.
	sub.ProxyGroupsData = buildProxyGroupsData(merged.ProxyGroups)

	now := time.Now()
	sub.NodeCount = len(proxies)
	sub.Status = "active"
	sub.LastFetchAt = &now
	if err := st.UpdateSubscription(ctx, sub); err != nil {
		return err
	}

	slog.Info("fetch succeeded",
		"subscription_id", sub.ID,
		"proxies", len(proxies),
		"rules", len(rules),
	)
	return nil
}

// convertProxies converts []map[string]interface{} to []*model.Proxy.
// Maps missing a "name" key are skipped gracefully.
func convertProxies(subID int64, raw []map[string]interface{}) []*model.Proxy {
	proxies := make([]*model.Proxy, 0, len(raw))
	for _, m := range raw {
		name, ok := m["name"].(string)
		if !ok || name == "" {
			continue
		}
		typ, _ := m["type"].(string)
		server, _ := m["server"].(string)

		var port int
		switch v := m["port"].(type) {
		case float64:
			port = int(v)
		case int:
			port = v
		case int64:
			port = int(v)
		}

		proxies = append(proxies, &model.Proxy{
			SubscriptionID: model.Int64Ptr(subID),
			Name:           name,
			Type:           typ,
			Server:         server,
			Port:           port,
			Config:         m,
			Region:         "",
		})
	}
	return proxies
}

// convertRules converts []model.RuleWithSource to []*model.Rule.
//
// Rule format: "TYPE,payload,TARGET"  (3+ parts)
//              "TYPE,TARGET"          (2 parts, no payload)
//              "MATCH,TARGET"         (MATCH rules have no payload)
//
// Entries with only 1 part or that are empty are skipped.
func convertRules(subID int64, raw []model.RuleWithSource) []*model.Rule {
	rules := make([]*model.Rule, 0, len(raw))
	for _, rws := range raw {
		if rws.Rule == "" {
			continue
		}
		parts := strings.Split(rws.Rule, ",")
		if len(parts) < 2 {
			continue // malformed — skip
		}

		ruleType := parts[0]
		var payload, target string

		if ruleType == "MATCH" {
			// MATCH rules: MATCH,TARGET (no payload)
			target = parts[len(parts)-1]
			payload = ""
		} else if len(parts) >= 3 {
			// General rule: TYPE, <middle parts as payload>, TARGET
			payload = strings.Join(parts[1:len(parts)-1], ",")
			target = parts[len(parts)-1]
		} else {
			// Exactly 2 parts: TYPE,TARGET
			target = parts[1]
			payload = ""
		}

		rules = append(rules, &model.Rule{
			SubscriptionID: model.Int64Ptr(subID),
			ProviderName:   rws.Source,
			Type:           ruleType,
			Payload:        payload,
			Target:         target,
		})
	}
	return rules
}

// buildProxyGroupsData extracts proxy-group membership from the merged config's
// proxy-groups slice.  Each group entry is expected to have a "name" key and a
// "proxies" key (list of member names, already expanded by ExpandAllProxyGroups).
func buildProxyGroupsData(groups []map[string]interface{}) *model.ProxyGroupData {
	if len(groups) == 0 {
		return nil
	}
	pgd := &model.ProxyGroupData{Groups: make(map[string][]string, len(groups))}
	for _, g := range groups {
		name, ok := g["name"].(string)
		if !ok || name == "" {
			continue
		}
		proxies, ok := g["proxies"].([]interface{})
		if !ok {
			continue
		}
		members := make([]string, 0, len(proxies))
		for _, ref := range proxies {
			if s, ok := ref.(string); ok && s != "" {
				members = append(members, s)
			}
		}
		if len(members) > 0 {
			pgd.Groups[name] = members
		}
	}
	if len(pgd.Groups) == 0 {
		return nil
	}
	return pgd
}
