package engine

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"

	"github.com/Cluas/subhub/internal/model"
)

// TestFormatProxyOutput_Clash verifies clash format returns valid proxy-provider YAML.
func TestFormatProxyOutput_Clash(t *testing.T) {
	proxies := []map[string]interface{}{
		{
			"name":   "sg-01",
			"type":   "ss",
			"server": "sg.example.com",
			"port":   8388,
			"cipher": "aes-128-gcm",
			"password": "secret",
		},
	}

	body, ct, err := FormatProxyOutput("clash", proxies)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(ct, "text/yaml") {
		t.Errorf("expected text/yaml content-type, got %q", ct)
	}
	out := string(body)
	if !strings.Contains(out, "proxies:") {
		t.Errorf("expected 'proxies:' in output, got:\n%s", out)
	}
	if !strings.Contains(out, "sg-01") {
		t.Errorf("expected proxy name 'sg-01' in output, got:\n%s", out)
	}
}

// TestFormatProxyOutput_Surge verifies surge format returns correct [Proxy] block.
func TestFormatProxyOutput_Surge(t *testing.T) {
	proxies := []map[string]interface{}{
		{"name": "ss-node", "type": "ss", "server": "s1.example.com", "port": 8388, "cipher": "aes-128-gcm", "password": "pw1"},
		{"name": "vmess-node", "type": "vmess", "server": "v1.example.com", "port": 443, "uuid": "uuid-1234"},
		{"name": "trojan-node", "type": "trojan", "server": "t1.example.com", "port": 443, "password": "pw2"},
		{"name": "socks5-node", "type": "socks5", "server": "s5.example.com", "port": 1080},
		{"name": "vless-node", "type": "vless", "server": "vl.example.com", "port": 443, "uuid": "uuid-5678"},
	}

	body, ct, err := FormatProxyOutput("surge", proxies)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(ct, "text/plain") {
		t.Errorf("expected text/plain content-type, got %q", ct)
	}
	out := string(body)
	if !strings.Contains(out, "[Proxy]") {
		t.Errorf("expected '[Proxy]' header in surge output, got:\n%s", out)
	}
	if !strings.Contains(out, "= ss,") {
		t.Errorf("expected '= ss,' in surge output, got:\n%s", out)
	}
	if !strings.Contains(out, "= vmess,") {
		t.Errorf("expected '= vmess,' in surge output, got:\n%s", out)
	}
	if !strings.Contains(out, "= trojan,") {
		t.Errorf("expected '= trojan,' in surge output, got:\n%s", out)
	}
	if !strings.Contains(out, "= socks5,") {
		t.Errorf("expected '= socks5,' in surge output, got:\n%s", out)
	}
	if !strings.Contains(out, "# unsupported: vless-node") {
		t.Errorf("expected '# unsupported: vless-node' in surge output, got:\n%s", out)
	}
}

// TestFormatProxyOutput_Shadowrocket verifies shadowrocket format returns valid Base64.
func TestFormatProxyOutput_Shadowrocket(t *testing.T) {
	proxies := []map[string]interface{}{
		{
			"name":   "vmess-us",
			"type":   "vmess",
			"server": "us.example.com",
			"port":   443,
			"uuid":   "00000000-0000-0000-0000-000000000001",
			"alterId": 0,
		},
	}

	body, ct, err := FormatProxyOutput("shadowrocket", proxies)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(ct, "text/plain") {
		t.Errorf("expected text/plain content-type, got %q", ct)
	}
	decoded, err := base64.StdEncoding.DecodeString(string(body))
	if err != nil {
		t.Fatalf("shadowrocket output is not valid Base64: %v", err)
	}
	if len(decoded) == 0 {
		t.Error("expected non-empty decoded shadowrocket content")
	}
}

// TestFormatRuleOutput_Clash verifies clash rule format returns payload YAML.
func TestFormatRuleOutput_Clash(t *testing.T) {
	rules := []*model.Rule{
		{Type: "DOMAIN-SUFFIX", Payload: "google.com", Target: "PROXY"},
		{Type: "DOMAIN-KEYWORD", Payload: "youtube", Target: "PROXY"},
		{Type: "MATCH", Payload: "", Target: "DIRECT"}, // should be skipped
	}

	body, ct, err := FormatRuleOutput("clash", rules)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(ct, "text/yaml") {
		t.Errorf("expected text/yaml content-type, got %q", ct)
	}
	out := string(body)
	if !strings.Contains(out, "payload:") {
		t.Errorf("expected 'payload:' in rule output, got:\n%s", out)
	}
	if !strings.Contains(out, "DOMAIN-SUFFIX,google.com") {
		t.Errorf("expected 'DOMAIN-SUFFIX,google.com' in payload, got:\n%s", out)
	}
	if strings.Contains(out, "MATCH") {
		t.Errorf("expected MATCH rule to be excluded, but found in output:\n%s", out)
	}
}

// TestFormatProxyOutput_UnknownFormat verifies unsupported format returns error.
func TestFormatProxyOutput_UnknownFormat(t *testing.T) {
	proxies := []map[string]interface{}{
		{"name": "test", "type": "ss", "server": "s.example.com", "port": 8388},
	}
	body, _, err := FormatProxyOutput("pdf", proxies)
	if err == nil {
		t.Error("expected error for unknown format, got nil")
	}
	if body != nil {
		t.Errorf("expected nil body for unknown format, got %v", body)
	}
}

// sharedProxyFixtures returns a common set of proxies used across format tests.
func sharedProxyFixtures() []map[string]interface{} {
	return []map[string]interface{}{
		{"name": "ss-node", "type": "ss", "server": "s1.example.com", "port": 8388, "cipher": "aes-128-gcm", "password": "pw1"},
		{"name": "vmess-node", "type": "vmess", "server": "v1.example.com", "port": 443, "uuid": "uuid-1234"},
		{"name": "trojan-node", "type": "trojan", "server": "t1.example.com", "port": 443, "password": "pw2"},
		{"name": "vless-node", "type": "vless", "server": "vl.example.com", "port": 443, "uuid": "uuid-5678"},
	}
}

// sharedRuleFixtures returns a common set of rules used across format tests.
func sharedRuleFixtures() []*model.Rule {
	return []*model.Rule{
		{Type: "DOMAIN-SUFFIX", Payload: "google.com", Target: "PROXY"},
		{Type: "DOMAIN-KEYWORD", Payload: "youtube", Target: "PROXY"},
		{Type: "IP-CIDR", Payload: "8.8.8.8/32", Target: "PROXY"},
		{Type: "MATCH", Payload: "", Target: "DIRECT"},
	}
}

// TestFormatProxyOutput_QuantumultX verifies QuantumultX proxy format output.
func TestFormatProxyOutput_QuantumultX(t *testing.T) {
	proxies := sharedProxyFixtures()
	body, ct, err := FormatProxyOutput("quantumultx", proxies)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(ct, "text/plain") {
		t.Errorf("expected text/plain content-type, got %q", ct)
	}
	out := string(body)
	if !strings.Contains(out, "[Remote Proxy]") {
		t.Errorf("expected '[Remote Proxy]' header, got:\n%s", out)
	}
	if !strings.Contains(out, "shadowsocks=") {
		t.Errorf("expected 'shadowsocks=' line, got:\n%s", out)
	}
	if !strings.Contains(out, "vmess=") {
		t.Errorf("expected 'vmess=' line, got:\n%s", out)
	}
	if !strings.Contains(out, "trojan=") {
		t.Errorf("expected 'trojan=' line, got:\n%s", out)
	}
	if !strings.Contains(out, "# unsupported: vless-node") {
		t.Errorf("expected '# unsupported: vless-node', got:\n%s", out)
	}
}

// TestFormatProxyOutput_Loon verifies Loon proxy format output.
func TestFormatProxyOutput_Loon(t *testing.T) {
	proxies := sharedProxyFixtures()
	body, ct, err := FormatProxyOutput("loon", proxies)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(ct, "text/plain") {
		t.Errorf("expected text/plain content-type, got %q", ct)
	}
	out := string(body)
	if !strings.Contains(out, "[Remote Proxy]") {
		t.Errorf("expected '[Remote Proxy]' header, got:\n%s", out)
	}
	if !strings.Contains(out, "= Shadowsocks,") {
		t.Errorf("expected '= Shadowsocks,' line, got:\n%s", out)
	}
	if !strings.Contains(out, "= VMESS,") {
		t.Errorf("expected '= VMESS,' line, got:\n%s", out)
	}
	if !strings.Contains(out, "= Trojan,") {
		t.Errorf("expected '= Trojan,' line, got:\n%s", out)
	}
	if !strings.Contains(out, "# unsupported: vless-node") {
		t.Errorf("expected '# unsupported: vless-node', got:\n%s", out)
	}
}

// TestFormatProxyOutput_Singbox verifies singbox proxy format returns valid JSON with outbounds.
func TestFormatProxyOutput_Singbox(t *testing.T) {
	proxies := sharedProxyFixtures()
	body, ct, err := FormatProxyOutput("singbox", proxies)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(ct, "application/json") {
		t.Errorf("expected application/json content-type, got %q", ct)
	}
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("failed to unmarshal singbox JSON: %v", err)
	}
	outboundsRaw, ok := result["outbounds"]
	if !ok {
		t.Fatal("expected 'outbounds' key in singbox output")
	}
	outbounds, ok := outboundsRaw.([]interface{})
	if !ok {
		t.Fatalf("expected outbounds to be a slice, got %T", outboundsRaw)
	}
	// Find the ss entry and verify its fields.
	var ssEntry map[string]interface{}
	for _, ob := range outbounds {
		m, _ := ob.(map[string]interface{})
		if m["type"] == "shadowsocks" {
			ssEntry = m
			break
		}
	}
	if ssEntry == nil {
		t.Fatal("expected a shadowsocks outbound entry")
	}
	portVal, ok := ssEntry["server_port"]
	if !ok {
		t.Fatal("expected 'server_port' in shadowsocks outbound")
	}
	// JSON numbers unmarshal as float64.
	if _, ok := portVal.(float64); !ok {
		t.Errorf("expected server_port to be a number, got %T", portVal)
	}
}

// TestFormatRuleOutput_QuantumultX verifies QuantumultX rule format: plain text, no header.
func TestFormatRuleOutput_QuantumultX(t *testing.T) {
	rules := sharedRuleFixtures()
	body, ct, err := FormatRuleOutput("quantumultx", rules)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(ct, "text/plain") {
		t.Errorf("expected text/plain content-type, got %q", ct)
	}
	out := string(body)
	if strings.Contains(out, "[") {
		t.Errorf("expected no section header in QuantumultX rule output, got:\n%s", out)
	}
	if !strings.Contains(out, "DOMAIN-SUFFIX,google.com") {
		t.Errorf("expected 'DOMAIN-SUFFIX,google.com', got:\n%s", out)
	}
	if strings.Contains(out, "MATCH") {
		t.Errorf("expected MATCH to be excluded, got:\n%s", out)
	}
}

// TestFormatRuleOutput_Loon verifies Loon rule format: [Remote Filter] header + spaced lines.
func TestFormatRuleOutput_Loon(t *testing.T) {
	rules := sharedRuleFixtures()
	body, ct, err := FormatRuleOutput("loon", rules)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(ct, "text/plain") {
		t.Errorf("expected text/plain content-type, got %q", ct)
	}
	out := string(body)
	if !strings.Contains(out, "[Remote Filter]") {
		t.Errorf("expected '[Remote Filter]' header, got:\n%s", out)
	}
	if !strings.Contains(out, "DOMAIN-SUFFIX, google.com") {
		t.Errorf("expected 'DOMAIN-SUFFIX, google.com', got:\n%s", out)
	}
}

// TestFormatRuleOutput_Singbox verifies singbox rule format: JSON with version=2 and rules array.
func TestFormatRuleOutput_Singbox(t *testing.T) {
	rules := sharedRuleFixtures()
	body, ct, err := FormatRuleOutput("singbox", rules)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(ct, "application/json") {
		t.Errorf("expected application/json content-type, got %q", ct)
	}
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("failed to unmarshal singbox rule JSON: %v", err)
	}
	version, ok := result["version"]
	if !ok {
		t.Fatal("expected 'version' key in singbox rule output")
	}
	if version.(float64) != 2 {
		t.Errorf("expected version=2, got %v", version)
	}
	rulesRaw, ok := result["rules"]
	if !ok {
		t.Fatal("expected 'rules' key in singbox rule output")
	}
	rulesSlice, ok := rulesRaw.([]interface{})
	if !ok || len(rulesSlice) == 0 {
		t.Fatal("expected non-empty rules array")
	}
	firstRule, ok := rulesSlice[0].(map[string]interface{})
	if !ok {
		t.Fatal("expected rules[0] to be an object")
	}
	domainSuffixRaw, ok := firstRule["domain_suffix"]
	if !ok {
		t.Fatal("expected 'domain_suffix' in rules[0]")
	}
	domainSuffix, ok := domainSuffixRaw.([]interface{})
	if !ok {
		t.Fatalf("expected domain_suffix to be an array, got %T", domainSuffixRaw)
	}
	found := false
	for _, d := range domainSuffix {
		if d.(string) == "google.com" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected 'google.com' in domain_suffix, got %v", domainSuffix)
	}
}
