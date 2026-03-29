package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/Cluas/subhub/internal/model"
	"github.com/Cluas/subhub/internal/store"
)

// CollectionHandler handles CRUD operations on collections and their content.
type CollectionHandler struct {
	Store store.Store
}

// collectionRequest is the JSON body accepted by Create and Update.
type collectionRequest struct {
	Name        string `json:"name"`
	ContentType string `json:"content_type"` // "proxy" | "rule"
	Description string `json:"description"`
}

// List handles GET /api/collections — returns all collections.
func (h *CollectionHandler) List(w http.ResponseWriter, r *http.Request) {
	collections, err := h.Store.ListCollections(r.Context())
	if err != nil {
		slog.Error("collection list: store failure", "err", err)
		writeError(w, http.StatusInternalServerError, "failed to list collections")
		return
	}
	if collections == nil {
		collections = []*model.Collection{}
	}
	slog.Info("collection list", "count", len(collections))
	writeJSON(w, http.StatusOK, collections)
}

// Create handles POST /api/collections — creates a new collection.
func (h *CollectionHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req collectionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	if req.ContentType != "proxy" && req.ContentType != "rule" {
		writeError(w, http.StatusBadRequest, "content_type must be 'proxy' or 'rule'")
		return
	}

	c := &model.Collection{
		Name:        req.Name,
		ContentType: req.ContentType,
		Description: req.Description,
	}
	if err := h.Store.CreateCollection(r.Context(), c); err != nil {
		slog.Error("collection create: store failure", "err", err)
		writeError(w, http.StatusInternalServerError, "failed to create collection")
		return
	}
	slog.Info("collection created", "id", c.ID, "name", c.Name, "content_type", c.ContentType)
	writeJSON(w, http.StatusCreated, c)
}

// Get handles GET /api/collections/{id} — returns a single collection.
func (h *CollectionHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := parseCollectionID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid collection id")
		return
	}
	c, err := h.Store.GetCollection(r.Context(), id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "collection not found")
			return
		}
		slog.Error("collection get: store failure", "id", id, "err", err)
		writeError(w, http.StatusInternalServerError, "failed to get collection")
		return
	}
	writeJSON(w, http.StatusOK, c)
}

// Update handles PUT /api/collections/{id} — updates name and/or description.
// content_type is immutable once created.
func (h *CollectionHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := parseCollectionID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid collection id")
		return
	}
	c, err := h.Store.GetCollection(r.Context(), id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "collection not found")
			return
		}
		slog.Error("collection update: get failure", "id", id, "err", err)
		writeError(w, http.StatusInternalServerError, "failed to get collection")
		return
	}

	var req collectionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.Name != "" {
		c.Name = req.Name
	}
	if req.Description != "" {
		c.Description = req.Description
	}
	// content_type is intentionally immutable — ignore any change in req.

	if err := h.Store.UpdateCollection(r.Context(), c); err != nil {
		slog.Error("collection update: store failure", "id", id, "err", err)
		writeError(w, http.StatusInternalServerError, "failed to update collection")
		return
	}
	slog.Info("collection updated", "id", id, "name", c.Name)
	writeJSON(w, http.StatusOK, c)
}

// Delete handles DELETE /api/collections/{id} — deletes the collection and all its content.
func (h *CollectionHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := parseCollectionID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid collection id")
		return
	}
	if err := h.Store.DeleteCollection(r.Context(), id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "collection not found")
			return
		}
		slog.Error("collection delete: store failure", "id", id, "err", err)
		writeError(w, http.StatusInternalServerError, "failed to delete collection")
		return
	}
	slog.Info("collection deleted", "id", id)
	w.WriteHeader(http.StatusNoContent)
}

// ListProxies handles GET /api/collections/{id}/proxies.
// Returns all proxies belonging to the collection.
func (h *CollectionHandler) ListProxies(w http.ResponseWriter, r *http.Request) {
	id, err := parseCollectionID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid collection id")
		return
	}
	proxies, err := h.Store.ListProxies(r.Context(), store.ProxyFilter{CollectionID: id})
	if err != nil {
		slog.Error("collection list proxies: store failure", "collection_id", id, "err", err)
		writeError(w, http.StatusInternalServerError, "failed to list proxies")
		return
	}
	if proxies == nil {
		proxies = []*model.Proxy{}
	}
	slog.Info("collection list proxies", "collection_id", id, "count", len(proxies))
	writeJSON(w, http.StatusOK, proxies)
}

// ListRules handles GET /api/collections/{id}/rules.
// Returns all rules belonging to the collection.
func (h *CollectionHandler) ListRules(w http.ResponseWriter, r *http.Request) {
	id, err := parseCollectionID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid collection id")
		return
	}
	rules, err := h.Store.ListRules(r.Context(), store.RuleFilter{CollectionID: id})
	if err != nil {
		slog.Error("collection list rules: store failure", "collection_id", id, "err", err)
		writeError(w, http.StatusInternalServerError, "failed to list rules")
		return
	}
	if rules == nil {
		rules = []*model.Rule{}
	}
	slog.Info("collection list rules", "collection_id", id, "count", len(rules))
	writeJSON(w, http.StatusOK, rules)
}

// parseCollectionID reads the "id" URL param (set by chi) as int64.
// Reuses parseID which reads chi URLParam("id").
func parseCollectionID(r *http.Request) (int64, error) {
	return parseID(r)
}
