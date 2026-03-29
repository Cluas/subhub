package main

import (
	"context"
	"crypto/md5"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/spf13/cobra"

	"github.com/Cluas/subhub/internal/api"
	"github.com/Cluas/subhub/internal/config"
	"github.com/Cluas/subhub/internal/engine"
	"github.com/Cluas/subhub/internal/scheduler"
	"github.com/Cluas/subhub/internal/store"
	subweb "github.com/Cluas/subhub/web"
)

const (
	targetTypeRule  = "rule"
	targetTypeProxy = "proxy"
)

// ---- cache ----

type cacheItem struct {
	data      []byte
	timestamp time.Time
}

// boundedCache is a mutex-protected in-memory cache with a configurable entry cap.
// When the cache is full, one arbitrary entry is evicted before inserting a new one.
type boundedCache struct {
	mu         sync.Mutex
	items      map[string]cacheItem
	maxEntries int
}

func newBoundedCache(maxEntries int) *boundedCache {
	return &boundedCache{
		items:      make(map[string]cacheItem),
		maxEntries: maxEntries,
	}
}

func (c *boundedCache) get(key string, ttl time.Duration) ([]byte, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	item, ok := c.items[key]
	if !ok {
		return nil, false
	}
	if time.Since(item.timestamp) > ttl {
		delete(c.items, key)
		return nil, false
	}
	return item.data, true
}

func (c *boundedCache) set(key string, data []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.items) >= c.maxEntries {
		// Evict one arbitrary entry to stay within the cap.
		for k := range c.items {
			delete(c.items, k)
			break
		}
	}
	c.items[key] = cacheItem{data: data, timestamp: time.Now()}
}

func (c *boundedCache) startCleanup(ttl time.Duration) {
	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			now := time.Now()
			c.mu.Lock()
			for k, v := range c.items {
				if now.Sub(v.timestamp) > ttl {
					delete(c.items, k)
				}
			}
			c.mu.Unlock()
		}
	}()
}

func cacheKey(subscribeURL, filter, targetType string) string {
	h := md5.Sum([]byte(fmt.Sprintf("%s:%s:%s", subscribeURL, filter, targetType)))
	return fmt.Sprintf("%x", h)
}

// writeJSONError writes a JSON error response: {"error": msg}.
func writeJSONError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

// ---- metrics ----

type metrics struct {
	totalRequests int64
	cacheHits     int64
	cacheMisses   int64
	errors        int64
	totalDuration time.Duration
	mu            sync.RWMutex
}

var stats metrics

func recordMetrics(cacheHit bool, duration time.Duration, isError bool) {
	stats.mu.Lock()
	defer stats.mu.Unlock()
	stats.totalRequests++
	if cacheHit {
		stats.cacheHits++
	} else {
		stats.cacheMisses++
	}
	if isError {
		stats.errors++
	}
	stats.totalDuration += duration
}

func getMetrics() map[string]interface{} {
	stats.mu.RLock()
	defer stats.mu.RUnlock()
	total := stats.totalRequests
	if total == 0 {
		total = 1
	}
	return map[string]interface{}{
		"total_requests":  stats.totalRequests,
		"cache_hits":      stats.cacheHits,
		"cache_misses":    stats.cacheMisses,
		"errors":          stats.errors,
		"cache_hit_rate":  float64(stats.cacheHits) / float64(total) * 100,
		"avg_duration_ms": stats.totalDuration.Milliseconds() / total,
	}
}

// ---- middleware ----

// bearerAuthMiddleware returns a chi middleware that validates Authorization: Bearer <token>.
// If the configured token is empty, all requests are allowed through (dev mode).
func bearerAuthMiddleware(token string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if token == "" {
				// Dev mode: no token configured, skip auth
				next.ServeHTTP(w, r)
				return
			}
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				json.NewEncoder(w).Encode(map[string]string{"error": "authorization header required"})
				return
			}
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || parts[1] != token {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				json.NewEncoder(w).Encode(map[string]string{"error": "invalid or missing bearer token"})
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// ---- handlers ----

func generateRequestID() string {
	b := make([]byte, 4)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func getClientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		return strings.TrimSpace(strings.Split(xff, ",")[0])
	}
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return strings.TrimSpace(xri)
	}
	ip := r.RemoteAddr
	if i := strings.LastIndex(ip, ":"); i != -1 {
		ip = ip[:i]
	}
	return ip
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	resp := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().Unix(),
		"version":   "1.0.0",
	}
	if r.URL.Query().Get("metrics") == "true" {
		resp["metrics"] = getMetrics()
	}
	json.NewEncoder(w).Encode(resp)
}

func rootHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"service": "SubHub - Clash Subscription Converter",
		"version": "1.0.0",
		"endpoints": map[string]string{
			"health":  "/healthz",
			"convert": "/clash?subscribe_url=<URL>&target_type=rule|proxy&filter=<filter>",
		},
	})
}

func clashHandler(bc *boundedCache, cacheTTL time.Duration) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		reqID := generateRequestID()
		slog.Info("request", "id", reqID, "ip", getClientIP(r), "path", r.URL.Path)

		w.Header().Set("Content-Type", "text/yaml; charset=utf-8")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("X-Request-ID", reqID)

		subscribeURL := r.URL.Query().Get("subscribe_url")
		if subscribeURL == "" {
			writeJSONError(w, http.StatusBadRequest, "subscribe_url parameter is required")
			recordMetrics(false, time.Since(start), true)
			return
		}
		if _, err := url.ParseRequestURI(subscribeURL); err != nil {
			writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("invalid subscribe_url: %v", err))
			recordMetrics(false, time.Since(start), true)
			return
		}

		filter := r.URL.Query().Get("filter")
		targetType := r.URL.Query().Get("target_type")
		if targetType != targetTypeRule && targetType != targetTypeProxy {
			writeJSONError(w, http.StatusBadRequest, "target_type must be 'rule' or 'proxy'")
			recordMetrics(false, time.Since(start), true)
			return
		}

		key := cacheKey(subscribeURL, filter, targetType)
		if cached, ok := bc.get(key, cacheTTL); ok {
			slog.Info("cache hit", "id", reqID)
			w.Write(cached)
			recordMetrics(true, time.Since(start), false)
			return
		}

		var filters []string
		if filter != "" {
			for _, f := range strings.Split(filter, ",") {
				if f = strings.TrimSpace(f); f != "" {
					filters = append(filters, f)
				}
			}
		}

		var (
			result string
			err    error
		)
		switch targetType {
		case targetTypeRule:
			result, err = engine.ConvertToRuleProvider(r.Context(), subscribeURL, filters)
		case targetTypeProxy:
			result, err = engine.ConvertToProxyProvider(r.Context(), subscribeURL, filters)
		}
		if err != nil {
			slog.Error("conversion failed", "id", reqID, "err", err)
			writeJSONError(w, http.StatusInternalServerError, fmt.Sprintf("conversion failed: %v", err))
			recordMetrics(false, time.Since(start), true)
			return
		}

		bc.set(key, []byte(result))
		fmt.Fprint(w, result)

		duration := time.Since(start)
		slog.Info("request done", "id", reqID, "duration", duration)
		recordMetrics(false, duration, false)
	}
}

// newRouter constructs and returns the chi router with all middleware and routes wired.
// Extracted so tests can call it directly without starting a real listener.
// sched may be nil — when nil, the SubscriptionHandler guards all Scheduler calls.
func newRouter(cfg *config.Config, st store.Store, sched *scheduler.Scheduler, bc *boundedCache) chi.Router {
	r := chi.NewRouter()

	// Global middleware stack: CORS → RealIP → Logger → Recoverer
	corsOrigins := []string{cfg.CORSOrigins}
	if cfg.CORSOrigins == "*" {
		corsOrigins = []string{"*"}
	}
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   corsOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: false,
		MaxAge:           300,
	}))
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// Endpoint handler — declared here so both public /p/{slug} and
	// protected /api/endpoints routes can reference the same instance.
	epHandler := &api.EndpointHandler{Store: st}

	// Profile handler — declared here so the public /profile/{id} route and
	// the protected /api/profiles routes share the same instance.
	profHandler := &api.ProfileHandler{Store: st, BaseURL: cfg.BaseURL}

	// Public routes — no auth required
	r.Get("/healthz", healthHandler)
	r.Get("/p/{slug}", epHandler.ServeProvider)
	r.Get("/profile/{id}", profHandler.ServeProfile)

	// Protected routes — require Bearer token
	r.Group(func(r chi.Router) {
		r.Use(bearerAuthMiddleware(cfg.APIToken))
		r.Get("/clash", clashHandler(bc, cfg.CacheTTL))

		// Subscription CRUD + fetch
		subHandler := &api.SubscriptionHandler{Store: st, Scheduler: sched}
		r.Route("/api/subscriptions", func(r chi.Router) {
			r.Get("/", subHandler.List)
			r.Post("/", subHandler.Create)
			r.Route("/{id}", func(r chi.Router) {
				r.Get("/", subHandler.Get)
				r.Put("/", subHandler.Update)
				r.Delete("/", subHandler.Delete)
				r.Post("/fetch", subHandler.Fetch)
				r.Post("/health-check", subHandler.HealthCheck)
			})
		})

		// Provider output endpoints — read from DB, no network access
		provHandler := &api.ProviderHandler{Store: st}
		r.Route("/provider", func(r chi.Router) {
			r.Get("/proxy/{id}", provHandler.ProxyProvider)
			r.Get("/rule/{id}", provHandler.RuleProvider)
			r.Get("/link/{id}", provHandler.SubscriptionLink)
		})

		// Endpoint CRUD + preview
		r.Route("/api/endpoints", func(r chi.Router) {
			r.Get("/", epHandler.List)
			r.Post("/", epHandler.Create)
			r.Route("/{id}", func(r chi.Router) {
				r.Get("/", epHandler.Get)
				r.Put("/", epHandler.Update)
				r.Delete("/", epHandler.Delete)
				r.Get("/preview", epHandler.Preview)
			})
		})

		// Proxies list + self-managed proxy CRUD
		proxiesHandler := &api.ProxiesHandler{Store: st}
		r.Get("/api/proxies", proxiesHandler.List)
		r.Post("/api/proxies", proxiesHandler.Create)
		r.Route("/api/proxies/{id}", func(r chi.Router) {
			r.Get("/", proxiesHandler.Get)
			r.Put("/", proxiesHandler.Update)
			r.Delete("/", proxiesHandler.Delete)
		})

		// Rules list with optional filters + self-managed rule CRUD
		rulesHandler := &api.RulesHandler{Store: st}
		r.Get("/api/rules", rulesHandler.List)
		r.Post("/api/rules", rulesHandler.Create)
		r.Route("/api/rules/{id}", func(r chi.Router) {
			r.Get("/", rulesHandler.Get)
			r.Put("/", rulesHandler.Update)
			r.Delete("/", rulesHandler.Delete)
		})

		// Collection CRUD + nested proxies/rules
		collectionHandler := &api.CollectionHandler{Store: st}
		r.Route("/api/collections", func(r chi.Router) {
			r.Get("/", collectionHandler.List)
			r.Post("/", collectionHandler.Create)
			r.Route("/{id}", func(r chi.Router) {
				r.Get("/", collectionHandler.Get)
				r.Put("/", collectionHandler.Update)
				r.Delete("/", collectionHandler.Delete)
				r.Get("/proxies", collectionHandler.ListProxies)
				r.Get("/rules", collectionHandler.ListRules)
			})
		})

		// Profile CRUD + block management
		r.Route("/api/profiles", func(r chi.Router) {
			r.Get("/", profHandler.List)
			r.Post("/", profHandler.Create)
			r.Route("/{id}", func(r chi.Router) {
				r.Get("/", profHandler.Get)
				r.Put("/", profHandler.Update)
				r.Delete("/", profHandler.Delete)
				r.Route("/node-pools", func(r chi.Router) {
					r.Get("/", profHandler.ListNodePools)
					r.Post("/", profHandler.CreateNodePool)
					r.Put("/{bid}", profHandler.UpdateNodePool)
					r.Delete("/{bid}", profHandler.DeleteNodePool)
				})
				r.Route("/rule-sets", func(r chi.Router) {
					r.Get("/", profHandler.ListRuleSets)
					r.Post("/", profHandler.CreateRuleSet)
					r.Put("/{bid}", profHandler.UpdateRuleSet)
					r.Delete("/{bid}", profHandler.DeleteRuleSet)
				})
				r.Route("/strategies", func(r chi.Router) {
					r.Get("/", profHandler.ListStrategies)
					r.Post("/", profHandler.CreateStrategy)
					r.Put("/{bid}", profHandler.UpdateStrategy)
					r.Delete("/{bid}", profHandler.DeleteStrategy)
				})
				r.Route("/routing-rules", func(r chi.Router) {
					r.Get("/", profHandler.ListRoutingRules)
					r.Post("/", profHandler.CreateRoutingRule)
					r.Put("/{bid}", profHandler.UpdateRoutingRule)
					r.Delete("/{bid}", profHandler.DeleteRoutingRule)
				})
			})
		})

		// Dashboard aggregate stats
		dashHandler := &api.DashboardHandler{Store: st}
		r.Get("/api/dashboard/stats", dashHandler.Stats)

		// Runtime settings (read-only, masked)
		settingsHandler := &api.SettingsHandler{Config: cfg, Store: st}
		r.Get("/api/settings", settingsHandler.Get)
		r.Put("/api/settings", settingsHandler.Update)
	})

	// SPA catch-all: must come AFTER all API/health/clash/provider routes.
	// Any path not matched above is served as the embedded React SPA,
	// with fallback to index.html for client-side routing.
	r.Handle("/*", subweb.NewSPAHandler())

	return r
}

// ---- cobra commands ----

var rootCmd = &cobra.Command{
	Use:   "subhub",
	Short: "Clash subscription manager",
}

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the HTTP server",
	RunE:  runServe,
}

// ---- CLI global flags ----

var (
	serverAddr string
	apiToken   string
	jsonOutput bool
	quietMode  bool
)

// getEnvOr returns the value of env key, or def if unset/empty.
func getEnvOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// apiRequest sends an authenticated HTTP request to the subhub server.
func apiRequest(method, path string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequest(method, serverAddr+path, body)
	if err != nil {
		return nil, err
	}
	if apiToken != "" {
		req.Header.Set("Authorization", "Bearer "+apiToken)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return http.DefaultClient.Do(req)
}

// printTable writes a tab-separated table with headers and rows.
func printTable(headers []string, rows [][]string) {
	fmt.Println(strings.Join(headers, "\t"))
	for _, row := range rows {
		fmt.Println(strings.Join(row, "\t"))
	}
}

// ---- sub commands ----

var subCmd = &cobra.Command{
	Use:   "sub",
	Short: "Manage subscriptions",
}

var (
	subAddName string
	subAddCron string
)

var subAddCmd = &cobra.Command{
	Use:   "add <url>",
	Short: "Add a new subscription",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		subURL := args[0]
		name := subAddName
		if name == "" {
			name = subURL
		}
		payload := map[string]interface{}{"name": name, "url": subURL}
		if subAddCron != "" {
			payload["cron"] = subAddCron
		}
		b, _ := json.Marshal(payload)
		resp, err := apiRequest(http.MethodPost, "/api/subscriptions", strings.NewReader(string(b)))
		if err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		if resp.StatusCode != http.StatusCreated {
			fmt.Fprintf(os.Stderr, "server error %d: %s\n", resp.StatusCode, body)
			if resp.StatusCode == http.StatusNotFound {
				os.Exit(2)
			}
			os.Exit(1)
		}
		if jsonOutput {
			fmt.Print(string(body))
			return nil
		}
		var sub map[string]interface{}
		json.Unmarshal(body, &sub)
		fmt.Printf("created subscription %v: %s\n", sub["id"], name)
		return nil
	},
}

var subListCmd = &cobra.Command{
	Use:   "list",
	Short: "List subscriptions",
	RunE: func(cmd *cobra.Command, args []string) error {
		resp, err := apiRequest(http.MethodGet, "/api/subscriptions", nil)
		if err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		if resp.StatusCode != http.StatusOK {
			fmt.Fprintf(os.Stderr, "server error %d: %s\n", resp.StatusCode, body)
			os.Exit(1)
		}
		if jsonOutput {
			fmt.Print(string(body))
			return nil
		}
		var subs []map[string]interface{}
		json.Unmarshal(body, &subs)
		rows := make([][]string, 0, len(subs))
		for _, s := range subs {
			id := fmt.Sprintf("%v", s["id"])
			name, _ := s["name"].(string)
			rawURL, _ := s["url"].(string)
			status, _ := s["status"].(string)
			nodes := fmt.Sprintf("%v", s["node_count"])
			rows = append(rows, []string{id, name, rawURL, status, nodes})
		}
		printTable([]string{"ID", "Name", "URL", "Status", "Nodes"}, rows)
		return nil
	},
}

var subRemoveCmd = &cobra.Command{
	Use:   "remove <id>",
	Short: "Remove a subscription",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id := args[0]
		if _, err := strconv.ParseInt(id, 10, 64); err != nil {
			fmt.Fprintln(os.Stderr, "invalid id:", id)
			os.Exit(3)
		}
		resp, err := apiRequest(http.MethodDelete, "/api/subscriptions/"+id, nil)
		if err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
		defer resp.Body.Close()
		if resp.StatusCode == http.StatusNotFound {
			fmt.Fprintln(os.Stderr, "not found:", id)
			os.Exit(2)
		}
		if resp.StatusCode != http.StatusNoContent {
			b, _ := io.ReadAll(resp.Body)
			fmt.Fprintf(os.Stderr, "server error %d: %s\n", resp.StatusCode, b)
			os.Exit(1)
		}
		if !quietMode {
			fmt.Printf("deleted subscription %s\n", id)
		}
		return nil
	},
}

var subRefreshCmd = &cobra.Command{
	Use:   "refresh <id>",
	Short: "Re-fetch nodes for a subscription",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id := args[0]
		if _, err := strconv.ParseInt(id, 10, 64); err != nil {
			fmt.Fprintln(os.Stderr, "invalid id:", id)
			os.Exit(3)
		}
		resp, err := apiRequest(http.MethodPost, "/api/subscriptions/"+id+"/fetch", nil)
		if err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		if resp.StatusCode == http.StatusNotFound {
			fmt.Fprintln(os.Stderr, "not found:", id)
			os.Exit(2)
		}
		if resp.StatusCode != http.StatusOK {
			fmt.Fprintf(os.Stderr, "server error %d: %s\n", resp.StatusCode, body)
			os.Exit(1)
		}
		if jsonOutput {
			fmt.Print(string(body))
			return nil
		}
		var sub map[string]interface{}
		json.Unmarshal(body, &sub)
		if !quietMode {
			fmt.Printf("subscription %s refreshed, nodes: %v\n", id, sub["node_count"])
		}
		return nil
	},
}

// ---- node commands ----

var nodeCmd = &cobra.Command{
	Use:   "node",
	Short: "Manage nodes (proxies)",
}

var nodeListSubID string
var nodeListCollectionID string

var nodeListCmd = &cobra.Command{
	Use:   "list",
	Short: "List nodes",
	RunE: func(cmd *cobra.Command, args []string) error {
		path := "/api/proxies"
		sep := "?"
		if nodeListSubID != "" {
			path += sep + "subscription_id=" + nodeListSubID
			sep = "&"
		}
		if nodeListCollectionID != "" {
			path += sep + "collection_id=" + nodeListCollectionID
		}
		resp, err := apiRequest(http.MethodGet, path, nil)
		if err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		if resp.StatusCode != http.StatusOK {
			fmt.Fprintf(os.Stderr, "server error %d: %s\n", resp.StatusCode, body)
			os.Exit(1)
		}
		if jsonOutput {
			fmt.Print(string(body))
			return nil
		}
		var proxies []map[string]interface{}
		json.Unmarshal(body, &proxies)
		rows := make([][]string, 0, len(proxies))
		for _, p := range proxies {
			id := fmt.Sprintf("%v", p["id"])
			name, _ := p["name"].(string)
			ptype, _ := p["type"].(string)
			server, _ := p["server"].(string)
			rows = append(rows, []string{id, name, ptype, server})
		}
		printTable([]string{"ID", "Name", "Type", "Server"}, rows)
		return nil
	},
}

// ---- rule commands ----

var ruleCmd = &cobra.Command{
	Use:   "rule",
	Short: "Manage rules",
}

var ruleListSubID string
var ruleListCollectionID string

var ruleListCmd = &cobra.Command{
	Use:   "list",
	Short: "List rules",
	RunE: func(cmd *cobra.Command, args []string) error {
		path := "/api/rules"
		sep := "?"
		if ruleListSubID != "" {
			path += sep + "subscription_id=" + ruleListSubID
			sep = "&"
		}
		if ruleListCollectionID != "" {
			path += sep + "collection_id=" + ruleListCollectionID
		}
		resp, err := apiRequest(http.MethodGet, path, nil)
		if err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		if resp.StatusCode != http.StatusOK {
			fmt.Fprintf(os.Stderr, "server error %d: %s\n", resp.StatusCode, body)
			os.Exit(1)
		}
		if jsonOutput {
			fmt.Print(string(body))
			return nil
		}
		var rules []map[string]interface{}
		json.Unmarshal(body, &rules)
		rows := make([][]string, 0, len(rules))
		for _, r := range rules {
			id := fmt.Sprintf("%v", r["id"])
			rtype, _ := r["type"].(string)
			payload, _ := r["payload"].(string)
			target, _ := r["target"].(string)
			rows = append(rows, []string{id, rtype, payload, target})
		}
		printTable([]string{"ID", "Type", "Payload", "Target"}, rows)
		return nil
	},
}

// ---- node add command ----

var (
	nodeAddCollectionID string
	nodeAddName         string
	nodeAddType         string
	nodeAddServer       string
	nodeAddPort         string
	nodeAddRegion       string
)

var nodeAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a node (proxy)",
	RunE: func(cmd *cobra.Command, args []string) error {
		if nodeAddName == "" {
			fmt.Fprintln(os.Stderr, "--name is required")
			os.Exit(3)
		}
		if nodeAddType == "" {
			fmt.Fprintln(os.Stderr, "--type is required")
			os.Exit(3)
		}
		if nodeAddServer == "" {
			fmt.Fprintln(os.Stderr, "--server is required")
			os.Exit(3)
		}
		if nodeAddPort == "" {
			fmt.Fprintln(os.Stderr, "--port is required")
			os.Exit(3)
		}
		port, err := strconv.Atoi(nodeAddPort)
		if err != nil || port <= 0 || port > 65535 {
			fmt.Fprintln(os.Stderr, "invalid --port:", nodeAddPort)
			os.Exit(3)
		}
		payload := map[string]interface{}{
			"name":   nodeAddName,
			"type":   nodeAddType,
			"server": nodeAddServer,
			"port":   port,
		}
		if nodeAddRegion != "" {
			payload["region"] = nodeAddRegion
		}
		if nodeAddCollectionID != "" {
			cid, err := strconv.ParseInt(nodeAddCollectionID, 10, 64)
			if err != nil {
				fmt.Fprintln(os.Stderr, "invalid --collection:", nodeAddCollectionID)
				os.Exit(3)
			}
			payload["collection_id"] = cid
		}
		b, _ := json.Marshal(payload)
		resp, err := apiRequest(http.MethodPost, "/api/proxies", strings.NewReader(string(b)))
		if err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		if resp.StatusCode != http.StatusCreated {
			fmt.Fprintf(os.Stderr, "server error %d: %s\n", resp.StatusCode, body)
			os.Exit(1)
		}
		if jsonOutput {
			fmt.Print(string(body))
			return nil
		}
		var p map[string]interface{}
		json.Unmarshal(body, &p)
		if !quietMode {
			fmt.Printf("created node %v (name: %v)\n", p["id"], p["name"])
		}
		return nil
	},
}

// ---- rule add command ----

var (
	ruleAddCollectionID string
	ruleAddType         string
	ruleAddPayload      string
	ruleAddTarget       string
	ruleAddProvider     string
)

var ruleAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a rule",
	RunE: func(cmd *cobra.Command, args []string) error {
		if ruleAddType == "" {
			fmt.Fprintln(os.Stderr, "--type is required")
			os.Exit(3)
		}
		if ruleAddTarget == "" {
			fmt.Fprintln(os.Stderr, "--target is required")
			os.Exit(3)
		}
		payload := map[string]interface{}{
			"type":    ruleAddType,
			"payload": ruleAddPayload,
			"target":  ruleAddTarget,
		}
		if ruleAddProvider != "" {
			payload["provider_name"] = ruleAddProvider
		}
		if ruleAddCollectionID != "" {
			cid, err := strconv.ParseInt(ruleAddCollectionID, 10, 64)
			if err != nil {
				fmt.Fprintln(os.Stderr, "invalid --collection:", ruleAddCollectionID)
				os.Exit(3)
			}
			payload["collection_id"] = cid
		}
		b, _ := json.Marshal(payload)
		resp, err := apiRequest(http.MethodPost, "/api/rules", strings.NewReader(string(b)))
		if err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		if resp.StatusCode != http.StatusCreated {
			fmt.Fprintf(os.Stderr, "server error %d: %s\n", resp.StatusCode, body)
			os.Exit(1)
		}
		if jsonOutput {
			fmt.Print(string(body))
			return nil
		}
		var r map[string]interface{}
		json.Unmarshal(body, &r)
		if !quietMode {
			fmt.Printf("created rule %v (%v %v → %v)\n", r["id"], r["type"], r["payload"], r["target"])
		}
		return nil
	},
}

// ---- collection commands ----

var collectionCmd = &cobra.Command{
	Use:   "collection",
	Short: "Manage collections (local node/rule groups)",
}

var (
	collectionCreateName        string
	collectionCreateType        string
	collectionCreateDescription string
)

var collectionCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new collection",
	RunE: func(cmd *cobra.Command, args []string) error {
		if collectionCreateName == "" {
			fmt.Fprintln(os.Stderr, "--name is required")
			os.Exit(3)
		}
		if collectionCreateType == "" {
			fmt.Fprintln(os.Stderr, "--type is required (proxy|rule)")
			os.Exit(3)
		}
		if collectionCreateType != "proxy" && collectionCreateType != "rule" {
			fmt.Fprintln(os.Stderr, "--type must be 'proxy' or 'rule'")
			os.Exit(3)
		}
		payload := map[string]interface{}{
			"name":         collectionCreateName,
			"content_type": collectionCreateType,
			"description":  collectionCreateDescription,
		}
		b, _ := json.Marshal(payload)
		resp, err := apiRequest(http.MethodPost, "/api/collections", strings.NewReader(string(b)))
		if err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		if resp.StatusCode != http.StatusCreated {
			fmt.Fprintf(os.Stderr, "server error %d: %s\n", resp.StatusCode, body)
			os.Exit(1)
		}
		if jsonOutput {
			fmt.Print(string(body))
			return nil
		}
		var c map[string]interface{}
		json.Unmarshal(body, &c)
		if !quietMode {
			fmt.Printf("created collection %v (name: %v, type: %v)\n", c["id"], c["name"], c["content_type"])
		}
		return nil
	},
}

var collectionListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all collections",
	RunE: func(cmd *cobra.Command, args []string) error {
		resp, err := apiRequest(http.MethodGet, "/api/collections", nil)
		if err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		if resp.StatusCode != http.StatusOK {
			fmt.Fprintf(os.Stderr, "server error %d: %s\n", resp.StatusCode, body)
			os.Exit(1)
		}
		if jsonOutput {
			fmt.Print(string(body))
			return nil
		}
		var cols []map[string]interface{}
		json.Unmarshal(body, &cols)
		rows := make([][]string, 0, len(cols))
		for _, c := range cols {
			id := fmt.Sprintf("%v", c["id"])
			name, _ := c["name"].(string)
			ctype, _ := c["content_type"].(string)
			desc, _ := c["description"].(string)
			rows = append(rows, []string{id, name, ctype, desc})
		}
		printTable([]string{"ID", "Name", "Type", "Description"}, rows)
		return nil
	},
}

var collectionRemoveCmd = &cobra.Command{
	Use:   "remove <id>",
	Short: "Delete a collection (and all its nodes/rules)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id := args[0]
		if _, err := strconv.ParseInt(id, 10, 64); err != nil {
			fmt.Fprintln(os.Stderr, "invalid id:", id)
			os.Exit(3)
		}
		resp, err := apiRequest(http.MethodDelete, "/api/collections/"+id, nil)
		if err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
		defer resp.Body.Close()
		if resp.StatusCode == http.StatusNotFound {
			fmt.Fprintln(os.Stderr, "not found:", id)
			os.Exit(2)
		}
		if resp.StatusCode != http.StatusNoContent {
			b, _ := io.ReadAll(resp.Body)
			fmt.Fprintf(os.Stderr, "server error %d: %s\n", resp.StatusCode, b)
			os.Exit(1)
		}
		if !quietMode {
			fmt.Printf("deleted collection %s\n", id)
		}
		return nil
	},
}

// ---- endpoint commands ----

var endpointCmd = &cobra.Command{
	Use:   "endpoint",
	Short: "Manage output endpoints",
}

var endpointListCmd = &cobra.Command{
	Use:   "list",
	Short: "List endpoints",
	RunE: func(cmd *cobra.Command, args []string) error {
		resp, err := apiRequest(http.MethodGet, "/api/endpoints", nil)
		if err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		if resp.StatusCode != http.StatusOK {
			fmt.Fprintf(os.Stderr, "server error %d: %s\n", resp.StatusCode, body)
			os.Exit(1)
		}
		if jsonOutput {
			fmt.Print(string(body))
			return nil
		}
		var eps []map[string]interface{}
		json.Unmarshal(body, &eps)
		rows := make([][]string, 0, len(eps))
		for _, ep := range eps {
			id := fmt.Sprintf("%v", ep["id"])
			name, _ := ep["name"].(string)
			format, _ := ep["format"].(string)
			slug, _ := ep["slug"].(string)
			rows = append(rows, []string{id, name, format, slug})
		}
		printTable([]string{"ID", "Name", "Format", "Slug"}, rows)
		return nil
	},
}

var (
	endpointCreateSubID        string
	endpointCreateCollectionID string
	endpointCreateFormat       string
	endpointCreateName         string
	endpointCreateOutputType   string
)

var endpointCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create an endpoint",
	RunE: func(cmd *cobra.Command, args []string) error {
		if endpointCreateFormat == "" {
			fmt.Fprintln(os.Stderr, "--format is required")
			os.Exit(3)
		}
		// At least one source must be provided (or none for all-sources endpoint).
		name := endpointCreateName
		if name == "" {
			name = endpointCreateFormat + "-endpoint"
		}
		outputType := endpointCreateOutputType
		if outputType == "" {
			outputType = "proxy"
		}
		payload := map[string]interface{}{
			"name":        name,
			"output_type": outputType,
			"format":      endpointCreateFormat,
			"filters":     map[string]interface{}{},
		}
		if endpointCreateSubID != "" {
			subIDInt, err := strconv.ParseInt(endpointCreateSubID, 10, 64)
			if err != nil {
				fmt.Fprintln(os.Stderr, "invalid --sub:", endpointCreateSubID)
				os.Exit(3)
			}
			payload["subscription_id"] = subIDInt
		}
		if endpointCreateCollectionID != "" {
			cid, err := strconv.ParseInt(endpointCreateCollectionID, 10, 64)
			if err != nil {
				fmt.Fprintln(os.Stderr, "invalid --collection:", endpointCreateCollectionID)
				os.Exit(3)
			}
			payload["collection_id"] = cid
		}
		b, _ := json.Marshal(payload)
		resp, err := apiRequest(http.MethodPost, "/api/endpoints", strings.NewReader(string(b)))
		if err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		if resp.StatusCode != http.StatusCreated {
			fmt.Fprintf(os.Stderr, "server error %d: %s\n", resp.StatusCode, body)
			if resp.StatusCode == http.StatusNotFound {
				os.Exit(2)
			}
			os.Exit(1)
		}
		if jsonOutput {
			fmt.Print(string(body))
			return nil
		}
		var ep map[string]interface{}
		json.Unmarshal(body, &ep)
		if !quietMode {
			fmt.Printf("created endpoint %v (slug: %v)\n", ep["id"], ep["slug"])
		}
		return nil
	},
}

var endpointRemoveCmd = &cobra.Command{
	Use:   "remove <id>",
	Short: "Remove an endpoint",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id := args[0]
		if _, err := strconv.ParseInt(id, 10, 64); err != nil {
			fmt.Fprintln(os.Stderr, "invalid id:", id)
			os.Exit(3)
		}
		resp, err := apiRequest(http.MethodDelete, "/api/endpoints/"+id, nil)
		if err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
		defer resp.Body.Close()
		if resp.StatusCode == http.StatusNotFound {
			fmt.Fprintln(os.Stderr, "not found:", id)
			os.Exit(2)
		}
		if resp.StatusCode != http.StatusNoContent {
			b, _ := io.ReadAll(resp.Body)
			fmt.Fprintf(os.Stderr, "server error %d: %s\n", resp.StatusCode, b)
			os.Exit(1)
		}
		if !quietMode {
			fmt.Printf("deleted endpoint %s\n", id)
		}
		return nil
	},
}

// ---- convert command (offline, no server required) ----

var (
	convertFormat string
	convertType   string
	convertFilter string
)

var convertCmd = &cobra.Command{
	Use:   "convert <url>",
	Short: "Convert a subscription URL to a given format (offline, no server required)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		subURL := args[0]
		if convertFormat == "" {
			fmt.Fprintln(os.Stderr, "--format is required")
			os.Exit(3)
		}
		var filters []string
		if convertFilter != "" {
			for _, f := range strings.Split(convertFilter, ",") {
				if f = strings.TrimSpace(f); f != "" {
					filters = append(filters, f)
				}
			}
		}
		ctx := context.Background()
		switch strings.ToLower(convertType) {
		case "rule":
			result, err := engine.ConvertToRuleProvider(ctx, subURL, filters)
			if err != nil {
				fmt.Fprintln(os.Stderr, "conversion failed:", err)
				os.Exit(1)
			}
			// ConvertToRuleProvider returns Clash YAML; for non-clash formats
			// we would need to parse and re-format — for now output as-is.
			fmt.Print(result)
		default: // "proxy"
			result, err := engine.ConvertToProxyProvider(ctx, subURL, filters)
			if err != nil {
				fmt.Fprintln(os.Stderr, "conversion failed:", err)
				os.Exit(1)
			}
			if strings.ToLower(convertFormat) == "clash" {
				fmt.Print(result)
				return nil
			}
			// Parse the provider YAML into proxy maps, then re-format.
			var provider struct {
				Proxies []map[string]interface{} `json:"proxies"`
			}
			// sigs.k8s.io/yaml converts YAML→JSON first, so we marshal via engine helpers.
			// Use a simple hand-parse: unmarshal with encoding/json after yaml→json conversion
			// is not available here directly; call engine.ConvertToProxyProvider which
			// returns a YAML string. We re-fetch via a proxy-list parse helper.
			// Since we can't import sigs.k8s.io/yaml directly without a new import,
			// and the plan constraint says no new third-party imports, we pipe through
			// the clash engine format result which is Clash YAML.
			// For non-clash formats we rely on the fact that engine.ConvertToProxyProvider
			// returns a provider YAML that can be parsed generically.
			// Practical approach: use engine.MergeClashConfig + FormatProxyOutput directly.
			merged, err := engine.MergeClashConfig(ctx, subURL)
			if err != nil {
				fmt.Fprintln(os.Stderr, "fetch failed:", err)
				os.Exit(1)
			}
			proxyMaps := merged.Proxies
			if len(filters) > 0 {
				// Apply name filter.
				filtered := make([]map[string]interface{}, 0)
				for _, p := range proxyMaps {
					name, _ := p["name"].(string)
					for _, f := range filters {
						if strings.Contains(name, f) {
							filtered = append(filtered, p)
							break
						}
					}
				}
				proxyMaps = filtered
			}
			_ = provider // silence unused var
			out, _, err := engine.FormatProxyOutput(convertFormat, proxyMaps)
			if err != nil {
				fmt.Fprintln(os.Stderr, "format error:", err)
				os.Exit(1)
			}
			fmt.Print(string(out))
		}
		return nil
	},
}

// ---- profile commands ----

var profileCmd = &cobra.Command{
	Use:   "profile",
	Short: "Manage profiles",
}

var profileCreateName string

var profileCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a profile",
	RunE: func(cmd *cobra.Command, args []string) error {
		if profileCreateName == "" {
			fmt.Fprintln(os.Stderr, "--name is required")
			os.Exit(3)
		}
		b, _ := json.Marshal(map[string]interface{}{"name": profileCreateName})
		resp, err := apiRequest(http.MethodPost, "/api/profiles", strings.NewReader(string(b)))
		if err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		if resp.StatusCode != http.StatusCreated {
			fmt.Fprintf(os.Stderr, "server error %d: %s\n", resp.StatusCode, body)
			if resp.StatusCode == http.StatusNotFound {
				os.Exit(2)
			}
			os.Exit(1)
		}
		if jsonOutput {
			fmt.Print(string(body))
			return nil
		}
		var p map[string]interface{}
		json.Unmarshal(body, &p)
		if !quietMode {
			fmt.Printf("created profile %v\n", p["id"])
		}
		return nil
	},
}

var profileListCmd = &cobra.Command{
	Use:   "list",
	Short: "List profiles",
	RunE: func(cmd *cobra.Command, args []string) error {
		resp, err := apiRequest(http.MethodGet, "/api/profiles", nil)
		if err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		if resp.StatusCode != http.StatusOK {
			fmt.Fprintf(os.Stderr, "server error %d: %s\n", resp.StatusCode, body)
			os.Exit(1)
		}
		if jsonOutput {
			fmt.Print(string(body))
			return nil
		}
		var profiles []map[string]interface{}
		json.Unmarshal(body, &profiles)
		rows := make([][]string, 0, len(profiles))
		for _, p := range profiles {
			id := fmt.Sprintf("%v", p["id"])
			name, _ := p["name"].(string)
			slug, _ := p["slug"].(string)
			rows = append(rows, []string{id, name, slug})
		}
		printTable([]string{"ID", "Name", "Slug"}, rows)
		return nil
	},
}

var profileGetCmd = &cobra.Command{
	Use:   "get <id>",
	Short: "Output Clash YAML for a profile",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id := args[0]
		if _, err := strconv.ParseInt(id, 10, 64); err != nil {
			fmt.Fprintln(os.Stderr, "invalid id:", id)
			os.Exit(3)
		}
		// Step 1: fetch profile JSON to get the slug.
		resp, err := apiRequest(http.MethodGet, "/api/profiles/"+id, nil)
		if err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		if resp.StatusCode == http.StatusNotFound {
			fmt.Fprintln(os.Stderr, "not found:", id)
			os.Exit(2)
		}
		if resp.StatusCode != http.StatusOK {
			fmt.Fprintf(os.Stderr, "server error %d: %s\n", resp.StatusCode, body)
			os.Exit(1)
		}
		if jsonOutput {
			fmt.Print(string(body))
			return nil
		}
		var p map[string]interface{}
		json.Unmarshal(body, &p)
		slug, _ := p["slug"].(string)
		// Step 2: fetch Clash YAML via the public render endpoint.
		yamlResp, err := apiRequest(http.MethodGet, "/profile/"+slug, nil)
		if err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
		defer yamlResp.Body.Close()
		yamlBody, _ := io.ReadAll(yamlResp.Body)
		if yamlResp.StatusCode != http.StatusOK {
			fmt.Fprintf(os.Stderr, "server error %d: %s\n", yamlResp.StatusCode, yamlBody)
			os.Exit(1)
		}
		fmt.Print(string(yamlBody))
		return nil
	},
}

var profileRemoveCmd = &cobra.Command{
	Use:   "remove <id>",
	Short: "Remove a profile",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id := args[0]
		if _, err := strconv.ParseInt(id, 10, 64); err != nil {
			fmt.Fprintln(os.Stderr, "invalid id:", id)
			os.Exit(3)
		}
		resp, err := apiRequest(http.MethodDelete, "/api/profiles/"+id, nil)
		if err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
		defer resp.Body.Close()
		if resp.StatusCode == http.StatusNotFound {
			fmt.Fprintln(os.Stderr, "not found:", id)
			os.Exit(2)
		}
		if resp.StatusCode != http.StatusNoContent {
			b, _ := io.ReadAll(resp.Body)
			fmt.Fprintf(os.Stderr, "server error %d: %s\n", resp.StatusCode, b)
			os.Exit(1)
		}
		if !quietMode {
			fmt.Printf("deleted profile %s\n", id)
		}
		return nil
	},
}

// ─── profile add-pool ─────────────────────────────────────────────────────────

var (
	profileAddPoolProfileID  int64
	profileAddPoolName       string
	profileAddPoolSlug       string
	profileAddPoolHCURL      string
	profileAddPoolInterval   int
)

var profileAddPoolCmd = &cobra.Command{
	Use:   "add-pool",
	Short: "Add a node pool to a profile",
	RunE: func(cmd *cobra.Command, args []string) error {
		if profileAddPoolProfileID == 0 {
			fmt.Fprintln(os.Stderr, "--profile is required")
			os.Exit(3)
		}
		if profileAddPoolName == "" {
			fmt.Fprintln(os.Stderr, "--name is required")
			os.Exit(3)
		}
		b, _ := json.Marshal(map[string]interface{}{
			"name":             profileAddPoolName,
			"endpoint_slug":    profileAddPoolSlug,
			"health_check_url": profileAddPoolHCURL,
			"interval":         profileAddPoolInterval,
		})
		path := fmt.Sprintf("/api/profiles/%d/node-pools", profileAddPoolProfileID)
		resp, err := apiRequest(http.MethodPost, path, strings.NewReader(string(b)))
		if err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		if resp.StatusCode != http.StatusCreated {
			fmt.Fprintf(os.Stderr, "server error %d: %s\n", resp.StatusCode, body)
			os.Exit(1)
		}
		var np map[string]interface{}
		json.Unmarshal(body, &np)
		if !quietMode {
			fmt.Printf("added node pool %v\n", np["id"])
		}
		return nil
	},
}

// ─── profile add-ruleset ──────────────────────────────────────────────────────

var (
	profileAddRuleSetProfileID int64
	profileAddRuleSetName      string
	profileAddRuleSetSlug      string
	profileAddRuleSetURL       string
	profileAddRuleSetBehavior  string
	profileAddRuleSetFormat    string
)

var profileAddRuleSetCmd = &cobra.Command{
	Use:   "add-ruleset",
	Short: "Add a rule set to a profile",
	RunE: func(cmd *cobra.Command, args []string) error {
		if profileAddRuleSetProfileID == 0 {
			fmt.Fprintln(os.Stderr, "--profile is required")
			os.Exit(3)
		}
		if profileAddRuleSetName == "" {
			fmt.Fprintln(os.Stderr, "--name is required")
			os.Exit(3)
		}
		metadata := map[string]interface{}{
			"behavior": profileAddRuleSetBehavior,
			"format":   profileAddRuleSetFormat,
		}
		b, _ := json.Marshal(map[string]interface{}{
			"name":          profileAddRuleSetName,
			"endpoint_slug": profileAddRuleSetSlug,
			"url":           profileAddRuleSetURL,
			"metadata":      metadata,
		})
		path := fmt.Sprintf("/api/profiles/%d/rule-sets", profileAddRuleSetProfileID)
		resp, err := apiRequest(http.MethodPost, path, strings.NewReader(string(b)))
		if err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		if resp.StatusCode != http.StatusCreated {
			fmt.Fprintf(os.Stderr, "server error %d: %s\n", resp.StatusCode, body)
			os.Exit(1)
		}
		var rs map[string]interface{}
		json.Unmarshal(body, &rs)
		if !quietMode {
			fmt.Printf("added rule set %v\n", rs["id"])
		}
		return nil
	},
}

// ─── profile add-strategy ────────────────────────────────────────────────────

var (
	profileAddStrategyProfileID int64
	profileAddStrategyName      string
	profileAddStrategyStrategy  string
	profileAddStrategyPools     []string
	profileAddStrategyProxies   []string
)

var profileAddStrategyCmd = &cobra.Command{
	Use:   "add-strategy",
	Short: "Add a proxy group strategy to a profile",
	RunE: func(cmd *cobra.Command, args []string) error {
		if profileAddStrategyProfileID == 0 {
			fmt.Fprintln(os.Stderr, "--profile is required")
			os.Exit(3)
		}
		if profileAddStrategyName == "" {
			fmt.Fprintln(os.Stderr, "--name is required")
			os.Exit(3)
		}
		b, _ := json.Marshal(map[string]interface{}{
			"name":     profileAddStrategyName,
			"strategy": profileAddStrategyStrategy,
			"pools":    profileAddStrategyPools,
			"proxies":  profileAddStrategyProxies,
		})
		path := fmt.Sprintf("/api/profiles/%d/strategies", profileAddStrategyProfileID)
		resp, err := apiRequest(http.MethodPost, path, strings.NewReader(string(b)))
		if err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		if resp.StatusCode != http.StatusCreated {
			fmt.Fprintf(os.Stderr, "server error %d: %s\n", resp.StatusCode, body)
			os.Exit(1)
		}
		var st map[string]interface{}
		json.Unmarshal(body, &st)
		if !quietMode {
			fmt.Printf("added strategy %v\n", st["id"])
		}
		return nil
	},
}

// ─── profile add-routing-rule ─────────────────────────────────────────────────

var (
	profileAddRRProfileID int64
	profileAddRRMatch     string
	profileAddRRValue     string
	profileAddRRTarget    string
	profileAddRRPosition  int
	profileAddRRNoResolve bool
)

var profileAddRoutingRuleCmd = &cobra.Command{
	Use:   "add-routing-rule",
	Short: "Add a routing rule to a profile",
	RunE: func(cmd *cobra.Command, args []string) error {
		if profileAddRRProfileID == 0 {
			fmt.Fprintln(os.Stderr, "--profile is required")
			os.Exit(3)
		}
		if profileAddRRMatch == "" {
			fmt.Fprintln(os.Stderr, "--match is required")
			os.Exit(3)
		}
		if profileAddRRTarget == "" {
			fmt.Fprintln(os.Stderr, "--target is required")
			os.Exit(3)
		}
		b, _ := json.Marshal(map[string]interface{}{
			"match":      profileAddRRMatch,
			"value":      profileAddRRValue,
			"target":     profileAddRRTarget,
			"position":   profileAddRRPosition,
			"no_resolve": profileAddRRNoResolve,
		})
		path := fmt.Sprintf("/api/profiles/%d/routing-rules", profileAddRRProfileID)
		resp, err := apiRequest(http.MethodPost, path, strings.NewReader(string(b)))
		if err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		if resp.StatusCode != http.StatusCreated {
			fmt.Fprintf(os.Stderr, "server error %d: %s\n", resp.StatusCode, body)
			os.Exit(1)
		}
		var rr map[string]interface{}
		json.Unmarshal(body, &rr)
		if !quietMode {
			fmt.Printf("added routing rule %v\n", rr["id"])
		}
		return nil
	},
}

func init() {
	// Global persistent flags
	rootCmd.PersistentFlags().StringVar(&serverAddr, "server", getEnvOr("SUBHUB_SERVER", "http://localhost:9000"), "server address")
	rootCmd.PersistentFlags().StringVar(&apiToken, "token", os.Getenv("SUBHUB_TOKEN"), "API token")
	rootCmd.PersistentFlags().BoolVar(&jsonOutput, "json", false, "output raw JSON")
	rootCmd.PersistentFlags().BoolVar(&quietMode, "quiet", false, "suppress non-error output")

	// sub add flags
	subAddCmd.Flags().StringVar(&subAddName, "name", "", "subscription name")
	subAddCmd.Flags().StringVar(&subAddCron, "cron", "", "cron expression for auto-refresh")

	// node list flags
	nodeListCmd.Flags().StringVar(&nodeListSubID, "sub", "", "filter by subscription ID")
	nodeListCmd.Flags().StringVar(&nodeListCollectionID, "collection", "", "filter by collection ID")

	// node add flags
	nodeAddCmd.Flags().StringVar(&nodeAddCollectionID, "collection", "", "collection ID to add node to")
	nodeAddCmd.Flags().StringVar(&nodeAddName, "name", "", "node name (required)")
	nodeAddCmd.Flags().StringVar(&nodeAddType, "type", "", "protocol type: ss/vmess/vless/trojan (required)")
	nodeAddCmd.Flags().StringVar(&nodeAddServer, "server", "", "server hostname or IP (required)")
	nodeAddCmd.Flags().StringVar(&nodeAddPort, "port", "", "server port (required)")
	nodeAddCmd.Flags().StringVar(&nodeAddRegion, "region", "", "region tag")

	// rule list flags
	ruleListCmd.Flags().StringVar(&ruleListSubID, "sub", "", "filter by subscription ID")
	ruleListCmd.Flags().StringVar(&ruleListCollectionID, "collection", "", "filter by collection ID")

	// rule add flags
	ruleAddCmd.Flags().StringVar(&ruleAddCollectionID, "collection", "", "collection ID to add rule to")
	ruleAddCmd.Flags().StringVar(&ruleAddType, "type", "", "rule type (required)")
	ruleAddCmd.Flags().StringVar(&ruleAddPayload, "payload", "", "rule payload")
	ruleAddCmd.Flags().StringVar(&ruleAddTarget, "target", "", "target proxy group (required)")
	ruleAddCmd.Flags().StringVar(&ruleAddProvider, "provider", "", "provider name")

	// endpoint create flags
	endpointCreateCmd.Flags().StringVar(&endpointCreateSubID, "sub", "", "subscription ID (optional)")
	endpointCreateCmd.Flags().StringVar(&endpointCreateCollectionID, "collection", "", "collection ID (optional)")
	endpointCreateCmd.Flags().StringVar(&endpointCreateFormat, "format", "", "output format (required)")
	endpointCreateCmd.Flags().StringVar(&endpointCreateName, "name", "", "endpoint name")
	endpointCreateCmd.Flags().StringVar(&endpointCreateOutputType, "type", "proxy", "output type: proxy|rule")

	// collection flags
	collectionCreateCmd.Flags().StringVar(&collectionCreateName, "name", "", "collection name (required)")
	collectionCreateCmd.Flags().StringVar(&collectionCreateType, "type", "", "content type: proxy|rule (required)")
	collectionCreateCmd.Flags().StringVar(&collectionCreateDescription, "description", "", "optional description")

	// convert flags
	convertCmd.Flags().StringVar(&convertFormat, "format", "", "output format: clash, surge, quantumultx, loon, singbox (required)")
	convertCmd.Flags().StringVar(&convertType, "type", "proxy", "conversion type: proxy or rule")
	convertCmd.Flags().StringVar(&convertFilter, "filter", "", "comma-separated keyword filters")

	// profile create flags
	profileCreateCmd.Flags().StringVar(&profileCreateName, "name", "", "profile name (required)")

	// profile add-pool flags
	profileAddPoolCmd.Flags().Int64Var(&profileAddPoolProfileID, "profile", 0, "profile ID (required)")
	profileAddPoolCmd.Flags().StringVar(&profileAddPoolName, "name", "", "node pool name (required)")
	profileAddPoolCmd.Flags().StringVar(&profileAddPoolSlug, "endpoint-slug", "", "endpoint slug")
	profileAddPoolCmd.Flags().StringVar(&profileAddPoolHCURL, "health-check-url", "http://www.gstatic.com/generate_204", "health check URL")
	profileAddPoolCmd.Flags().IntVar(&profileAddPoolInterval, "interval", 300, "health check interval in seconds")

	// profile add-ruleset flags
	profileAddRuleSetCmd.Flags().Int64Var(&profileAddRuleSetProfileID, "profile", 0, "profile ID (required)")
	profileAddRuleSetCmd.Flags().StringVar(&profileAddRuleSetName, "name", "", "rule set name (required)")
	profileAddRuleSetCmd.Flags().StringVar(&profileAddRuleSetSlug, "endpoint-slug", "", "SubHub endpoint slug")
	profileAddRuleSetCmd.Flags().StringVar(&profileAddRuleSetURL, "url", "", "external rule set URL")
	profileAddRuleSetCmd.Flags().StringVar(&profileAddRuleSetBehavior, "behavior", "classical", "rule behavior: classical|domain|ipcidr")
	profileAddRuleSetCmd.Flags().StringVar(&profileAddRuleSetFormat, "format", "yaml", "rule format: yaml|text")

	// profile add-strategy flags
	profileAddStrategyCmd.Flags().Int64Var(&profileAddStrategyProfileID, "profile", 0, "profile ID (required)")
	profileAddStrategyCmd.Flags().StringVar(&profileAddStrategyName, "name", "", "strategy name (required)")
	profileAddStrategyCmd.Flags().StringVar(&profileAddStrategyStrategy, "strategy", "select", "strategy type: select|auto|load_balance|fallback")
	profileAddStrategyCmd.Flags().StringArrayVar(&profileAddStrategyPools, "pool", nil, "node pool name (repeatable)")
	profileAddStrategyCmd.Flags().StringArrayVar(&profileAddStrategyProxies, "proxy", nil, "proxy name (repeatable)")

	// profile add-routing-rule flags
	profileAddRoutingRuleCmd.Flags().Int64Var(&profileAddRRProfileID, "profile", 0, "profile ID (required)")
	profileAddRoutingRuleCmd.Flags().StringVar(&profileAddRRMatch, "match", "", "rule match type, e.g. DOMAIN-SUFFIX (required)")
	profileAddRoutingRuleCmd.Flags().StringVar(&profileAddRRValue, "value", "", "match value")
	profileAddRoutingRuleCmd.Flags().StringVar(&profileAddRRTarget, "target", "", "target proxy group (required)")
	profileAddRoutingRuleCmd.Flags().IntVar(&profileAddRRPosition, "position", 0, "rule position (lower = higher priority)")
	profileAddRoutingRuleCmd.Flags().BoolVar(&profileAddRRNoResolve, "no-resolve", false, "append no-resolve flag")

	// Wire all subcommands
	rootCmd.AddCommand(serveCmd)
	rootCmd.AddCommand(subCmd)
	rootCmd.AddCommand(nodeCmd)
	rootCmd.AddCommand(ruleCmd)
	rootCmd.AddCommand(endpointCmd)
	rootCmd.AddCommand(convertCmd)
	rootCmd.AddCommand(profileCmd)
	rootCmd.AddCommand(collectionCmd)
	subCmd.AddCommand(subAddCmd, subListCmd, subRemoveCmd, subRefreshCmd)
	nodeCmd.AddCommand(nodeListCmd, nodeAddCmd)
	ruleCmd.AddCommand(ruleListCmd, ruleAddCmd)
	endpointCmd.AddCommand(endpointListCmd, endpointCreateCmd, endpointRemoveCmd)
	profileCmd.AddCommand(profileCreateCmd, profileListCmd, profileGetCmd, profileRemoveCmd,
		profileAddPoolCmd, profileAddRuleSetCmd, profileAddStrategyCmd, profileAddRoutingRuleCmd)
	collectionCmd.AddCommand(collectionCreateCmd, collectionListCmd, collectionRemoveCmd)
}

func runServe(cmd *cobra.Command, args []string) error {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))

	cfg := config.Load()
	slog.Info("starting SubHub", "port", cfg.Port, "db", cfg.DBPath)

	st, err := store.NewSQLite(cfg.DBPath)
	if err != nil {
		slog.Error("failed to open database", "err", err)
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer st.Close()

	bc := newBoundedCache(cfg.CacheMaxEntries)
	bc.startCleanup(cfg.CacheTTL)

	// Construct and start the cron scheduler.
	sched := scheduler.New(st)
	sched.Start()

	// Sync all existing subscriptions at startup.
	startupSubs, err := st.ListSubscriptions(context.Background())
	if err != nil {
		slog.Warn("failed to load subscriptions for scheduler startup sync", "err", err)
	} else {
		sched.Sync(context.Background(), startupSubs)
	}

	r := newRouter(cfg, st, sched, bc)

	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      r,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		slog.Info("server listening", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "err", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("shutting down...")
	// Stop the cron scheduler first so in-flight cron jobs drain before the
	// HTTP server closes its listener.
	sched.Stop()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	srv.Shutdown(ctx)
	return nil
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
