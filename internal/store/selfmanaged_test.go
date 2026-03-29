package store

import (
	"context"
	"strings"
	"testing"

	"github.com/Cluas/subhub/internal/model"
)

func TestSelfManagedProxyCRUD(t *testing.T) {
	st, err := NewSQLite(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	ctx := context.Background()
	impl := st.(*sqliteStore)
	_ = impl

	// CreateProxy with nil subscription_id
	p := &model.Proxy{
		SubscriptionID: nil,
		Name:           "self-node",
		Type:           "ss",
		Server:         "sg.example.com",
		Port:           8388,
		Region:         "SG",
	}
	if err := st.CreateProxy(ctx, p); err != nil {
		t.Fatal("CreateProxy:", err)
	}
	if p.ID == 0 {
		t.Fatal("expected non-zero ID")
	}

	// GetProxy verifies subscription_id is nil
	got, err := st.GetProxy(ctx, p.ID)
	if err != nil {
		t.Fatal("GetProxy:", err)
	}
	if got.SubscriptionID != nil {
		t.Fatalf("expected nil SubscriptionID, got %d", *got.SubscriptionID)
	}
	if got.Name != "self-node" {
		t.Fatalf("name: got %q want self-node", got.Name)
	}

	// UpdateProxy
	got.Name = "updated-node"
	if err := st.UpdateProxy(ctx, got); err != nil {
		t.Fatal("UpdateProxy:", err)
	}
	got2, _ := st.GetProxy(ctx, p.ID)
	if got2.Name != "updated-node" {
		t.Fatalf("after UpdateProxy: got %q want updated-node", got2.Name)
	}

	// DeleteProxy
	if err := st.DeleteProxy(ctx, p.ID); err != nil {
		t.Fatal("DeleteProxy:", err)
	}
}

func TestSelfManagedRuleCRUD(t *testing.T) {
	st, err := NewSQLite(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	ctx := context.Background()

	// CreateRule with nil subscription_id
	r := &model.Rule{
		SubscriptionID: nil,
		ProviderName:   "manual",
		Type:           "DOMAIN-SUFFIX",
		Payload:        "example.com",
		Target:         "DIRECT",
	}
	if err := st.CreateRule(ctx, r); err != nil {
		t.Fatal("CreateRule:", err)
	}
	if r.ID == 0 {
		t.Fatal("expected non-zero ID")
	}

	// ListRules — self-managed rule appears in all-rules query
	rules, err := st.ListRules(ctx, RuleFilter{})
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, ru := range rules {
		if ru.ID == r.ID {
			found = true
			if ru.SubscriptionID != nil {
				t.Fatalf("expected nil SubscriptionID, got %d", *ru.SubscriptionID)
			}
		}
	}
	if !found {
		t.Fatal("created rule not found in ListRules")
	}

	// UpdateRule
	r.Payload = "updated.com"
	if err := st.UpdateRule(ctx, r); err != nil {
		t.Fatal("UpdateRule:", err)
	}

	// DeleteRule
	if err := st.DeleteRule(ctx, r.ID); err != nil {
		t.Fatal("DeleteRule:", err)
	}
}

// TestListProxies_GroupsFilter verifies that ProxyFilter.Groups filters by name substring (OR).
func TestListProxies_GroupsFilter(t *testing.T) {
	st, err := NewSQLite(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	ctx := context.Background()

	// Seed a subscription and proxies.
	sub := &model.Subscription{Name: "test", URL: "http://example.com"}
	if err := st.CreateSubscription(ctx, sub); err != nil {
		t.Fatal("CreateSubscription:", err)
	}
	proxies := []*model.Proxy{
		{SubscriptionID: &sub.ID, Name: "HK-01", Type: "ss", Server: "hk1.example.com", Port: 443},
		{SubscriptionID: &sub.ID, Name: "HK-02", Type: "ss", Server: "hk2.example.com", Port: 443},
		{SubscriptionID: &sub.ID, Name: "SG-01", Type: "ss", Server: "sg1.example.com", Port: 443},
		{SubscriptionID: &sub.ID, Name: "US-01", Type: "ss", Server: "us1.example.com", Port: 443},
	}
	if err := st.UpsertProxies(ctx, sub.ID, proxies); err != nil {
		t.Fatal("UpsertProxies:", err)
	}

	// Single group filter: only HK proxies.
	result, err := st.ListProxies(ctx, ProxyFilter{Groups: []string{"HK"}})
	if err != nil {
		t.Fatal("ListProxies single group:", err)
	}
	if len(result) != 2 {
		t.Errorf("expected 2 HK proxies, got %d", len(result))
	}
	for _, p := range result {
		if !strings.Contains(p.Name, "HK") {
			t.Errorf("unexpected proxy %q in HK group filter", p.Name)
		}
	}

	// Multi-group filter: HK OR SG (OR semantics).
	result, err = st.ListProxies(ctx, ProxyFilter{Groups: []string{"HK", "SG"}})
	if err != nil {
		t.Fatal("ListProxies multi group:", err)
	}
	if len(result) != 3 {
		t.Errorf("expected 3 proxies (HK+SG), got %d", len(result))
	}
	names := make(map[string]bool)
	for _, p := range result {
		names[p.Name] = true
	}
	if names["US-01"] {
		t.Error("US-01 should not be included in groups=[HK,SG] filter")
	}

	// Empty groups filter: returns all proxies.
	result, err = st.ListProxies(ctx, ProxyFilter{})
	if err != nil {
		t.Fatal("ListProxies empty filter:", err)
	}
	if len(result) != 4 {
		t.Errorf("expected 4 proxies with no filter, got %d", len(result))
	}
}
