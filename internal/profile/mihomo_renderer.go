// Package profile contains the profile rendering domain.
package profile

import (
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"

	"go.yaml.in/yaml/v3"
)

// Compile-time assertion that MihomoRenderer satisfies Renderer.
var _ Renderer = (*MihomoRenderer)(nil)

// MihomoRenderer translates a RenderInput into Mihomo/Clash YAML.
type MihomoRenderer struct{}

// Name returns the renderer identifier.
func (r *MihomoRenderer) Name() string { return "mihomo" }

// ContentType returns the MIME type for the rendered output.
func (r *MihomoRenderer) ContentType() string { return "text/yaml; charset=utf-8" }

// ─── YAML structs ─────────────────────────────────────────────────────────────

// ClashConfig is the top-level Mihomo configuration document.
// Fields follow Mihomo's conventional field order.
type ClashConfig struct {
	MixedPort          *int                       `yaml:"mixed-port,omitempty"`
	SocksPort          *int                       `yaml:"socks-port,omitempty"`
	AllowLan           *bool                      `yaml:"allow-lan,omitempty"`
	Mode               string                     `yaml:"mode,omitempty"`
	LogLevel           string                     `yaml:"log-level,omitempty"`
	ExternalController string                     `yaml:"external-controller,omitempty"`
	ProxyProviders     map[string]*ProxyProvider  `yaml:"proxy-providers,omitempty"`
	ProxyGroups        []map[string]any           `yaml:"proxy-groups,omitempty"`
	RuleProviders      map[string]*RuleProvider   `yaml:"rule-providers,omitempty"`
	Rules              []string                   `yaml:"rules,omitempty"`
}

// ProxyProvider maps to a Mihomo proxy-providers entry.
type ProxyProvider struct {
	Type        string                   `yaml:"type"`
	URL         string                   `yaml:"url"`
	Path        string                   `yaml:"path"`
	HealthCheck *MihomoHealthCheckConfig `yaml:"health-check,omitempty"`
}

// MihomoHealthCheckConfig is the health-check block inside a proxy-provider.
type MihomoHealthCheckConfig struct {
	Enable   bool   `yaml:"enable"`
	URL      string `yaml:"url"`
	Interval int    `yaml:"interval"`
}

// RuleProvider maps to a Mihomo rule-providers entry.
type RuleProvider struct {
	Type     string `yaml:"type"`
	Behavior string `yaml:"behavior,omitempty"`
	Format   string `yaml:"format,omitempty"`
	URL      string `yaml:"url"`
	Path     string `yaml:"path"`
	Interval int    `yaml:"interval,omitempty"`
}

// ─── Render ───────────────────────────────────────────────────────────────────

// Render converts a RenderInput into Mihomo YAML bytes.
func (r *MihomoRenderer) Render(input *RenderInput) ([]byte, error) {
	cfg := &ClashConfig{}

	// ── Settings ──────────────────────────────────────────────────────────────
	if input.Settings != nil {
		if v, ok := toInt(input.Settings["mixed-port"]); ok {
			cfg.MixedPort = &v
		}
		if v, ok := toInt(input.Settings["socks-port"]); ok {
			cfg.SocksPort = &v
		}
		if v, ok := toBool(input.Settings["allow-lan"]); ok {
			cfg.AllowLan = &v
		}
		if v, ok := input.Settings["mode"].(string); ok {
			cfg.Mode = v
		}
		if v, ok := input.Settings["log-level"].(string); ok {
			cfg.LogLevel = v
		}
		if v, ok := input.Settings["external-controller"].(string); ok {
			cfg.ExternalController = v
		}
	}

	// ── Node Pools → proxy-providers ──────────────────────────────────────────
	if len(input.NodePools) > 0 {
		cfg.ProxyProviders = make(map[string]*ProxyProvider, len(input.NodePools))
		for _, pool := range input.NodePools {
			slug := pool.EndpointSlug()
			hc := pool.HealthCheck()
			hcURL := hc.URL
			if hcURL == "" {
				hcURL = "http://www.gstatic.com/generate_204"
			}
			hcInterval := hc.Interval
			if hcInterval == 0 {
				hcInterval = 300
			}
			cfg.ProxyProviders[pool.Name()] = &ProxyProvider{
				Type: "http",
				URL:  input.BaseURL + "/p/" + slug,
				Path: "./proxy_provider/" + slug + ".yaml",
				HealthCheck: &MihomoHealthCheckConfig{
					Enable:   true,
					URL:      hcURL,
					Interval: hcInterval,
				},
			}
		}
	}

	// ── Rule Sets → rule-providers ─────────────────────────────────────────────
	if len(input.RuleSets) > 0 {
		cfg.RuleProviders = make(map[string]*RuleProvider, len(input.RuleSets))
		for _, rs := range input.RuleSets {
			var providerURL string
			if slug := rs.EndpointSlug(); slug != "" {
				providerURL = input.BaseURL + "/p/" + slug
			} else {
				providerURL = rs.URL()
			}

			meta := rs.Metadata()
			behavior, _ := meta["behavior"].(string)
			format, _ := meta["format"].(string)
			rpType, _ := meta["type"].(string)
			if rpType == "" {
				rpType = "http"
			}
			interval := 86400
			if v, ok := toInt(meta["interval"]); ok && v > 0 {
				interval = v
			}

			slug := rs.EndpointSlug()
			if slug == "" {
				slug = rs.Name()
			}
			cfg.RuleProviders[rs.Name()] = &RuleProvider{
				Type:     rpType,
				Behavior: behavior,
				Format:   format,
				URL:      providerURL,
				Path:     "./rule_provider/" + slug + ".yaml",
				Interval: interval,
			}
		}
	}

	// ── Groups → proxy-groups ─────────────────────────────────────────────────
	if len(input.Groups) > 0 {
		cfg.ProxyGroups = make([]map[string]any, 0, len(input.Groups))
		for _, g := range input.Groups {
			entry := map[string]any{
				"name": g.Name(),
				"type": mapStrategy(g.Strategy()),
			}

			if pools := g.Pools(); len(pools) > 0 {
				entry["use"] = pools
			}
			if proxies := g.Proxies(); len(proxies) > 0 {
				entry["proxies"] = proxies
			}

			// Merge Mihomo-specific config keys.
			for _, key := range []string{"filter", "url", "interval", "tolerance", "include-all"} {
				if v, ok := g.Config()[key]; ok {
					entry[key] = v
				}
			}

			cfg.ProxyGroups = append(cfg.ProxyGroups, entry)
		}
	}

	// ── Routing Rules → rules ─────────────────────────────────────────────────
	if len(input.RoutingRules) > 0 {
		cfg.Rules = make([]string, 0, len(input.RoutingRules))
		for _, rr := range input.RoutingRules {
			cfg.Rules = append(cfg.Rules, formatRule(rr))
		}
	}

	// ── Marshal ───────────────────────────────────────────────────────────────

	// Use yaml.Node tree so we can force DoubleQuotedStyle on all scalar
	// string values, producing consistent quoting across the entire document.
	var docNode yaml.Node
	if err := docNode.Encode(cfg); err != nil {
		return nil, fmt.Errorf("mihomo: yaml encode: %w", err)
	}
	forceQuoteStrings(&docNode)

	data, err := yaml.Marshal(&docNode)
	if err != nil {
		return nil, fmt.Errorf("mihomo: yaml marshal: %w", err)
	}

	return unescapeYAMLUnicode(data), nil
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

// mapStrategy maps internal strategy names to Mihomo type names.
func mapStrategy(s string) string {
	switch s {
	case "auto":
		return "url-test"
	case "load_balance":
		return "load-balance"
	default:
		return s // "select", "fallback" are identical in Mihomo
	}
}

// formatRule converts a RoutingRule to the Mihomo rules string format.
func formatRule(rr RoutingRule) string {
	m := rr.Match()
	target := rr.Target()
	ruleType := strings.ToUpper(m.Type)

	// MATCH type has no payload.
	if ruleType == "MATCH" {
		return fmt.Sprintf("MATCH,%s", target)
	}

	r := fmt.Sprintf("%s,%s,%s", ruleType, m.Value, target)

	// Use the NoResolve() interface method instead of a concrete type assertion.
	if rr.NoResolve() {
		r += ",no-resolve"
	}

	return r
}

// toInt type-asserts a value to int, supporting int, int64, float64 sources.
func toInt(v any) (int, bool) {
	switch x := v.(type) {
	case int:
		return x, true
	case int64:
		return int(x), true
	case float64:
		return int(x), true
	case json_number:
		// json.Number from JSON unmarshalling
		if n, err := x.Int64(); err == nil {
			return int(n), true
		}
	}
	return 0, false
}

// json_number is an alias so we don't import encoding/json just for the type check.
type json_number interface {
	Int64() (int64, error)
}

// toBool type-asserts a value to bool.
func toBool(v any) (bool, bool) {
	if b, ok := v.(bool); ok {
		return b, true
	}
	return false, false
}

// forceQuoteStrings recursively walks a yaml.Node tree and sets
// DoubleQuotedStyle on string scalar *values* only (not mapping keys),
// ensuring consistent quoting in the output YAML.
func forceQuoteStrings(n *yaml.Node) {
	if n == nil {
		return
	}
	switch n.Kind {
	case yaml.DocumentNode:
		for _, child := range n.Content {
			forceQuoteStrings(child)
		}
	case yaml.MappingNode:
		// Content alternates: key, value, key, value, ...
		for i := 0; i < len(n.Content)-1; i += 2 {
			// Skip keys (i), only process values (i+1)
			forceQuoteStrings(n.Content[i+1])
		}
	case yaml.SequenceNode:
		for _, child := range n.Content {
			forceQuoteStrings(child)
		}
	case yaml.ScalarNode:
		if n.Tag == "!!str" {
			n.Style = yaml.DoubleQuotedStyle
		}
	}
}

// unescapeYAMLUnicode replaces YAML \Uxxxxxxxx escape sequences with the
// actual UTF-8 characters. go.yaml.in/yaml/v3 escapes non-ASCII characters
// in double-quoted strings; this post-processor reverses that for readability.
var unicodeEscapeRe = regexp.MustCompile(`\\U[0-9A-Fa-f]{8}`)

func unescapeYAMLUnicode(data []byte) []byte {
	return unicodeEscapeRe.ReplaceAllFunc(data, func(match []byte) []byte {
		// match is e.g. \U0001F600 — strip the \U prefix
		hexStr := string(match[2:]) // 8 hex digits
		decoded, err := hex.DecodeString(hexStr)
		if err != nil || len(decoded) != 4 {
			return match // leave as-is on error
		}
		// Interpret as big-endian uint32 rune value.
		r := rune(uint32(decoded[0])<<24 | uint32(decoded[1])<<16 | uint32(decoded[2])<<8 | uint32(decoded[3]))
		if !utf8.ValidRune(r) {
			return match
		}
		buf := make([]byte, utf8.RuneLen(r))
		utf8.EncodeRune(buf, r)
		return buf
	})
}
