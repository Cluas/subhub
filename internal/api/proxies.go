package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/Cluas/subhub/internal/model"
	"github.com/Cluas/subhub/internal/store"
)

// ProxiesHandler handles REST operations on proxies.
type ProxiesHandler struct {
	Store store.Store
}

// proxyRequest is the shape of the JSON body accepted by Create and Update.
type proxyRequest struct {
	SubscriptionID *int64         `json:"subscription_id"` // optional — nil for self-managed
	CollectionID   *int64         `json:"collection_id"`   // optional — mutually exclusive with subscription_id
	Name           string         `json:"name"`
	Type           string         `json:"type"`
	Server         string         `json:"server"`
	Port           int            `json:"port"`
	Config         map[string]any `json:"config"`
	Region         string         `json:"region"`
}

// List handles GET /api/proxies — returns proxies matching optional query filters.
// Supported query params: subscription_id, type, region, latency_max, alive.
// Always returns a JSON array (never null).
func (h *ProxiesHandler) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	filter := store.ProxyFilter{}

	if raw := q.Get("subscription_id"); raw != "" {
		id, err := strconv.ParseInt(raw, 10, 64)
		if err == nil {
			filter.SubscriptionID = id
		}
	}

	if raw := q.Get("collection_id"); raw != "" {
		id, err := strconv.ParseInt(raw, 10, 64)
		if err == nil {
			filter.CollectionID = id
		}
	}

	filter.Type = q.Get("type")
	filter.Region = q.Get("region")
	filter.NameContains = q.Get("name_contains")

	if raw := q.Get("latency_max"); raw != "" {
		if v, err := strconv.Atoi(raw); err == nil {
			filter.LatencyMax = v
		}
	}

	// Reuse parseAlive from provider.go (same package).
	filter.Alive = parseAlive(r)

	proxies, err := h.Store.ListProxies(r.Context(), filter)
	if err != nil {
		slog.Error("proxies list: store failure", "err", err)
		writeError(w, http.StatusInternalServerError, "failed to list proxies")
		return
	}

	// Ensure we send [] not null when the slice is empty.
	if proxies == nil {
		proxies = []*model.Proxy{}
	}

	slog.Info("proxies list",
		"count", len(proxies),
		"filter_subscription_id", filter.SubscriptionID,
		"filter_collection_id", filter.CollectionID,
		"filter_type", filter.Type,
		"filter_alive", filter.Alive,
	)

	writeJSON(w, http.StatusOK, proxies)
}

// Create handles POST /api/proxies — creates a new self-managed (or subscription-bound) proxy.
func (h *ProxiesHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req proxyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	if req.Type == "" {
		writeError(w, http.StatusBadRequest, "type is required")
		return
	}
	if req.Server == "" {
		writeError(w, http.StatusBadRequest, "server is required")
		return
	}
	if req.Port <= 0 || req.Port > 65535 {
		writeError(w, http.StatusBadRequest, "port must be between 1 and 65535")
		return
	}

	// subscription_id and collection_id are mutually exclusive.
	if req.SubscriptionID != nil && *req.SubscriptionID != 0 &&
		req.CollectionID != nil && *req.CollectionID != 0 {
		writeError(w, http.StatusBadRequest, "subscription_id and collection_id are mutually exclusive")
		return
	}

	p := &model.Proxy{
		SubscriptionID: req.SubscriptionID,
		CollectionID:   req.CollectionID,
		Name:           req.Name,
		Type:           req.Type,
		Server:         req.Server,
		Port:           req.Port,
		Config:         req.Config,
		Region:         req.Region,
	}

	if err := h.Store.CreateProxy(r.Context(), p); err != nil {
		slog.Error("proxy create: store failure", "err", err)
		writeError(w, http.StatusInternalServerError, "failed to create proxy")
		return
	}

	slog.Info("proxy created", "id", p.ID, "name", p.Name, "subscription_id", p.SubscriptionID, "collection_id", p.CollectionID)
	writeJSON(w, http.StatusCreated, p)
}

// Get handles GET /api/proxies/{id} — returns a single proxy by ID.
func (h *ProxiesHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid proxy id")
		return
	}

	p, err := h.Store.GetProxy(r.Context(), id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "proxy not found")
			return
		}
		slog.Error("proxy get: store failure", "id", id, "err", err)
		writeError(w, http.StatusInternalServerError, "failed to get proxy")
		return
	}

	writeJSON(w, http.StatusOK, p)
}

// Update handles PUT /api/proxies/{id} — merges JSON body fields onto the existing proxy.
func (h *ProxiesHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid proxy id")
		return
	}

	existing, err := h.Store.GetProxy(r.Context(), id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "proxy not found")
			return
		}
		slog.Error("proxy update get: store failure", "id", id, "err", err)
		writeError(w, http.StatusInternalServerError, "failed to get proxy")
		return
	}

	var req proxyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	// Merge non-zero fields.
	if req.Name != "" {
		existing.Name = req.Name
	}
	if req.Type != "" {
		existing.Type = req.Type
	}
	if req.Server != "" {
		existing.Server = req.Server
	}
	if req.Port > 0 {
		existing.Port = req.Port
	}
	if req.Config != nil {
		existing.Config = req.Config
	}
	if req.Region != "" {
		existing.Region = req.Region
	}

	if err := h.Store.UpdateProxy(r.Context(), existing); err != nil {
		slog.Error("proxy update: store failure", "id", id, "err", err)
		writeError(w, http.StatusInternalServerError, "failed to update proxy")
		return
	}

	slog.Info("proxy updated", "id", id)
	writeJSON(w, http.StatusOK, existing)
}

// Delete handles DELETE /api/proxies/{id} — removes a proxy by ID.
func (h *ProxiesHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid proxy id")
		return
	}

	if err := h.Store.DeleteProxy(r.Context(), id); err != nil {
		slog.Error("proxy delete: store failure", "id", id, "err", err)
		writeError(w, http.StatusInternalServerError, "failed to delete proxy")
		return
	}

	slog.Info("proxy deleted", "id", id)
	w.WriteHeader(http.StatusNoContent)
}
