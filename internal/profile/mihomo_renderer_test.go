package profile_test

import (
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"

	"github.com/Cluas/subhub/internal/profile"
	"github.com/Cluas/subhub/internal/rule"
)

// ─── Mock implementations ─────────────────────────────────────────────────────

type mockNodePool struct {
	name        string
	slug        string
	healthCheck profile.HealthCheckConfig
}

func (m *mockNodePool) Name() string                          { return m.name }
func (m *mockNodePool) EndpointSlug() string                  { return m.slug }
func (m *mockNodePool) HealthCheck() profile.HealthCheckConfig { return m.healthCheck }

type mockRuleSet struct {
	name     string
	slug     string
	url      string
	metadata map[string]any
}

func (m *mockRuleSet) Name() string             { return m.name }
func (m *mockRuleSet) EndpointSlug() string     { return m.slug }
func (m *mockRuleSet) URL() string              { return m.url }
func (m *mockRuleSet) Metadata() map[string]any { return m.metadata }

type mockGroup struct {
	name     string
	strategy string
	pools    []string
	proxies  []string
	config   map[string]any
}

func (m *mockGroup) Name() string           { return m.name }
func (m *mockGroup) Strategy() string       { return m.strategy }
func (m *mockGroup) Pools() []string        { return m.pools }
func (m *mockGroup) Proxies() []string      { return m.proxies }
func (m *mockGroup) Config() map[string]any { return m.config }

type mockRoutingRule struct {
	match     rule.RuleMatch
	target    string
	position  int
	noResolve bool
}

func (m *mockRoutingRule) Match() rule.RuleMatch { return m.match }
func (m *mockRoutingRule) Target() string        { return m.target }
func (m *mockRoutingRule) Position() int         { return m.position }
func (m *mockRoutingRule) NoResolve() bool       { return m.noResolve }

// ─── Helpers ──────────────────────────────────────────────────────────────────

func mustRender(t *testing.T, input *profile.RenderInput) string {
	t.Helper()
	r := &profile.MihomoRenderer{}
	out, err := r.Render(input)
	if err != nil {
		t.Fatalf("Render() error: %v", err)
	}
	return string(out)
}

func mustParseYAML(t *testing.T, s string) map[string]any {
	t.Helper()
	var m map[string]any
	if err := yaml.Unmarshal([]byte(s), &m); err != nil {
		t.Fatalf("YAML parse error: %v\nYAML:\n%s", err, s)
	}
	return m
}

// ─── Tests ────────────────────────────────────────────────────────────────────

// TestRender_Empty verifies an empty RenderInput produces valid, minimal YAML
// with no proxy-providers or rules sections.
func TestRender_Empty(t *testing.T) {
	out := mustRender(t, &profile.RenderInput{})
	// Must be valid YAML.
	m := mustParseYAML(t, out)

	if _, ok := m["proxy-providers"]; ok {
		t.Error("expected no proxy-providers in empty render, got one")
	}
	if _, ok := m["rules"]; ok {
		t.Error("expected no rules in empty render, got one")
	}
	if _, ok := m["proxy-groups"]; ok {
		t.Error("expected no proxy-groups in empty render, got one")
	}
	if _, ok := m["rule-providers"]; ok {
		t.Error("expected no rule-providers in empty render, got one")
	}
}

// TestRender_Settings verifies settings fields are written correctly.
func TestRender_Settings(t *testing.T) {
	out := mustRender(t, &profile.RenderInput{
		Settings: map[string]any{
			"mixed-port":          7890,
			"socks-port":          7891,
			"allow-lan":           true,
			"mode":                "rule",
			"log-level":           "info",
			"external-controller": "127.0.0.1:9090",
		},
	})

	// YAML should begin with mixed-port (Mihomo field order).
	if !strings.HasPrefix(out, "mixed-port: 7890") {
		t.Errorf("expected output to start with 'mixed-port: 7890', got:\n%s", out)
	}

	m := mustParseYAML(t, out)
	assertYAMLInt(t, m, "mixed-port", 7890)
	assertYAMLInt(t, m, "socks-port", 7891)

	if al, ok := m["allow-lan"].(bool); !ok || !al {
		t.Errorf("expected allow-lan: true, got %v", m["allow-lan"])
	}
	if m["mode"] != "rule" {
		t.Errorf("expected mode: rule, got %v", m["mode"])
	}
	if m["log-level"] != "info" {
		t.Errorf("expected log-level: info, got %v", m["log-level"])
	}
	if m["external-controller"] != "127.0.0.1:9090" {
		t.Errorf("expected external-controller: 127.0.0.1:9090, got %v", m["external-controller"])
	}
}

// TestRender_NodePools verifies a single node pool maps to proxy-providers.
func TestRender_NodePools(t *testing.T) {
	pool := &mockNodePool{
		name: "MyPool",
		slug: "my-pool-slug",
		healthCheck: profile.HealthCheckConfig{
			URL:      "http://www.gstatic.com/generate_204",
			Interval: 300,
		},
	}
	out := mustRender(t, &profile.RenderInput{
		BaseURL:   "http://localhost:9000",
		NodePools: []profile.NodePool{pool},
	})

	if !strings.Contains(out, "proxy-providers:") {
		t.Error("expected proxy-providers section, not found")
	}
	expectedURL := "http://localhost:9000/p/my-pool-slug"
	if !strings.Contains(out, expectedURL) {
		t.Errorf("expected URL %q in output, got:\n%s", expectedURL, out)
	}
	if !strings.Contains(out, "health-check:") {
		t.Error("expected health-check block, not found")
	}
}

// TestRender_NodePools_Defaults verifies health-check defaults are applied when pool values are zero.
func TestRender_NodePools_Defaults(t *testing.T) {
	pool := &mockNodePool{
		name: "EmptyPool",
		slug: "empty-pool",
		// healthCheck left at zero values
	}
	out := mustRender(t, &profile.RenderInput{
		BaseURL:   "http://localhost:9000",
		NodePools: []profile.NodePool{pool},
	})
	if !strings.Contains(out, "http://www.gstatic.com/generate_204") {
		t.Error("expected default health-check URL, not found")
	}
	if !strings.Contains(out, "interval: 300") {
		t.Error("expected default health-check interval 300, not found")
	}
}

// TestRender_RuleSets_External verifies an external URL rule set uses the URL directly.
func TestRender_RuleSets_External(t *testing.T) {
	rs := &mockRuleSet{
		name:     "GeoIP-CN",
		slug:     "", // no SubHub slug = external
		url:      "https://cdn.example.com/rules/geoip-cn.yaml",
		metadata: map[string]any{"behavior": "ipcidr", "format": "yaml", "type": "http"},
	}
	out := mustRender(t, &profile.RenderInput{
		RuleSets: []profile.RuleSet{rs},
	})

	if !strings.Contains(out, "rule-providers:") {
		t.Error("expected rule-providers section, not found")
	}
	if !strings.Contains(out, "https://cdn.example.com/rules/geoip-cn.yaml") {
		t.Errorf("expected external URL in output, got:\n%s", out)
	}
	if !strings.Contains(out, `behavior: "ipcidr"`) {
		t.Error("expected behavior: \"ipcidr\", not found")
	}
}

// TestRender_RuleSets_SubHub verifies a SubHub-managed rule set uses the BaseURL+slug URL.
func TestRender_RuleSets_SubHub(t *testing.T) {
	rs := &mockRuleSet{
		name:     "MyRules",
		slug:     "my-rules-slug",
		url:      "", // SubHub managed
		metadata: map[string]any{"behavior": "domain", "format": "yaml"},
	}
	out := mustRender(t, &profile.RenderInput{
		BaseURL:  "http://localhost:9000",
		RuleSets: []profile.RuleSet{rs},
	})

	expectedURL := "http://localhost:9000/p/my-rules-slug"
	if !strings.Contains(out, expectedURL) {
		t.Errorf("expected SubHub URL %q in output, got:\n%s", expectedURL, out)
	}
}

// TestRender_Groups verifies a strategy group maps correctly to proxy-groups,
// including the "auto" -> "url-test" strategy mapping.
func TestRender_Groups(t *testing.T) {
	grp := &mockGroup{
		name:     "MyGroup",
		strategy: "auto",
		pools:    []string{"MyPool"},
		proxies:  []string{"DIRECT"},
		config:   map[string]any{"url": "http://www.gstatic.com/generate_204", "interval": 300},
	}
	out := mustRender(t, &profile.RenderInput{
		Groups: []profile.Group{grp},
	})

	if !strings.Contains(out, "proxy-groups:") {
		t.Error("expected proxy-groups section, not found")
	}
	// "auto" must map to "url-test"
	if !strings.Contains(out, `type: "url-test"`) {
		t.Errorf("expected 'type: \"url-test\"' (mapped from auto), got:\n%s", out)
	}
	// Pools -> "use:" field
	if !strings.Contains(out, "use:") {
		t.Errorf("expected 'use:' field for pools, got:\n%s", out)
	}
	if !strings.Contains(out, "MyPool") {
		t.Errorf("expected MyPool in use list, got:\n%s", out)
	}
}

// TestRender_RoutingRules verifies routing rules are rendered in position order
// and MATCH type has no payload.
func TestRender_RoutingRules(t *testing.T) {
	rules := []profile.RoutingRule{
		// Position 10 comes first (already sorted, as store guarantees)
		&mockRoutingRule{match: rule.RuleMatch{Type: "DOMAIN-SUFFIX", Value: "google.com"}, target: "Proxy", position: 10},
		&mockRoutingRule{match: rule.RuleMatch{Type: "MATCH"}, target: "DIRECT", position: 100},
	}
	out := mustRender(t, &profile.RenderInput{
		RoutingRules: rules,
	})

	if !strings.Contains(out, "rules:") {
		t.Error("expected rules section, not found")
	}
	if !strings.Contains(out, "DOMAIN-SUFFIX,google.com,Proxy") {
		t.Errorf("expected DOMAIN-SUFFIX rule, got:\n%s", out)
	}
	// MATCH type must not include a payload
	if !strings.Contains(out, "MATCH,DIRECT") {
		t.Errorf("expected 'MATCH,DIRECT', got:\n%s", out)
	}
	// Verify order: DOMAIN-SUFFIX must appear before MATCH in the output
	domainIdx := strings.Index(out, "DOMAIN-SUFFIX")
	matchIdx := strings.Index(out, "MATCH,DIRECT")
	if domainIdx >= matchIdx {
		t.Errorf("DOMAIN-SUFFIX rule should appear before MATCH rule")
	}
}

// TestRender_FullProfile verifies a fully populated RenderInput produces
// a valid YAML document containing all expected sections.
func TestRender_FullProfile(t *testing.T) {
	out := mustRender(t, &profile.RenderInput{
		BaseURL: "http://localhost:9000",
		Settings: map[string]any{
			"mixed-port": 7890,
			"mode":       "rule",
		},
		NodePools: []profile.NodePool{
			&mockNodePool{name: "HK", slug: "hk-pool"},
		},
		RuleSets: []profile.RuleSet{
			&mockRuleSet{
				name:     "CN",
				slug:     "cn-rules",
				metadata: map[string]any{"behavior": "domain"},
			},
		},
		Groups: []profile.Group{
			&mockGroup{name: "HKGroup", strategy: "select", pools: []string{"HK"}},
		},
		RoutingRules: []profile.RoutingRule{
			&mockRoutingRule{match: rule.RuleMatch{Type: "GEOIP", Value: "CN"}, target: "DIRECT", position: 1},
			&mockRoutingRule{match: rule.RuleMatch{Type: "MATCH"}, target: "HKGroup", position: 999},
		},
	})

	// Must be valid YAML.
	m := mustParseYAML(t, out)

	// All sections must be present.
	for _, section := range []string{"proxy-providers", "rule-providers", "proxy-groups", "rules"} {
		if _, ok := m[section]; !ok {
			t.Errorf("expected section %q in full profile output, not found.\nYAML:\n%s", section, out)
		}
	}

	assertYAMLInt(t, m, "mixed-port", 7890)
}

// TestRender_NoResolve verifies the no-resolve flag is appended when NoResolve() returns true.
func TestRender_NoResolve(t *testing.T) {
	ruleWithResolve := &mockRoutingRule{
		match:     rule.RuleMatch{Type: "IP-CIDR", Value: "192.168.0.0/16"},
		target:    "LAN",
		noResolve: true,
	}
	out := mustRender(t, &profile.RenderInput{RoutingRules: []profile.RoutingRule{ruleWithResolve}})
	if !strings.Contains(out, "no-resolve") {
		t.Errorf("expected no-resolve for rule with NoResolve()=true: %s", out)
	}

	ruleWithoutResolve := &mockRoutingRule{
		match:  rule.RuleMatch{Type: "IP-CIDR", Value: "192.168.0.0/16"},
		target: "LAN",
	}
	out = mustRender(t, &profile.RenderInput{RoutingRules: []profile.RoutingRule{ruleWithoutResolve}})
	if strings.Contains(out, "no-resolve") {
		t.Errorf("mock rule should not have no-resolve: %s", out)
	}
}

// TestRender_UnescapeUnicode verifies that emoji/CJK names in YAML are not escaped.
func TestRender_UnescapeUnicode(t *testing.T) {
	// A group with a CJK/emoji name — go.yaml.in/yaml/v3 would escape it as \Uxxxxxxxx.
	grp := &mockGroup{
		name:     "\U0001F1ED\U0001F1F0 \u9999\u6e2f",
		strategy: "select",
	}
	out := mustRender(t, &profile.RenderInput{
		Groups: []profile.Group{grp},
	})

	// After unescaping, the actual runes should be present in the output.
	if strings.Contains(out, `\U`) {
		t.Errorf("expected unicode to be unescaped, but found \\U escape sequence:\n%s", out)
	}
	if !strings.Contains(out, "\u9999\u6e2f") {
		t.Errorf("expected CJK characters in output, got:\n%s", out)
	}
}

// ─── Assertion helpers ────────────────────────────────────────────────────────

func assertYAMLInt(t *testing.T, m map[string]any, key string, want int) {
	t.Helper()
	v, ok := m[key]
	if !ok {
		t.Errorf("YAML key %q not found", key)
		return
	}
	// go.yaml.in/yaml/v3 unmarshals integers as int.
	got, ok := v.(int)
	if !ok {
		t.Errorf("YAML key %q: expected int, got %T (%v)", key, v, v)
		return
	}
	if got != want {
		t.Errorf("YAML key %q: expected %d, got %d", key, want, got)
	}
}
