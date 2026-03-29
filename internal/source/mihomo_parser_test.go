package subscription

import (
	"testing"
)

// ─── Detect tests ────────────────────────────────────────────────────────────

func TestDetect_ValidMihomoYAML(t *testing.T) {
	data := []byte(`
proxies:
  - name: proxy1
    type: ss
    server: 1.2.3.4
    port: 443
`)
	p := NewMihomoParser()
	if !p.Detect(data) {
		t.Fatal("expected Detect to return true for YAML with proxies key")
	}
}

func TestDetect_ProviderOnlyYAML(t *testing.T) {
	p := NewMihomoParser()

	proxyProviderYAML := []byte(`
proxy-providers:
  myprovider:
    url: https://example.com/provider.yaml
    type: http
`)
	if !p.Detect(proxyProviderYAML) {
		t.Fatal("expected Detect to return true for YAML with proxy-providers key")
	}

	ruleProviderYAML := []byte(`
rule-providers:
  myrules:
    url: https://example.com/rules.yaml
    type: http
`)
	if !p.Detect(ruleProviderYAML) {
		t.Fatal("expected Detect to return true for YAML with rule-providers key")
	}
}

func TestDetect_InvalidYAML(t *testing.T) {
	p := NewMihomoParser()

	cases := []struct {
		name string
		data []byte
	}{
		{"random text", []byte("this is just some random text with no yaml structure")},
		{"json without proxy keys", []byte(`{"foo":"bar","baz":123}`)},
		{"yaml without proxy keys", []byte("foo: bar\nbaz: 123\n")},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if p.Detect(tc.data) {
				t.Fatalf("expected Detect to return false for %q", tc.name)
			}
		})
	}
}

func TestDetect_EmptyInput(t *testing.T) {
	p := NewMihomoParser()

	if p.Detect(nil) {
		t.Fatal("expected Detect to return false for nil input")
	}
	if p.Detect([]byte{}) {
		t.Fatal("expected Detect to return false for empty bytes")
	}
}

// ─── Parse tests ─────────────────────────────────────────────────────────────

func TestParse_ProxiesAndRules(t *testing.T) {
	data := []byte(`
proxies:
  - name: proxy-ss
    type: ss
    server: 1.2.3.4
    port: 8388
    cipher: aes-256-gcm
    password: secret
  - name: proxy-vmess
    type: vmess
    server: 5.6.7.8
    port: 443
    uuid: 00000000-0000-0000-0000-000000000001
rules:
  - DOMAIN,example.com,PROXY
  - IP-CIDR,192.168.0.0/16,DIRECT
  - MATCH,DIRECT
`)
	p := NewMihomoParser()
	result, err := p.Parse(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// node count
	if len(result.Nodes) != 2 {
		t.Fatalf("expected 2 nodes, got %d", len(result.Nodes))
	}

	// node names
	if result.Nodes[0].Name() != "proxy-ss" {
		t.Errorf("expected node[0].Name()=%q, got %q", "proxy-ss", result.Nodes[0].Name())
	}
	if result.Nodes[1].Name() != "proxy-vmess" {
		t.Errorf("expected node[1].Name()=%q, got %q", "proxy-vmess", result.Nodes[1].Name())
	}

	// RawConfig contains original proxy fields
	raw0 := result.Nodes[0].RawConfig()
	if raw0["type"] != "ss" {
		t.Errorf("expected node[0] type=ss, got %v", raw0["type"])
	}
	if raw0["cipher"] != "aes-256-gcm" {
		t.Errorf("expected node[0] cipher=aes-256-gcm, got %v", raw0["cipher"])
	}

	// rule count
	if len(result.Rules) != 3 {
		t.Fatalf("expected 3 rules, got %d", len(result.Rules))
	}

	// DOMAIN rule
	r0 := result.Rules[0]
	if r0.Match().Type != "DOMAIN" {
		t.Errorf("rule[0].Type: want DOMAIN, got %q", r0.Match().Type)
	}
	if r0.Match().Value != "example.com" {
		t.Errorf("rule[0].Value: want example.com, got %q", r0.Match().Value)
	}
	if r0.Target() != "PROXY" {
		t.Errorf("rule[0].Target: want PROXY, got %q", r0.Target())
	}

	// IP-CIDR rule
	r1 := result.Rules[1]
	if r1.Match().Type != "IP-CIDR" {
		t.Errorf("rule[1].Type: want IP-CIDR, got %q", r1.Match().Type)
	}
	if r1.Match().Value != "192.168.0.0/16" {
		t.Errorf("rule[1].Value: want 192.168.0.0/16, got %q", r1.Match().Value)
	}
	if r1.Target() != "DIRECT" {
		t.Errorf("rule[1].Target: want DIRECT, got %q", r1.Target())
	}

	// MATCH rule
	r2 := result.Rules[2]
	if r2.Match().Type != "MATCH" {
		t.Errorf("rule[2].Type: want MATCH, got %q", r2.Match().Type)
	}
	if r2.Match().Value != "" {
		t.Errorf("rule[2].Value: want empty string, got %q", r2.Match().Value)
	}
	if r2.Target() != "DIRECT" {
		t.Errorf("rule[2].Target: want DIRECT, got %q", r2.Target())
	}
}

func TestParse_MissingNameSkipped(t *testing.T) {
	data := []byte(`
proxies:
  - name: valid-proxy
    type: ss
    server: 1.2.3.4
    port: 443
  - type: vmess
    server: 5.6.7.8
    port: 8080
`)
	p := NewMihomoParser()
	result, err := p.Parse(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Nodes) != 1 {
		t.Fatalf("expected 1 node (unnamed skipped), got %d", len(result.Nodes))
	}
	if result.Nodes[0].Name() != "valid-proxy" {
		t.Errorf("expected node name=valid-proxy, got %q", result.Nodes[0].Name())
	}
}

func TestParse_MatchRule(t *testing.T) {
	data := []byte(`
rules:
  - MATCH,DIRECT
`)
	p := NewMihomoParser()
	result, err := p.Parse(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(result.Rules))
	}
	r := result.Rules[0]
	if r.Match().Type != "MATCH" {
		t.Errorf("Type: want MATCH, got %q", r.Match().Type)
	}
	if r.Match().Value != "" {
		t.Errorf("Value: want empty string, got %q", r.Match().Value)
	}
	if r.Target() != "DIRECT" {
		t.Errorf("Target: want DIRECT, got %q", r.Target())
	}
}

func TestParse_ProxyProviders(t *testing.T) {
	data := []byte(`
proxy-providers:
  provider-a:
    type: http
    url: https://example.com/a.yaml
    interval: 3600
    health-check:
      enable: true
  provider-b:
    type: http
    url: https://example.com/b.yaml
`)
	p := NewMihomoParser()
	result, err := p.Parse(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.NodeProviders) != 2 {
		t.Fatalf("expected 2 NodeProviders, got %d", len(result.NodeProviders))
	}

	// Build a lookup map so we don't depend on iteration order.
	byName := make(map[string]int)
	for i, pd := range result.NodeProviders {
		byName[pd.Name] = i
	}

	for _, wantName := range []string{"provider-a", "provider-b"} {
		idx, ok := byName[wantName]
		if !ok {
			t.Errorf("NodeProvider %q not found", wantName)
			continue
		}
		pd := result.NodeProviders[idx]
		expectedURL := "https://example.com/" + string(wantName[len("provider-"):]) + ".yaml"
		if pd.URL != expectedURL {
			t.Errorf("NodeProvider[%s].URL: want %q, got %q", wantName, expectedURL, pd.URL)
		}
		if pd.Metadata == nil {
			t.Errorf("NodeProvider[%s].Metadata should not be nil", wantName)
		}
	}
}

func TestParse_RuleProviders(t *testing.T) {
	data := []byte(`
rule-providers:
  reject-list:
    type: http
    behavior: domain
    url: https://example.com/reject.yaml
    interval: 86400
  proxy-list:
    type: http
    behavior: classical
    url: https://example.com/proxy.yaml
`)
	p := NewMihomoParser()
	result, err := p.Parse(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.RuleProviders) != 2 {
		t.Fatalf("expected 2 RuleProviders, got %d", len(result.RuleProviders))
	}

	byName := make(map[string]int)
	for i, pd := range result.RuleProviders {
		byName[pd.Name] = i
	}

	if idx, ok := byName["reject-list"]; ok {
		pd := result.RuleProviders[idx]
		if pd.URL != "https://example.com/reject.yaml" {
			t.Errorf("reject-list URL: want %q, got %q", "https://example.com/reject.yaml", pd.URL)
		}
		if pd.Metadata["behavior"] != "domain" {
			t.Errorf("reject-list Metadata[behavior]: want domain, got %v", pd.Metadata["behavior"])
		}
	} else {
		t.Error("RuleProvider 'reject-list' not found")
	}

	if idx, ok := byName["proxy-list"]; ok {
		pd := result.RuleProviders[idx]
		if pd.URL != "https://example.com/proxy.yaml" {
			t.Errorf("proxy-list URL: want %q, got %q", "https://example.com/proxy.yaml", pd.URL)
		}
		if pd.Metadata["behavior"] != "classical" {
			t.Errorf("proxy-list Metadata[behavior]: want classical, got %v", pd.Metadata["behavior"])
		}
	} else {
		t.Error("RuleProvider 'proxy-list' not found")
	}
}

func TestParse_EmptyYAML(t *testing.T) {
	p := NewMihomoParser()

	result, err := p.Parse([]byte{})
	if err != nil {
		t.Fatalf("unexpected error for empty input: %v", err)
	}
	if len(result.Nodes) != 0 {
		t.Errorf("expected 0 nodes, got %d", len(result.Nodes))
	}
	if len(result.Rules) != 0 {
		t.Errorf("expected 0 rules, got %d", len(result.Rules))
	}
	if len(result.NodeProviders) != 0 {
		t.Errorf("expected 0 NodeProviders, got %d", len(result.NodeProviders))
	}
	if len(result.RuleProviders) != 0 {
		t.Errorf("expected 0 RuleProviders, got %d", len(result.RuleProviders))
	}
}

func TestParse_InvalidYAML(t *testing.T) {
	p := NewMihomoParser()

	invalidData := []byte("{\x80\x81\x82invalid utf-8 and definitely not valid yaml: [[[")
	_, err := p.Parse(invalidData)
	if err == nil {
		t.Fatal("expected error for invalid YAML, got nil")
	}
}
