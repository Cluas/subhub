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

// RulesHandler handles REST operations on rules.
type RulesHandler struct {
	Store store.Store
}

// ruleRequest is the shape of the JSON body accepted by Create and Update.
type ruleRequest struct {
	SubscriptionID *int64 `json:"subscription_id"` // optional — nil for self-managed
	CollectionID   *int64 `json:"collection_id"`   // optional — mutually exclusive with subscription_id
	ProviderName   string `json:"provider_name"`
	Type           string `json:"type"`
	Payload        string `json:"payload"`
	Target         string `json:"target"`
}

// List handles GET /api/rules — returns rules matching optional query filters.
// Supported query params: subscription_id, provider, type, target, q.
// Always returns a JSON array (never null).
func (h *RulesHandler) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	filter := store.RuleFilter{}

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

	filter.ProviderName = q.Get("provider")
	filter.Type = q.Get("type")
	filter.Target = q.Get("target")
	filter.Keyword = q.Get("q")

	rules, err := h.Store.ListRules(r.Context(), filter)
	if err != nil {
		slog.Error("rules list: store failure", "err", err)
		writeError(w, http.StatusInternalServerError, "failed to list rules")
		return
	}

	// Ensure we send [] not null when the slice is empty.
	if rules == nil {
		rules = []*model.Rule{}
	}

	slog.Info("rules list",
		"count", len(rules),
		"filter_subscription_id", filter.SubscriptionID,
		"filter_collection_id", filter.CollectionID,
		"filter_type", filter.Type,
		"filter_provider", filter.ProviderName,
		"filter_keyword", filter.Keyword,
	)

	writeJSON(w, http.StatusOK, rules)
}

// Create handles POST /api/rules — creates a new self-managed (or subscription-bound) rule.
func (h *RulesHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req ruleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	if req.Type == "" {
		writeError(w, http.StatusBadRequest, "type is required")
		return
	}
	if req.Payload == "" {
		writeError(w, http.StatusBadRequest, "payload is required")
		return
	}
	if req.Target == "" {
		writeError(w, http.StatusBadRequest, "target is required")
		return
	}

	// subscription_id and collection_id are mutually exclusive.
	if req.SubscriptionID != nil && *req.SubscriptionID != 0 &&
		req.CollectionID != nil && *req.CollectionID != 0 {
		writeError(w, http.StatusBadRequest, "subscription_id and collection_id are mutually exclusive")
		return
	}

	rule := &model.Rule{
		SubscriptionID: req.SubscriptionID,
		CollectionID:   req.CollectionID,
		ProviderName:   req.ProviderName,
		Type:           req.Type,
		Payload:        req.Payload,
		Target:         req.Target,
	}

	if err := h.Store.CreateRule(r.Context(), rule); err != nil {
		slog.Error("rule create: store failure", "err", err)
		writeError(w, http.StatusInternalServerError, "failed to create rule")
		return
	}

	slog.Info("rule created", "id", rule.ID, "type", rule.Type, "subscription_id", rule.SubscriptionID, "collection_id", rule.CollectionID)
	writeJSON(w, http.StatusCreated, rule)
}

// Get handles GET /api/rules/{id} — returns a single rule by ID.
func (h *RulesHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid rule id")
		return
	}

	rule, err := h.Store.GetRule(r.Context(), id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "rule not found")
			return
		}
		slog.Error("rule get: store failure", "id", id, "err", err)
		writeError(w, http.StatusInternalServerError, "failed to get rule")
		return
	}

	writeJSON(w, http.StatusOK, rule)
}

// Update handles PUT /api/rules/{id} — merges JSON body fields onto the existing rule.
func (h *RulesHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid rule id")
		return
	}

	existing, err := h.Store.GetRule(r.Context(), id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "rule not found")
			return
		}
		slog.Error("rule update get: store failure", "id", id, "err", err)
		writeError(w, http.StatusInternalServerError, "failed to get rule")
		return
	}

	var req ruleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	// Merge non-zero fields.
	if req.ProviderName != "" {
		existing.ProviderName = req.ProviderName
	}
	if req.Type != "" {
		existing.Type = req.Type
	}
	if req.Payload != "" {
		existing.Payload = req.Payload
	}
	if req.Target != "" {
		existing.Target = req.Target
	}

	if err := h.Store.UpdateRule(r.Context(), existing); err != nil {
		slog.Error("rule update: store failure", "id", id, "err", err)
		writeError(w, http.StatusInternalServerError, "failed to update rule")
		return
	}

	slog.Info("rule updated", "id", id)
	writeJSON(w, http.StatusOK, existing)
}

// Delete handles DELETE /api/rules/{id} — removes a rule by ID.
func (h *RulesHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid rule id")
		return
	}

	if err := h.Store.DeleteRule(r.Context(), id); err != nil {
		slog.Error("rule delete: store failure", "id", id, "err", err)
		writeError(w, http.StatusInternalServerError, "failed to delete rule")
		return
	}

	slog.Info("rule deleted", "id", id)
	w.WriteHeader(http.StatusNoContent)
}
