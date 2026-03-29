package api

import (
	"database/sql"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"sigs.k8s.io/yaml"

	"github.com/Cluas/subhub/internal/engine"
	"github.com/Cluas/subhub/internal/model"
	"github.com/Cluas/subhub/internal/store"
)

// ProviderHandler handles provider output endpoints that read from the DB.
type ProviderHandler struct {
	Store store.Store
}

// proxyToMap converts a stored model.Proxy back to the flat map[string]interface{}
// format that engine.ProxiesToLinks and model.Provider.Proxies expect.
// Config is copied first so canonical fields (name, type, server, port) always win.
func proxyToMap(p *model.Proxy) map[string]interface{} {
	m := make(map[string]interface{})
	for k, v := range p.Config {
		m[k] = v
	}
	m["name"] = p.Name
	m["type"] = p.Type
	m["server"] = p.Server
	m["port"] = p.Port
	return m
}

// parseAlive parses an optional ?alive= query param ("true"/"false") into *bool.
// Returns nil if the param is absent or unparseable.
func parseAlive(r *http.Request) *bool {
	raw := r.URL.Query().Get("alive")
	if raw == "" {
		return nil
	}
	v, err := strconv.ParseBool(raw)
	if err != nil {
		return nil
	}
	return &v
}

// ProxyProvider handles GET /provider/proxy/{id}
// Returns a proxy-provider YAML built from stored proxies for the subscription.
// Optional query params: ?region=<region>&alive=<bool>
func (h *ProviderHandler) ProxyProvider(w http.ResponseWriter, r *http.Request) {
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
		slog.Error("proxy provider get subscription", "id", id, "err", err)
		writeError(w, http.StatusInternalServerError, "failed to get subscription")
		return
	}
	if sub == nil {
		writeError(w, http.StatusNotFound, "subscription not found")
		return
	}

	filter := store.ProxyFilter{
		SubscriptionID: id,
		Region:         r.URL.Query().Get("region"),
		Alive:          parseAlive(r),
	}
	proxies, err := h.Store.ListProxies(r.Context(), filter)
	if err != nil {
		slog.Error("proxy provider list proxies", "id", id, "err", err)
		writeError(w, http.StatusInternalServerError, "failed to list proxies")
		return
	}

	maps := make([]map[string]interface{}, 0, len(proxies))
	for _, p := range proxies {
		maps = append(maps, proxyToMap(p))
	}

	provider := model.Provider{Proxies: maps}
	out, err := yaml.Marshal(provider)
	if err != nil {
		slog.Error("proxy provider marshal", "id", id, "err", err)
		writeError(w, http.StatusInternalServerError, "failed to marshal provider YAML")
		return
	}

	slog.Info("proxy provider output", "subscription_id", id, "proxy_count", len(maps))
	w.Header().Set("Content-Type", "text/yaml; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write(out)
}

// RuleProvider handles GET /provider/rule/{id}
// Returns a rule-provider YAML built from stored rules for the subscription.
// Optional query params: ?provider=<name>&type=<type>
func (h *ProviderHandler) RuleProvider(w http.ResponseWriter, r *http.Request) {
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
		slog.Error("rule provider get subscription", "id", id, "err", err)
		writeError(w, http.StatusInternalServerError, "failed to get subscription")
		return
	}
	if sub == nil {
		writeError(w, http.StatusNotFound, "subscription not found")
		return
	}

	filter := store.RuleFilter{
		SubscriptionID: id,
		ProviderName:   r.URL.Query().Get("provider"),
		Type:           r.URL.Query().Get("type"),
	}
	rules, err := h.Store.ListRules(r.Context(), filter)
	if err != nil {
		slog.Error("rule provider list rules", "id", id, "err", err)
		writeError(w, http.StatusInternalServerError, "failed to list rules")
		return
	}

	payload := make([]string, 0, len(rules))
	for _, ru := range rules {
		// Skip MATCH rules and rules with empty payload — they have no meaningful payload entry.
		if strings.EqualFold(ru.Type, "MATCH") || ru.Payload == "" {
			continue
		}
		payload = append(payload, ru.Type+","+ru.Payload)
	}

	provider := model.Provider{Payload: payload}
	out, err := yaml.Marshal(provider)
	if err != nil {
		slog.Error("rule provider marshal", "id", id, "err", err)
		writeError(w, http.StatusInternalServerError, "failed to marshal provider YAML")
		return
	}

	slog.Info("rule provider output", "subscription_id", id, "rule_count", len(payload))
	w.Header().Set("Content-Type", "text/yaml; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write(out)
}

// SubscriptionLink handles GET /provider/link/{id}
// Returns a Base64-encoded subscription link string built from stored proxies.
// Optional query params: ?region=<region>&alive=<bool>
func (h *ProviderHandler) SubscriptionLink(w http.ResponseWriter, r *http.Request) {
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
		slog.Error("subscription link get subscription", "id", id, "err", err)
		writeError(w, http.StatusInternalServerError, "failed to get subscription")
		return
	}
	if sub == nil {
		writeError(w, http.StatusNotFound, "subscription not found")
		return
	}

	filter := store.ProxyFilter{
		SubscriptionID: id,
		Region:         r.URL.Query().Get("region"),
		Alive:          parseAlive(r),
	}
	proxies, err := h.Store.ListProxies(r.Context(), filter)
	if err != nil {
		slog.Error("subscription link list proxies", "id", id, "err", err)
		writeError(w, http.StatusInternalServerError, "failed to list proxies")
		return
	}

	maps := make([]map[string]interface{}, 0, len(proxies))
	for _, p := range proxies {
		maps = append(maps, proxyToMap(p))
	}

	link := engine.ProxiesToLinks(maps)
	slog.Info("subscription link output", "subscription_id", id, "proxy_count", len(maps))
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(link))
}
