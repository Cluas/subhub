package api

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/Cluas/subhub/internal/engine"
	"github.com/Cluas/subhub/internal/model"
	"github.com/Cluas/subhub/internal/store"
)

// EndpointHandler handles CRUD operations on endpoints and serves /p/{slug} public output.
type EndpointHandler struct {
	Store store.Store
}

// endpointRequest is the JSON body accepted by Create and Update.
type endpointRequest struct {
	Name           string                `json:"name"`
	Slug           string                `json:"slug"`
	SubscriptionID *int64                `json:"subscription_id"`
	CollectionID   *int64                `json:"collection_id"` // mutually exclusive with subscription_id
	OutputType     string                `json:"output_type"`
	Format         string                `json:"format"`
	Filters        model.EndpointFilters `json:"filters"`
}

// generateSlug returns a random 12-character hex slug.
func generateSlug() string {
	b := make([]byte, 6)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// List handles GET /api/endpoints — returns all endpoints.
func (h *EndpointHandler) List(w http.ResponseWriter, r *http.Request) {
	endpoints, err := h.Store.ListEndpoints(r.Context())
	if err != nil {
		slog.Error("list endpoints", "err", err)
		writeError(w, http.StatusInternalServerError, "failed to list endpoints")
		return
	}
	if endpoints == nil {
		endpoints = []*model.Endpoint{}
	}
	writeJSON(w, http.StatusOK, endpoints)
}

// Create handles POST /api/endpoints — creates a new endpoint with a generated slug.
func (h *EndpointHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req endpointRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	if req.OutputType == "" {
		req.OutputType = "proxy"
	}
	if req.Format == "" {
		req.Format = "clash"
	}

	// subscription_id and collection_id are mutually exclusive.
	if req.SubscriptionID != nil && *req.SubscriptionID != 0 &&
		req.CollectionID != nil && *req.CollectionID != 0 {
		writeError(w, http.StatusBadRequest, "subscription_id and collection_id are mutually exclusive")
		return
	}

	slug := req.Slug
	if slug == "" {
		slug = generateSlug()
	}

	ep := &model.Endpoint{
		Name:           req.Name,
		Slug:           slug,
		SubscriptionID: req.SubscriptionID,
		CollectionID:   req.CollectionID,
		OutputType:     req.OutputType,
		Format:         req.Format,
		Filters:        req.Filters,
	}

	err := h.Store.CreateEndpoint(r.Context(), ep)
	if err != nil {
		// Retry once on UNIQUE slug collision (astronomically rare).
		if strings.Contains(err.Error(), "UNIQUE constraint") {
			ep.Slug = generateSlug()
			err = h.Store.CreateEndpoint(r.Context(), ep)
		}
		if err != nil {
			slog.Error("create endpoint", "err", err)
			writeError(w, http.StatusInternalServerError, "failed to create endpoint")
			return
		}
	}
	writeJSON(w, http.StatusCreated, ep)
}

// Get handles GET /api/endpoints/{id} — returns a single endpoint.
func (h *EndpointHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	ep, err := h.Store.GetEndpoint(r.Context(), id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "endpoint not found")
			return
		}
		slog.Error("get endpoint", "id", id, "err", err)
		writeError(w, http.StatusInternalServerError, "failed to get endpoint")
		return
	}
	writeJSON(w, http.StatusOK, ep)
}

// Update handles PUT /api/endpoints/{id} — updates name, format, and filters.
func (h *EndpointHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	ep, err := h.Store.GetEndpoint(r.Context(), id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "endpoint not found")
			return
		}
		slog.Error("get endpoint for update", "id", id, "err", err)
		writeError(w, http.StatusInternalServerError, "failed to get endpoint")
		return
	}

	var req endpointRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.Name != "" {
		ep.Name = req.Name
	}
	if req.Slug != "" && req.Slug != ep.Slug {
		// Check slug uniqueness
		existing, slugErr := h.Store.GetEndpointBySlug(r.Context(), req.Slug)
		if slugErr == nil && existing.ID != ep.ID {
			writeError(w, http.StatusConflict, "slug already in use")
			return
		}
		ep.Slug = req.Slug
	}
	if req.OutputType != "" {
		ep.OutputType = req.OutputType
	}
	if req.Format != "" {
		ep.Format = req.Format
	}

	// subscription_id and collection_id are mutually exclusive.
	if req.SubscriptionID != nil && *req.SubscriptionID != 0 &&
		req.CollectionID != nil && *req.CollectionID != 0 {
		writeError(w, http.StatusBadRequest, "subscription_id and collection_id are mutually exclusive")
		return
	}
	if req.SubscriptionID != nil {
		ep.SubscriptionID = req.SubscriptionID
	}
	if req.CollectionID != nil {
		ep.CollectionID = req.CollectionID
	}

	ep.Filters = req.Filters

	if err := h.Store.UpdateEndpoint(r.Context(), ep); err != nil {
		slog.Error("update endpoint", "id", id, "err", err)
		writeError(w, http.StatusInternalServerError, "failed to update endpoint")
		return
	}
	writeJSON(w, http.StatusOK, ep)
}

// Delete handles DELETE /api/endpoints/{id} — deletes an endpoint.
func (h *EndpointHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.Store.DeleteEndpoint(r.Context(), id); err != nil {
		slog.Error("delete endpoint", "id", id, "err", err)
		writeError(w, http.StatusInternalServerError, "failed to delete endpoint")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// Preview handles GET /api/endpoints/{id}/preview — returns formatted output for an endpoint.
func (h *EndpointHandler) Preview(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	ep, err := h.Store.GetEndpoint(r.Context(), id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "endpoint not found")
			return
		}
		slog.Error("preview endpoint get", "id", id, "err", err)
		writeError(w, http.StatusInternalServerError, "failed to get endpoint")
		return
	}
	h.serveEndpoint(w, r, ep)
}

// ServeProvider handles GET /p/{slug} — public route, no auth required.
func (h *EndpointHandler) ServeProvider(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	ep, err := h.Store.GetEndpointBySlug(r.Context(), slug)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			slog.Warn("endpoint not found", "slug", slug)
			writeError(w, http.StatusNotFound, "endpoint not found")
			return
		}
		slog.Error("serve provider get endpoint", "slug", slug, "err", err)
		writeError(w, http.StatusInternalServerError, "failed to get endpoint")
		return
	}
	h.serveEndpoint(w, r, ep)
}

// serveEndpoint renders endpoint output, allowing ?format= query param to override stored format.
func (h *EndpointHandler) serveEndpoint(w http.ResponseWriter, r *http.Request, ep *model.Endpoint) {
	format := ep.Format
	if qf := r.URL.Query().Get("format"); qf != "" {
		format = qf
	}
	switch ep.OutputType {
	case "rule":
		h.serveRuleOutput(w, r, ep, format)
	default: // "proxy" and any unset value
		h.serveProxyOutput(w, r, ep, format)
	}
}

func (h *EndpointHandler) serveProxyOutput(w http.ResponseWriter, r *http.Request, ep *model.Endpoint, format string) {
	filter := endpointToProxyFilter(ep)

	// Resolve groups filter: look up the subscription's proxy-groups data and
	// convert group names to a list of member proxy names (Names filter).
	if len(ep.Filters.Groups) > 0 && filter.SubscriptionID != 0 {
		sub, err := h.Store.GetSubscription(r.Context(), filter.SubscriptionID)
		if err == nil && sub.ProxyGroupsData != nil {
			nameSet := make(map[string]bool)
			for _, groupFilter := range ep.Filters.Groups {
				for groupName, members := range sub.ProxyGroupsData.Groups {
					if strings.Contains(groupName, groupFilter) {
						for _, m := range members {
							nameSet[m] = true
						}
					}
				}
			}
			if len(nameSet) > 0 {
				names := make([]string, 0, len(nameSet))
				for n := range nameSet {
					names = append(names, n)
				}
				filter.Names = names
				filter.Groups = nil // clear the substring LIKE fallback
				slog.Info("resolved groups filter to proxy names",
					"slug", ep.Slug,
					"groups", ep.Filters.Groups,
					"resolved_count", len(names))
			}
		}
	}

	proxies, err := h.Store.ListProxies(r.Context(), filter)
	if err != nil {
		slog.Error("serve provider list proxies", "slug", ep.Slug, "err", err)
		writeError(w, http.StatusInternalServerError, "failed to list proxies")
		return
	}

	maps := make([]map[string]interface{}, 0, len(proxies))
	for _, p := range proxies {
		maps = append(maps, proxyToMap(p))
	}

	body, ct, err := engine.FormatProxyOutput(format, maps)
	if err != nil {
		slog.Error("serve provider format output", "slug", ep.Slug, "format", format, "err", err)
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to format output: %v", err))
		return
	}

	slog.Info("endpoint serve", "slug", ep.Slug, "format", format, "proxy_count", len(maps), "collection_id", ep.CollectionID)
	w.Header().Set("Content-Type", ct)
	w.WriteHeader(http.StatusOK)
	w.Write(body)
}

func (h *EndpointHandler) serveRuleOutput(w http.ResponseWriter, r *http.Request, ep *model.Endpoint, format string) {
	filter := store.RuleFilter{
		Target:  ep.Filters.Target,
		Keyword: ep.Filters.Keyword,
	}
	// Route by source: collection takes precedence over subscription.
	if ep.CollectionID != nil && *ep.CollectionID != 0 {
		filter.CollectionID = *ep.CollectionID
	} else {
		filter.SubscriptionID = derefInt64(ep.SubscriptionID)
	}
	if ep.Filters.RuleType != "" {
		filter.Type = ep.Filters.RuleType
	}
	if ep.Filters.Source != "" {
		filter.ProviderName = ep.Filters.Source
	}

	rules, err := h.Store.ListRules(r.Context(), filter)
	if err != nil {
		slog.Error("serve provider list rules", "slug", ep.Slug, "err", err)
		writeError(w, http.StatusInternalServerError, "failed to list rules")
		return
	}

	body, ct, err := engine.FormatRuleOutput(format, rules)
	if err != nil {
		slog.Error("serve provider format rule output", "slug", ep.Slug, "format", format, "err", err)
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("failed to format output: %v", err))
		return
	}

	slog.Info("endpoint serve rules", "slug", ep.Slug, "format", format, "rule_count", len(rules), "collection_id", ep.CollectionID)
	w.Header().Set("Content-Type", ct)
	w.WriteHeader(http.StatusOK)
	w.Write(body)
}

// endpointToProxyFilter maps an Endpoint's filters to a store.ProxyFilter.
// When the endpoint has a collection_id, it uses CollectionID as the source;
// otherwise it falls back to SubscriptionID (legacy behavior).
func endpointToProxyFilter(ep *model.Endpoint) store.ProxyFilter {
	filter := store.ProxyFilter{
		LatencyMax: ep.Filters.LatencyMax,
	}
	if ep.CollectionID != nil && *ep.CollectionID != 0 {
		filter.CollectionID = *ep.CollectionID
	} else {
		filter.SubscriptionID = derefInt64(ep.SubscriptionID)
	}
	if ep.Filters.AliveOnly {
		t := true
		filter.Alive = &t
	}
	if len(ep.Filters.Regions) > 0 {
		filter.Region = ep.Filters.Regions[0]
	}
	if len(ep.Filters.Types) > 0 {
		filter.Type = ep.Filters.Types[0]
	}
	if ep.Filters.NameContains != "" {
		filter.NameContains = ep.Filters.NameContains
	}
	// Groups: pass directly to store filter. Since proxy-group→member mapping is not
	// stored per row, the store interprets each group string as a name substring (OR).
	if len(ep.Filters.Groups) > 0 {
		filter.Groups = ep.Filters.Groups
	}
	return filter
}
