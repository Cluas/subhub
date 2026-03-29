package engine

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Cluas/subhub/internal/model"
	"github.com/Cluas/subhub/internal/store"
)

// minimalClashYAML contains 2 proxies and 3 rules in valid Clash YAML format.
const minimalClashYAML = `
proxies:
  - name: "proxy-hk"
    type: ss
    server: hk.example.com
    port: 8388
    cipher: aes-256-gcm
    password: test123
  - name: "proxy-jp"
    type: vmess
    server: jp.example.com
    port: 443
    uuid: abcd-1234

rules:
  - DOMAIN,example.com,DIRECT
  - DOMAIN-SUFFIX,google.com,Proxy
  - MATCH,DIRECT
`

// missingNameYAML has one proxy with no name field.
const missingNameYAML = `
proxies:
  - type: ss
    server: host.example.com
    port: 1234
    cipher: aes-256-gcm
    password: secret

rules:
  - MATCH,DIRECT
`

func newTestStore(t *testing.T) store.Store {
	t.Helper()
	st, err := store.NewSQLite(":memory:")
	if err != nil {
		t.Fatalf("open in-memory store: %v", err)
	}
	t.Cleanup(func() { st.Close() })
	return st
}

func createTestSubscription(t *testing.T, st store.Store, url string) *model.Subscription {
	t.Helper()
	sub := &model.Subscription{
		Name:   "test-sub",
		URL:    url,
		Type:   "clash",
		Status: "active",
	}
	if err := st.CreateSubscription(context.Background(), sub); err != nil {
		t.Fatalf("create subscription: %v", err)
	}
	return sub
}

// TestFetchAndPersist_HappyPath verifies that proxies and rules are stored and
// the subscription is updated on a successful fetch.
func TestFetchAndPersist(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/x-yaml")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(minimalClashYAML))
	}))
	defer srv.Close()

	st := newTestStore(t)
	sub := createTestSubscription(t, st, srv.URL)

	err := FetchAndPersist(context.Background(), st, sub)
	if err != nil {
		t.Fatalf("FetchAndPersist returned error: %v", err)
	}

	// Subscription metadata assertions
	if sub.NodeCount != 2 {
		t.Errorf("NodeCount = %d, want 2", sub.NodeCount)
	}
	if sub.Status != "active" {
		t.Errorf("Status = %q, want \"active\"", sub.Status)
	}
	if sub.LastFetchAt == nil {
		t.Error("LastFetchAt is nil, expected non-nil timestamp")
	} else if time.Since(*sub.LastFetchAt) > 5*time.Second {
		t.Errorf("LastFetchAt is too old: %v", sub.LastFetchAt)
	}

	// Verify proxies persisted in SQLite
	proxies, err := st.ListProxies(context.Background(), store.ProxyFilter{SubscriptionID: sub.ID})
	if err != nil {
		t.Fatalf("ListProxies error: %v", err)
	}
	if len(proxies) != 2 {
		t.Fatalf("ListProxies returned %d proxies, want 2", len(proxies))
	}

	// Build name→proxy map for easier assertion
	proxyByName := make(map[string]*model.Proxy)
	for _, p := range proxies {
		proxyByName[p.Name] = p
	}

	hk, ok := proxyByName["proxy-hk"]
	if !ok {
		t.Error("proxy-hk not found in stored proxies")
	} else {
		if hk.Type != "ss" {
			t.Errorf("proxy-hk type = %q, want \"ss\"", hk.Type)
		}
		if hk.Port != 8388 {
			t.Errorf("proxy-hk port = %d, want 8388", hk.Port)
		}
		if hk.Server != "hk.example.com" {
			t.Errorf("proxy-hk server = %q", hk.Server)
		}
	}

	jp, ok := proxyByName["proxy-jp"]
	if !ok {
		t.Error("proxy-jp not found in stored proxies")
	} else {
		if jp.Type != "vmess" {
			t.Errorf("proxy-jp type = %q, want \"vmess\"", jp.Type)
		}
		if jp.Port != 443 {
			t.Errorf("proxy-jp port = %d, want 443", jp.Port)
		}
	}

	// Verify rules persisted in SQLite
	rules, err := st.ListRules(context.Background(), store.RuleFilter{SubscriptionID: sub.ID})
	if err != nil {
		t.Fatalf("ListRules error: %v", err)
	}
	if len(rules) != 3 {
		t.Fatalf("ListRules returned %d rules, want 3", len(rules))
	}

	// Build type+payload→rule map for verification
	ruleByKey := make(map[string]*model.Rule)
	for _, r := range rules {
		ruleByKey[r.Type+":"+r.Payload] = r
	}

	// DOMAIN,example.com,DIRECT → Type=DOMAIN, Payload=example.com, Target=DIRECT
	r1, ok := ruleByKey["DOMAIN:example.com"]
	if !ok {
		t.Error("rule DOMAIN:example.com not found")
	} else if r1.Target != "DIRECT" {
		t.Errorf("DOMAIN rule target = %q, want \"DIRECT\"", r1.Target)
	}

	// DOMAIN-SUFFIX,google.com,Proxy → Type=DOMAIN-SUFFIX, Payload=google.com, Target=Proxy
	r2, ok := ruleByKey["DOMAIN-SUFFIX:google.com"]
	if !ok {
		t.Error("rule DOMAIN-SUFFIX:google.com not found")
	} else if r2.Target != "Proxy" {
		t.Errorf("DOMAIN-SUFFIX rule target = %q, want \"Proxy\"", r2.Target)
	}

	// MATCH,DIRECT → Type=MATCH, Payload="", Target=DIRECT
	r3, ok := ruleByKey["MATCH:"]
	if !ok {
		t.Error("MATCH rule not found")
	} else {
		if r3.Target != "DIRECT" {
			t.Errorf("MATCH rule target = %q, want \"DIRECT\"", r3.Target)
		}
		if r3.Payload != "" {
			t.Errorf("MATCH rule payload = %q, want \"\"", r3.Payload)
		}
	}
}

// TestFetchAndPersist_ServerError verifies that a 500 response sets Status="error"
// and ErrorMsg on the subscription.
func TestFetchAndPersist_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}))
	defer srv.Close()

	st := newTestStore(t)
	sub := createTestSubscription(t, st, srv.URL)

	err := FetchAndPersist(context.Background(), st, sub)
	if err == nil {
		t.Fatal("FetchAndPersist returned nil error, expected non-nil on 500")
	}
	if sub.Status != "error" {
		t.Errorf("Status = %q, want \"error\"", sub.Status)
	}
	if sub.ErrorMsg == "" {
		t.Error("ErrorMsg is empty, expected error text")
	}

	// Confirm the error status was persisted to the store
	persisted, err2 := st.GetSubscription(context.Background(), sub.ID)
	if err2 != nil {
		t.Fatalf("GetSubscription: %v", err2)
	}
	if persisted.Status != "error" {
		t.Errorf("persisted Status = %q, want \"error\"", persisted.Status)
	}
}

// TestFetchAndPersist_MissingProxyName verifies that proxies without a "name"
// key are skipped gracefully and do not cause a panic or error.
func TestFetchAndPersist_MissingProxyName(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/x-yaml")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(missingNameYAML))
	}))
	defer srv.Close()

	st := newTestStore(t)
	sub := createTestSubscription(t, st, srv.URL)

	err := FetchAndPersist(context.Background(), st, sub)
	if err != nil {
		t.Fatalf("FetchAndPersist error: %v", err)
	}

	// No named proxies → NodeCount should be 0
	if sub.NodeCount != 0 {
		t.Errorf("NodeCount = %d, want 0 (nameless proxy skipped)", sub.NodeCount)
	}

	proxies, _ := st.ListProxies(context.Background(), store.ProxyFilter{SubscriptionID: sub.ID})
	if len(proxies) != 0 {
		t.Errorf("stored %d proxies, want 0", len(proxies))
	}
}

// TestConvertRules_OnePart verifies that a single-segment rule string is skipped.
func TestConvertRules_OnePart(t *testing.T) {
	raw := []model.RuleWithSource{
		{Rule: "MATCH", Source: "main"},       // only 1 part → skip
		{Rule: "MATCH,DIRECT", Source: "main"}, // valid MATCH rule → keep
	}
	rules := convertRules(1, raw)
	if len(rules) != 1 {
		t.Fatalf("got %d rules, want 1 (single-part should be skipped)", len(rules))
	}
	if rules[0].Type != "MATCH" {
		t.Errorf("rule type = %q, want \"MATCH\"", rules[0].Type)
	}
	if rules[0].Payload != "" {
		t.Errorf("MATCH payload = %q, want \"\"", rules[0].Payload)
	}
	if rules[0].Target != "DIRECT" {
		t.Errorf("MATCH target = %q, want \"DIRECT\"", rules[0].Target)
	}
}

// TestConvertRules_CommaInPayload verifies rules where the payload itself
// contains a comma are handled correctly (middle parts joined).
func TestConvertRules_CommaInPayload(t *testing.T) {
	raw := []model.RuleWithSource{
		{Rule: "IP-CIDR,192.168.0.0/16,DIRECT", Source: "main"},
	}
	rules := convertRules(1, raw)
	if len(rules) != 1 {
		t.Fatalf("got %d rules, want 1", len(rules))
	}
	r := rules[0]
	if r.Type != "IP-CIDR" {
		t.Errorf("type = %q, want IP-CIDR", r.Type)
	}
	if r.Payload != "192.168.0.0/16" {
		t.Errorf("payload = %q, want 192.168.0.0/16", r.Payload)
	}
	if r.Target != "DIRECT" {
		t.Errorf("target = %q, want DIRECT", r.Target)
	}
}

// TestConvertProxies_FloatPort verifies that float64 port values (JSON numbers)
// are correctly cast to int.
func TestConvertProxies_FloatPort(t *testing.T) {
	raw := []map[string]interface{}{
		{"name": "test-proxy", "type": "ss", "server": "example.com", "port": float64(9999)},
	}
	proxies := convertProxies(1, raw)
	if len(proxies) != 1 {
		t.Fatalf("got %d proxies, want 1", len(proxies))
	}
	if proxies[0].Port != 9999 {
		t.Errorf("port = %d, want 9999", proxies[0].Port)
	}
}
