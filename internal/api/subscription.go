package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/Cluas/subhub/internal/engine"
	"github.com/Cluas/subhub/internal/health"
	"github.com/Cluas/subhub/internal/model"
	"github.com/Cluas/subhub/internal/scheduler"
	"github.com/Cluas/subhub/internal/store"
)

// SubscriptionHandler handles REST operations on subscriptions.
type SubscriptionHandler struct {
	Store     store.Store
	Scheduler *scheduler.Scheduler
}

// subscriptionRequest is the shape of the JSON body accepted by Create and Update.
type subscriptionRequest struct {
	Name        string `json:"name"`
	URL         string `json:"url"`
	Type        string `json:"type"`
	AutoRefresh bool   `json:"auto_refresh"`
	RefreshCron string `json:"refresh_cron"`
}

// writeJSON encodes v as JSON with the given status code.
func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Error("writeJSON encode error", "err", err)
	}
}

// writeError writes a {"error": msg} JSON response.
func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// parseID extracts the chi URL param "id" and parses it as int64.
func parseID(r *http.Request) (int64, error) {
	raw := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return 0, errors.New("invalid id")
	}
	return id, nil
}

// derefInt64 dereferences a *int64 safely, returning 0 for nil.
func derefInt64(v *int64) int64 {
	if v == nil {
		return 0
	}
	return *v
}

// syncScheduler re-loads all subscriptions and calls Sync on the scheduler if
// one is configured. Errors are logged but do not affect the HTTP response.
func (h *SubscriptionHandler) syncScheduler(ctx context.Context) {
	if h.Scheduler == nil {
		return
	}
	subs, err := h.Store.ListSubscriptions(ctx)
	if err != nil {
		slog.Error("syncScheduler: list subscriptions failed", "err", err)
		return
	}
	h.Scheduler.Sync(ctx, subs)
}

// List handles GET /api/subscriptions — returns all subscriptions.
// Always returns a JSON array (never null).
func (h *SubscriptionHandler) List(w http.ResponseWriter, r *http.Request) {
	subs, err := h.Store.ListSubscriptions(r.Context())
	if err != nil {
		slog.Error("list subscriptions", "err", err)
		writeError(w, http.StatusInternalServerError, "failed to list subscriptions")
		return
	}
	// Ensure we send [] not null when the slice is empty.
	if subs == nil {
		subs = []*model.Subscription{}
	}
	writeJSON(w, http.StatusOK, subs)
}

// Create handles POST /api/subscriptions — creates a new subscription.
func (h *SubscriptionHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req subscriptionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	if req.URL == "" {
		writeError(w, http.StatusBadRequest, "url is required")
		return
	}
	if req.Type == "" {
		req.Type = "clash"
	}

	sub := &model.Subscription{
		Name:        req.Name,
		URL:         req.URL,
		Type:        req.Type,
		AutoRefresh: req.AutoRefresh,
		RefreshCron: req.RefreshCron,
		Status:      "active",
	}
	if err := h.Store.CreateSubscription(r.Context(), sub); err != nil {
		slog.Error("create subscription", "err", err)
		writeError(w, http.StatusInternalServerError, "failed to create subscription")
		return
	}
	h.syncScheduler(r.Context())
	writeJSON(w, http.StatusCreated, sub)
}

// Get handles GET /api/subscriptions/{id} — returns a single subscription.
func (h *SubscriptionHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	sub, err := h.Store.GetSubscription(r.Context(), id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "subscription not found")
			return
		}
		slog.Error("get subscription", "id", id, "err", err)
		writeError(w, http.StatusInternalServerError, "failed to get subscription")
		return
	}
	if sub == nil {
		writeError(w, http.StatusNotFound, "subscription not found")
		return
	}
	writeJSON(w, http.StatusOK, sub)
}

// Update handles PUT /api/subscriptions/{id} — merges fields and saves.
func (h *SubscriptionHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	sub, err := h.Store.GetSubscription(r.Context(), id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "subscription not found")
			return
		}
		slog.Error("update subscription get", "id", id, "err", err)
		writeError(w, http.StatusInternalServerError, "failed to get subscription")
		return
	}
	if sub == nil {
		writeError(w, http.StatusNotFound, "subscription not found")
		return
	}

	var req subscriptionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	// Merge only non-zero/non-empty values from the request.
	if req.Name != "" {
		sub.Name = req.Name
	}
	if req.URL != "" {
		sub.URL = req.URL
	}
	if req.Type != "" {
		sub.Type = req.Type
	}
	if req.AutoRefresh {
		sub.AutoRefresh = req.AutoRefresh
	}
	if req.RefreshCron != "" {
		sub.RefreshCron = req.RefreshCron
	}

	if err := h.Store.UpdateSubscription(r.Context(), sub); err != nil {
		slog.Error("update subscription", "id", id, "err", err)
		writeError(w, http.StatusInternalServerError, "failed to update subscription")
		return
	}
	h.syncScheduler(r.Context())
	writeJSON(w, http.StatusOK, sub)
}

// Delete handles DELETE /api/subscriptions/{id} — returns 204 with no body.
func (h *SubscriptionHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.Store.DeleteSubscription(r.Context(), id); err != nil {
		slog.Error("delete subscription", "id", id, "err", err)
		writeError(w, http.StatusInternalServerError, "failed to delete subscription")
		return
	}
	h.syncScheduler(r.Context())
	w.WriteHeader(http.StatusNoContent)
}

// Fetch handles POST /api/subscriptions/{id}/fetch — pulls remote Clash config
// and persists proxies and rules to the store.
func (h *SubscriptionHandler) Fetch(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	sub, err := h.Store.GetSubscription(r.Context(), id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "subscription not found")
			return
		}
		slog.Error("fetch subscription get", "id", id, "err", err)
		writeError(w, http.StatusInternalServerError, "failed to get subscription")
		return
	}
	if sub == nil {
		writeError(w, http.StatusNotFound, "subscription not found")
		return
	}

	if err := engine.FetchAndPersist(r.Context(), h.Store, sub); err != nil {
		slog.Error("fetch and persist", "id", id, "err", err)
		writeError(w, http.StatusInternalServerError, "fetch failed: "+err.Error())
		return
	}

	// Re-read the subscription so the response reflects the updated node count, status, etc.
	updated, err := h.Store.GetSubscription(r.Context(), id)
	if err != nil {
		slog.Error("fetch re-read subscription", "id", id, "err", err)
		writeError(w, http.StatusInternalServerError, "fetch succeeded but failed to re-read subscription")
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

// healthCheckResult is one entry in the health-check response results array.
type healthCheckResult struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	Server    string `json:"server"`
	Port      int    `json:"port"`
	Alive     bool   `json:"alive"`
	LatencyMs int    `json:"latency_ms"`
}

// HealthCheck handles POST /api/subscriptions/{id}/health-check — tests all
// proxy nodes for the subscription and persists the results.
func (h *SubscriptionHandler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Load subscription — 404 if not found.
	sub, err := h.Store.GetSubscription(r.Context(), id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "subscription not found")
			return
		}
		slog.Error("health check get subscription", "id", id, "err", err)
		writeError(w, http.StatusInternalServerError, "failed to get subscription")
		return
	}
	if sub == nil {
		writeError(w, http.StatusNotFound, "subscription not found")
		return
	}

	// List all proxies for this subscription.
	proxies, err := h.Store.ListProxies(r.Context(), store.ProxyFilter{SubscriptionID: id})
	if err != nil {
		slog.Error("health check list proxies", "id", id, "err", err)
		writeError(w, http.StatusInternalServerError, "failed to list proxies")
		return
	}

	// Map model.Proxy → health.ProxyNode.
	nodes := make([]health.ProxyNode, 0, len(proxies))
	for _, p := range proxies {
		udp, _ := p.Config["udp"].(bool)
		nodes = append(nodes, health.ProxyNode{
			ID:     p.ID,
			Name:   p.Name,
			Server: p.Server,
			Port:   strconv.Itoa(p.Port),
			Type:   p.Type,
			UDP:    udp,
		})
	}

	// Run concurrent health checks; BatchCheck returns only alive results.
	aliveResults := health.BatchCheck(nodes)

	// Index alive results by proxy ID for O(1) lookup.
	aliveMap := make(map[int64]health.Result, len(aliveResults))
	for _, res := range aliveResults {
		aliveMap[res.ID] = res
	}

	// Persist health status for ALL proxies and build response array.
	results := make([]healthCheckResult, 0, len(proxies))
	for _, p := range proxies {
		if res, ok := aliveMap[p.ID]; ok {
			latencyMs := int(res.Latency.Milliseconds())
			if err := h.Store.UpdateProxyHealth(r.Context(), p.ID, true, latencyMs); err != nil {
				slog.Error("health check update proxy alive", "proxy_id", p.ID, "err", err)
			}
			results = append(results, healthCheckResult{
				ID:        p.ID,
				Name:      p.Name,
				Server:    p.Server,
				Port:      p.Port,
				Alive:     true,
				LatencyMs: latencyMs,
			})
		} else {
			if err := h.Store.UpdateProxyHealth(r.Context(), p.ID, false, 0); err != nil {
				slog.Error("health check update proxy dead", "proxy_id", p.ID, "err", err)
			}
			results = append(results, healthCheckResult{
				ID:        p.ID,
				Name:      p.Name,
				Server:    p.Server,
				Port:      p.Port,
				Alive:     false,
				LatencyMs: 0,
			})
		}
	}

	slog.Info("health check completed",
		"subscription_id", id,
		"checked", len(proxies),
		"alive", len(aliveResults),
	)

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"subscription_id": id,
		"checked":         len(proxies),
		"alive":           len(aliveResults),
		"results":         results,
	})
}
