package api

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/Cluas/subhub/internal/store"
)

// DashboardHandler serves aggregate statistics for the dashboard.
type DashboardHandler struct {
	Store store.Store
}

// dashboardStats is the JSON response shape for GET /api/dashboard/stats.
type dashboardStats struct {
	SubscriptionCount       int `json:"subscription_count"`
	ActiveSubscriptionCount int `json:"active_subscription_count"`
	NodeCount               int `json:"node_count"`
	AliveNodeCount          int `json:"alive_node_count"`
	EndpointCount           int `json:"endpoint_count"`
}

// Stats handles GET /api/dashboard/stats.
// It queries the store for subscriptions and proxies and returns aggregate counts.
// On store failure it returns 500 with a JSON error body.
func (h *DashboardHandler) Stats(w http.ResponseWriter, r *http.Request) {
	subs, err := h.Store.ListSubscriptions(r.Context())
	if err != nil {
		slog.Error("dashboard: failed to list subscriptions", "err", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "failed to query subscriptions"})
		return
	}

	proxies, err := h.Store.ListProxies(r.Context(), store.ProxyFilter{})
	if err != nil {
		slog.Error("dashboard: failed to list proxies", "err", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "failed to query proxies"})
		return
	}

	aliveTrue := true
	aliveProxies, err := h.Store.ListProxies(r.Context(), store.ProxyFilter{Alive: &aliveTrue})
	if err != nil {
		slog.Error("dashboard: failed to list alive proxies", "err", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "failed to query alive proxies"})
		return
	}

	endpoints, err := h.Store.ListEndpoints(r.Context())
	if err != nil {
		slog.Error("dashboard: failed to list endpoints", "err", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "failed to query endpoints"})
		return
	}

	activeSubs := 0
	for _, s := range subs {
		if s.Status == "active" {
			activeSubs++
		}
	}

	resp := dashboardStats{
		SubscriptionCount:       len(subs),
		ActiveSubscriptionCount: activeSubs,
		NodeCount:               len(proxies),
		AliveNodeCount:          len(aliveProxies),
		EndpointCount:           len(endpoints),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
