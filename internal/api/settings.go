package api

import (
	"encoding/json"
	"net/http"

	"github.com/Cluas/subhub/internal/config"
	"github.com/Cluas/subhub/internal/store"
)

// SettingsHandler serves runtime configuration for the settings page.
type SettingsHandler struct {
	Config *config.Config
	Store  store.Store
}

// Get handles GET /api/settings — returns masked runtime config.
func (h *SettingsHandler) Get(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	baseURL := h.Config.BaseURL
	if h.Store != nil {
		baseURL = h.Store.GetSystemSetting(ctx, "base_url", h.Config.BaseURL)
	}

	resp := map[string]interface{}{
		"port":              h.Config.Port,
		"db_path":           h.Config.DBPath,
		"base_url":          baseURL,
		"cache_ttl_seconds": int(h.Config.CacheTTL.Seconds()),
		"cache_max_entries": h.Config.CacheMaxEntries,
		"cors_origins":      h.Config.CORSOrigins,
		"log_level":         h.Config.LogLevel,
		"api_token_set":     h.Config.APIToken != "",
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// Update handles PUT /api/settings — updates system settings.
func (h *SettingsHandler) Update(w http.ResponseWriter, r *http.Request) {
	var req map[string]string
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	ctx := r.Context()
	if v, ok := req["base_url"]; ok {
		if err := h.Store.SetSystemSetting(ctx, "base_url", v); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to save base_url")
			return
		}
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
