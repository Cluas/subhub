package model

import "time"

// ProxyGroupData stores the membership of a proxy-group: group name -> list of proxy names.
type ProxyGroupData struct {
	// Groups maps proxy-group name to a list of member proxy names (after expansion).
	Groups map[string][]string `json:"groups"`
}

// Subscription represents a subscription source.
type Subscription struct {
	ID              int64           `json:"id"`
	Name            string          `json:"name"`
	URL             string          `json:"url"`
	Type            string          `json:"type"`         // clash, v2ray, sip002
	AutoRefresh     bool            `json:"auto_refresh"`
	RefreshCron     string          `json:"refresh_cron"` // e.g. "0 */6 * * *"
	LastFetchAt     *time.Time      `json:"last_fetch_at"`
	NodeCount       int             `json:"node_count"`
	Status          string          `json:"status"`    // active, error, disabled
	ErrorMsg        string          `json:"error_msg"`
	ProxyGroupsData *ProxyGroupData `json:"proxy_groups_data,omitempty"` // proxy-group membership, persisted as JSON
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
}
