package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/Cluas/subhub/internal/model"
	profilepkg "github.com/Cluas/subhub/internal/profile"
	"github.com/Cluas/subhub/internal/store"
)

// ProfileHandler handles CRUD operations on profiles and their sub-resources.
type ProfileHandler struct {
	Store   store.Store
	BaseURL string
}

// ServeProfile handles GET /profile/{id} — renders the profile as Mihomo/Clash YAML.
func (h *ProfileHandler) ServeProfile(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "id")
	ctx := r.Context()

	profile, err := h.Store.GetProfileBySlug(ctx, slug)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.NotFound(w, r)
			return
		}
		slog.Error("serve profile: get by slug", "slug", slug, "err", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	pools, err := h.Store.ListProfileNodePools(ctx, profile.ID)
	if err != nil {
		slog.Error("serve profile: list node pools", "profile_id", profile.ID, "err", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	ruleSets, err := h.Store.ListProfileRuleSets(ctx, profile.ID)
	if err != nil {
		slog.Error("serve profile: list rule sets", "profile_id", profile.ID, "err", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	strategies, err := h.Store.ListProfileStrategies(ctx, profile.ID)
	if err != nil {
		slog.Error("serve profile: list strategies", "profile_id", profile.ID, "err", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	routingRules, err := h.Store.ListProfileRoutingRules(ctx, profile.ID)
	if err != nil {
		slog.Error("serve profile: list routing rules", "profile_id", profile.ID, "err", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	nodePools := make([]profilepkg.NodePool, len(pools))
	for i, np := range pools {
		nodePools[i] = np
	}
	rs := make([]profilepkg.RuleSet, len(ruleSets))
	for i, ruleSet := range ruleSets {
		rs[i] = ruleSet
	}
	groups := make([]profilepkg.Group, len(strategies))
	for i, s := range strategies {
		groups[i] = s
	}
	rr := make([]profilepkg.RoutingRule, len(routingRules))
	for i, r := range routingRules {
		rr[i] = r
	}

	input := profilepkg.RenderInput{
		Settings:     profile.Settings,
		NodePools:    nodePools,
		RuleSets:     rs,
		Groups:       groups,
		RoutingRules: rr,
		BaseURL:      h.Store.GetSystemSetting(ctx, "base_url", h.BaseURL),
	}

	renderer := &profilepkg.MihomoRenderer{}
	out, err := renderer.Render(&input)
	if err != nil {
		slog.Error("serve profile: render", "profile_id", profile.ID, "slug", slug, "err", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", renderer.ContentType())
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(out)
}

// profileRequest is the JSON body accepted by Profile Create and Update.
type profileRequest struct {
	Name     string                 `json:"name"`
	Slug     string                 `json:"slug"`
	Settings map[string]interface{} `json:"settings"`
}

// List handles GET /api/profiles — returns all profiles.
func (h *ProfileHandler) List(w http.ResponseWriter, r *http.Request) {
	profiles, err := h.Store.ListProfiles(r.Context())
	if err != nil {
		slog.Error("list profiles", "err", err)
		writeError(w, http.StatusInternalServerError, "failed to list profiles")
		return
	}
	if profiles == nil {
		profiles = []*model.Profile{}
	}
	writeJSON(w, http.StatusOK, profiles)
}

// Create handles POST /api/profiles — creates a new profile with a generated slug.
func (h *ProfileHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req profileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	p := &model.Profile{
		Name: req.Name,
		Slug: generateSlug(),
	}

	err := h.Store.CreateProfile(r.Context(), p)
	if err != nil {
		// Retry once on UNIQUE slug collision (astronomically rare).
		if strings.Contains(err.Error(), "UNIQUE constraint") {
			p.Slug = generateSlug()
			err = h.Store.CreateProfile(r.Context(), p)
		}
		if err != nil {
			slog.Error("create profile", "err", err)
			writeError(w, http.StatusInternalServerError, "failed to create profile")
			return
		}
	}
	slog.Info("profile created", "id", p.ID, "slug", p.Slug)
	writeJSON(w, http.StatusCreated, p)
}

// Get handles GET /api/profiles/{id} — returns a single profile.
func (h *ProfileHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	p, err := h.Store.GetProfile(r.Context(), id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "profile not found")
			return
		}
		slog.Error("get profile", "id", id, "err", err)
		writeError(w, http.StatusInternalServerError, "failed to get profile")
		return
	}
	writeJSON(w, http.StatusOK, p)
}

// Update handles PUT /api/profiles/{id} — updates name and saves.
func (h *ProfileHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	p, err := h.Store.GetProfile(r.Context(), id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "profile not found")
			return
		}
		slog.Error("get profile for update", "id", id, "err", err)
		writeError(w, http.StatusInternalServerError, "failed to get profile")
		return
	}

	var req profileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.Name != "" {
		p.Name = req.Name
	}
	if req.Slug != "" && req.Slug != p.Slug {
		// Check slug uniqueness
		existing, err := h.Store.GetProfileBySlug(r.Context(), req.Slug)
		if err == nil && existing.ID != p.ID {
			writeError(w, http.StatusConflict, "slug already in use")
			return
		}
		p.Slug = req.Slug
	}
	if req.Settings != nil {
		p.Settings = req.Settings
	}

	if err := h.Store.UpdateProfile(r.Context(), p); err != nil {
		slog.Error("update profile", "id", id, "err", err)
		writeError(w, http.StatusInternalServerError, "failed to update profile")
		return
	}
	writeJSON(w, http.StatusOK, p)
}

// Delete handles DELETE /api/profiles/{id} — deletes a profile (cascades to sub-resources).
func (h *ProfileHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.Store.DeleteProfile(r.Context(), id); err != nil {
		slog.Error("delete profile", "id", id, "err", err)
		writeError(w, http.StatusInternalServerError, "failed to delete profile")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ListNodePools handles GET /api/profiles/{id}/node-pools — returns node pools for a profile.
func (h *ProfileHandler) ListNodePools(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	items, err := h.Store.ListProfileNodePools(r.Context(), id)
	if err != nil {
		slog.Error("list profile node pools", "profile_id", id, "err", err)
		writeError(w, http.StatusInternalServerError, "failed to list node pools")
		return
	}
	if items == nil {
		items = []*model.ProfileNodePool{}
	}
	writeJSON(w, http.StatusOK, items)
}

// ListRuleSets handles GET /api/profiles/{id}/rule-sets — returns rule sets for a profile.
func (h *ProfileHandler) ListRuleSets(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	items, err := h.Store.ListProfileRuleSets(r.Context(), id)
	if err != nil {
		slog.Error("list profile rule sets", "profile_id", id, "err", err)
		writeError(w, http.StatusInternalServerError, "failed to list rule sets")
		return
	}
	if items == nil {
		items = []*model.ProfileRuleSet{}
	}
	writeJSON(w, http.StatusOK, items)
}

// ListStrategies handles GET /api/profiles/{id}/strategies — returns strategies for a profile.
func (h *ProfileHandler) ListStrategies(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	items, err := h.Store.ListProfileStrategies(r.Context(), id)
	if err != nil {
		slog.Error("list profile strategies", "profile_id", id, "err", err)
		writeError(w, http.StatusInternalServerError, "failed to list strategies")
		return
	}
	if items == nil {
		items = []*model.ProfileStrategy{}
	}
	writeJSON(w, http.StatusOK, items)
}

// ListRoutingRules handles GET /api/profiles/{id}/routing-rules — returns routing rules for a profile.
func (h *ProfileHandler) ListRoutingRules(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	items, err := h.Store.ListProfileRoutingRules(r.Context(), id)
	if err != nil {
		slog.Error("list profile routing rules", "profile_id", id, "err", err)
		writeError(w, http.StatusInternalServerError, "failed to list routing rules")
		return
	}
	if items == nil {
		items = []*model.ProfileRoutingRule{}
	}
	writeJSON(w, http.StatusOK, items)
}

// ─── Create sub-resource handlers ────────────────────────────────────────────

// nodePoolRequest is the JSON body for CreateNodePool.
type nodePoolRequest struct {
	Name                string `json:"name"`
	EndpointID          *int64 `json:"endpoint_id"`
	EndpointSlug        string `json:"endpoint_slug"`
	HealthCheckURL      string `json:"health_check_url"`
	HealthCheckInterval int    `json:"health_check_interval"`
	Position            int    `json:"position"`
}

// CreateNodePool handles POST /api/profiles/{id}/node-pools.
func (h *ProfileHandler) CreateNodePool(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	var req nodePoolRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	if req.HealthCheckURL == "" {
		req.HealthCheckURL = "http://www.gstatic.com/generate_204"
	}
	if req.HealthCheckInterval == 0 {
		req.HealthCheckInterval = 300
	}
	// Resolve endpoint slug if endpoint_id is provided and slug is empty
	slug := req.EndpointSlug
	if slug == "" && req.EndpointID != nil {
		if ep, epErr := h.Store.GetEndpoint(r.Context(), *req.EndpointID); epErr == nil {
			slug = ep.Slug
		}
	}
	np := &model.ProfileNodePool{
		ProfileID:           id,
		NameStr:             req.Name,
		EndpointID:          req.EndpointID,
		EndpointSlugValue:   slug,
		HealthCheckURL:      req.HealthCheckURL,
		HealthCheckInterval: req.HealthCheckInterval,
		Position:            req.Position,
	}
	if err := h.Store.CreateProfileNodePool(r.Context(), np); err != nil {
		slog.Error("create profile node pool", "profile_id", id, "err", err)
		writeError(w, http.StatusInternalServerError, "failed to create node pool")
		return
	}
	writeJSON(w, http.StatusCreated, np)
}

// ruleSetRequest is the JSON body for CreateRuleSet.
type ruleSetRequest struct {
	Name         string         `json:"name"`
	EndpointID   *int64         `json:"endpoint_id"`
	EndpointSlug string         `json:"endpoint_slug"`
	URL          string         `json:"url"`
	Metadata     map[string]any `json:"metadata"`
	Interval     int            `json:"interval"`
	Position     int            `json:"position"`
}

// CreateRuleSet handles POST /api/profiles/{id}/rule-sets.
func (h *ProfileHandler) CreateRuleSet(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	var req ruleSetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	// Resolve endpoint slug if endpoint_id is provided and slug is empty
	rsSlug := req.EndpointSlug
	if rsSlug == "" && req.EndpointID != nil {
		if ep, epErr := h.Store.GetEndpoint(r.Context(), *req.EndpointID); epErr == nil {
			rsSlug = ep.Slug
		}
	}
	interval := req.Interval
	if interval == 0 {
		interval = 86400
	}
	rs := &model.ProfileRuleSet{
		ProfileID:         id,
		NameStr:           req.Name,
		EndpointSlugValue: rsSlug,
		ExternalURL:       req.URL,
		MetadataJSON:      req.Metadata,
	}
	if err := h.Store.CreateProfileRuleSet(r.Context(), rs); err != nil {
		slog.Error("create profile rule set", "profile_id", id, "err", err)
		writeError(w, http.StatusInternalServerError, "failed to create rule set")
		return
	}
	writeJSON(w, http.StatusCreated, rs)
}

// strategyRequest is the JSON body for CreateStrategy.
type strategyRequest struct {
	Name     string         `json:"name"`
	Strategy string         `json:"strategy"`
	Pools    []string       `json:"pools"`
	Proxies  []string       `json:"proxies"`
	Config   map[string]any `json:"config"`
	Position int            `json:"position"`
}

// CreateStrategy handles POST /api/profiles/{id}/strategies.
func (h *ProfileHandler) CreateStrategy(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	var req strategyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	if req.Strategy == "" {
		req.Strategy = "select"
	}
	st := &model.ProfileStrategy{
		ProfileID:  id,
		NameStr:    req.Name,
		StrategyV:  req.Strategy,
		PoolNames:  req.Pools,
		ProxyNames: req.Proxies,
		ConfigJSON: req.Config,
		Position:   req.Position,
	}
	if err := h.Store.CreateProfileStrategy(r.Context(), st); err != nil {
		slog.Error("create profile strategy", "profile_id", id, "err", err)
		writeError(w, http.StatusInternalServerError, "failed to create strategy")
		return
	}
	writeJSON(w, http.StatusCreated, st)
}

// routingRuleRequest is the JSON body for CreateRoutingRule.
type routingRuleRequest struct {
	Match     string `json:"match"`
	Value     string `json:"value"`
	Target    string `json:"target"`
	Position  int    `json:"position"`
	NoResolve bool   `json:"no_resolve"`
}

// CreateRoutingRule handles POST /api/profiles/{id}/routing-rules.
func (h *ProfileHandler) CreateRoutingRule(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	var req routingRuleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.Match == "" {
		writeError(w, http.StatusBadRequest, "match is required")
		return
	}
	if req.Target == "" {
		writeError(w, http.StatusBadRequest, "target is required")
		return
	}
	rr := &model.ProfileRoutingRule{
		ProfileID: id,
		Type:      req.Match,
		Payload:   req.Value,
		TargetStr: req.Target,
		PositionV: req.Position,
		NoResolveV: req.NoResolve,
	}
	if err := h.Store.CreateProfileRoutingRule(r.Context(), rr); err != nil {
		slog.Error("create profile routing rule", "profile_id", id, "err", err)
		writeError(w, http.StatusInternalServerError, "failed to create routing rule")
		return
	}
	writeJSON(w, http.StatusCreated, rr)
}

// ─── Update sub-resource handlers ────────────────────────────────────────────

// UpdateNodePool handles PUT /api/profiles/{id}/node-pools/{bid}.
func (h *ProfileHandler) UpdateNodePool(w http.ResponseWriter, r *http.Request) {
	bid, err := parseBIDParam(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	np, err := h.Store.GetProfileNodePool(r.Context(), bid)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "node pool not found")
			return
		}
		slog.Error("get profile node pool for update", "id", bid, "err", err)
		writeError(w, http.StatusInternalServerError, "failed to get node pool")
		return
	}
	var req nodePoolRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.Name != "" {
		np.NameStr = req.Name
	}
	if req.EndpointID != nil {
		np.EndpointID = req.EndpointID
	}
	if req.EndpointSlug != "" {
		np.EndpointSlugValue = req.EndpointSlug
	}
	// Resolve endpoint slug if endpoint_id is provided and slug is empty
	if np.EndpointSlugValue == "" && np.EndpointID != nil {
		if ep, epErr := h.Store.GetEndpoint(r.Context(), *np.EndpointID); epErr == nil {
			np.EndpointSlugValue = ep.Slug
		}
	}
	if req.HealthCheckURL != "" {
		np.HealthCheckURL = req.HealthCheckURL
	}
	if req.HealthCheckInterval != 0 {
		np.HealthCheckInterval = req.HealthCheckInterval
	}
	if req.Position != 0 {
		np.Position = req.Position
	}
	if err := h.Store.UpdateProfileNodePool(r.Context(), np); err != nil {
		slog.Error("update profile node pool", "id", bid, "err", err)
		writeError(w, http.StatusInternalServerError, "failed to update node pool")
		return
	}
	writeJSON(w, http.StatusOK, np)
}

// UpdateRuleSet handles PUT /api/profiles/{id}/rule-sets/{bid}.
func (h *ProfileHandler) UpdateRuleSet(w http.ResponseWriter, r *http.Request) {
	bid, err := parseBIDParam(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	rs, err := h.Store.GetProfileRuleSet(r.Context(), bid)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "rule set not found")
			return
		}
		slog.Error("get profile rule set for update", "id", bid, "err", err)
		writeError(w, http.StatusInternalServerError, "failed to get rule set")
		return
	}
	var req ruleSetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.Name != "" {
		rs.NameStr = req.Name
	}
	if req.EndpointSlug != "" {
		rs.EndpointSlugValue = req.EndpointSlug
	}
	// Resolve endpoint slug if endpoint_id is provided and slug is empty
	if rs.EndpointSlugValue == "" && req.EndpointID != nil {
		if ep, epErr := h.Store.GetEndpoint(r.Context(), *req.EndpointID); epErr == nil {
			rs.EndpointSlugValue = ep.Slug
		}
	}
	if req.URL != "" {
		rs.ExternalURL = req.URL
	}
	if req.Metadata != nil {
		rs.MetadataJSON = req.Metadata
	}
	if req.Interval != 0 {
		rs.Interval = req.Interval
	}
	if req.Position != 0 {
		rs.Position = req.Position
	}
	if err := h.Store.UpdateProfileRuleSet(r.Context(), rs); err != nil {
		slog.Error("update profile rule set", "id", bid, "err", err)
		writeError(w, http.StatusInternalServerError, "failed to update rule set")
		return
	}
	writeJSON(w, http.StatusOK, rs)
}

// UpdateStrategy handles PUT /api/profiles/{id}/strategies/{bid}.
func (h *ProfileHandler) UpdateStrategy(w http.ResponseWriter, r *http.Request) {
	bid, err := parseBIDParam(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	st, err := h.Store.GetProfileStrategy(r.Context(), bid)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "strategy not found")
			return
		}
		slog.Error("get profile strategy for update", "id", bid, "err", err)
		writeError(w, http.StatusInternalServerError, "failed to get strategy")
		return
	}
	var req strategyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.Name != "" {
		st.NameStr = req.Name
	}
	if req.Strategy != "" {
		st.StrategyV = req.Strategy
	}
	if req.Pools != nil {
		st.PoolNames = req.Pools
	}
	if req.Proxies != nil {
		st.ProxyNames = req.Proxies
	}
	if req.Config != nil {
		st.ConfigJSON = req.Config
	}
	if req.Position != 0 {
		st.Position = req.Position
	}
	if err := h.Store.UpdateProfileStrategy(r.Context(), st); err != nil {
		slog.Error("update profile strategy", "id", bid, "err", err)
		writeError(w, http.StatusInternalServerError, "failed to update strategy")
		return
	}
	writeJSON(w, http.StatusOK, st)
}

// UpdateRoutingRule handles PUT /api/profiles/{id}/routing-rules/{bid}.
func (h *ProfileHandler) UpdateRoutingRule(w http.ResponseWriter, r *http.Request) {
	bid, err := parseBIDParam(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	rr, err := h.Store.GetProfileRoutingRule(r.Context(), bid)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "routing rule not found")
			return
		}
		slog.Error("get profile routing rule for update", "id", bid, "err", err)
		writeError(w, http.StatusInternalServerError, "failed to get routing rule")
		return
	}
	var req routingRuleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.Match != "" {
		rr.Type = req.Match
	}
	if req.Value != "" {
		rr.Payload = req.Value
	}
	if req.Target != "" {
		rr.TargetStr = req.Target
	}
	if req.Position != 0 {
		rr.PositionV = req.Position
	}
	rr.NoResolveV = req.NoResolve
	if err := h.Store.UpdateProfileRoutingRule(r.Context(), rr); err != nil {
		slog.Error("update profile routing rule", "id", bid, "err", err)
		writeError(w, http.StatusInternalServerError, "failed to update routing rule")
		return
	}
	writeJSON(w, http.StatusOK, rr)
}

// parseBIDParam extracts chi URL param "bid" (sub-resource ID) as int64.
// Symmetric with parseID (which reads "id") for nested sub-resource routes.
func parseBIDParam(r *http.Request) (int64, error) {
	raw := chi.URLParam(r, "bid")
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return 0, errors.New("invalid bid")
	}
	return id, nil
}

// DeleteNodePool handles DELETE /api/profiles/{id}/node-pools/{bid}.
func (h *ProfileHandler) DeleteNodePool(w http.ResponseWriter, r *http.Request) {
	bid, err := parseBIDParam(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.Store.DeleteProfileNodePool(r.Context(), bid); err != nil {
		slog.Error("delete profile node pool", "id", bid, "err", err)
		writeError(w, http.StatusInternalServerError, "failed to delete node pool")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// DeleteRuleSet handles DELETE /api/profiles/{id}/rule-sets/{bid}.
func (h *ProfileHandler) DeleteRuleSet(w http.ResponseWriter, r *http.Request) {
	bid, err := parseBIDParam(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.Store.DeleteProfileRuleSet(r.Context(), bid); err != nil {
		slog.Error("delete profile rule set", "id", bid, "err", err)
		writeError(w, http.StatusInternalServerError, "failed to delete rule set")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// DeleteStrategy handles DELETE /api/profiles/{id}/strategies/{bid}.
func (h *ProfileHandler) DeleteStrategy(w http.ResponseWriter, r *http.Request) {
	bid, err := parseBIDParam(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.Store.DeleteProfileStrategy(r.Context(), bid); err != nil {
		slog.Error("delete profile strategy", "id", bid, "err", err)
		writeError(w, http.StatusInternalServerError, "failed to delete strategy")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// DeleteRoutingRule handles DELETE /api/profiles/{id}/routing-rules/{bid}.
func (h *ProfileHandler) DeleteRoutingRule(w http.ResponseWriter, r *http.Request) {
	bid, err := parseBIDParam(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.Store.DeleteProfileRoutingRule(r.Context(), bid); err != nil {
		slog.Error("delete profile routing rule", "id", bid, "err", err)
		writeError(w, http.StatusInternalServerError, "failed to delete routing rule")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
