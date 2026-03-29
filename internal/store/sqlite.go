package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/Cluas/subhub/internal/model"

	_ "modernc.org/sqlite"
)

type sqliteStore struct {
	db *sql.DB
}

// NewSQLite creates a SQLite storage instance and auto-initializes the schema.
func NewSQLite(path string) (Store, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	// WAL mode improves concurrent performance
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		db.Close()
		return nil, fmt.Errorf("set WAL mode: %w", err)
	}
	if _, err := db.Exec("PRAGMA foreign_keys=ON"); err != nil {
		db.Close()
		return nil, fmt.Errorf("enable foreign keys: %w", err)
	}

	s := &sqliteStore{db: db}
	if err := s.migrate(); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return s, nil
}

func (s *sqliteStore) migrate() error {
	// Step 1: Create tables with original schema (IF NOT EXISTS is idempotent).
	_, err := s.db.Exec(`
CREATE TABLE IF NOT EXISTS subscriptions (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    name         TEXT NOT NULL,
    url          TEXT NOT NULL,
    type         TEXT NOT NULL DEFAULT 'clash',
    auto_refresh INTEGER NOT NULL DEFAULT 0,
    refresh_cron TEXT NOT NULL DEFAULT '',
    last_fetch_at DATETIME,
    node_count   INTEGER NOT NULL DEFAULT 0,
    status       TEXT NOT NULL DEFAULT 'active',
    error_msg    TEXT NOT NULL DEFAULT '',
    created_at   DATETIME NOT NULL,
    updated_at   DATETIME NOT NULL
);

CREATE TABLE IF NOT EXISTS proxies (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    subscription_id INTEGER NOT NULL,
    name            TEXT NOT NULL,
    type            TEXT NOT NULL,
    server          TEXT NOT NULL,
    port            INTEGER NOT NULL,
    config          TEXT NOT NULL DEFAULT '{}',
    region          TEXT NOT NULL DEFAULT '',
    latency         INTEGER,
    alive           INTEGER,
    last_check_at   DATETIME,
    created_at      DATETIME NOT NULL,
    updated_at      DATETIME NOT NULL,
    FOREIGN KEY (subscription_id) REFERENCES subscriptions(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS rules (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    subscription_id INTEGER NOT NULL,
    provider_name   TEXT NOT NULL,
    type            TEXT NOT NULL,
    payload         TEXT NOT NULL,
    target          TEXT NOT NULL,
    FOREIGN KEY (subscription_id) REFERENCES subscriptions(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_proxies_subscription ON proxies(subscription_id);
CREATE INDEX IF NOT EXISTS idx_rules_subscription ON rules(subscription_id);

CREATE TABLE IF NOT EXISTS endpoints (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    name            TEXT NOT NULL,
    slug            TEXT NOT NULL UNIQUE,
    subscription_id INTEGER NOT NULL,
    output_type     TEXT NOT NULL DEFAULT 'proxy',
    format          TEXT NOT NULL DEFAULT 'clash',
    filters         TEXT NOT NULL DEFAULT '{}',
    created_at      DATETIME NOT NULL,
    updated_at      DATETIME NOT NULL,
    FOREIGN KEY (subscription_id) REFERENCES subscriptions(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_endpoints_slug ON endpoints(slug);

CREATE TABLE IF NOT EXISTS profiles (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    name        TEXT NOT NULL,
    slug        TEXT NOT NULL UNIQUE,
    created_at  DATETIME NOT NULL,
    updated_at  DATETIME NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_profiles_slug ON profiles(slug);

CREATE TABLE IF NOT EXISTS collections (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    name         TEXT NOT NULL,
    content_type TEXT NOT NULL CHECK(content_type IN ('proxy', 'rule')),
    description  TEXT NOT NULL DEFAULT '',
    created_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS profile_node_pools (
    id                    INTEGER PRIMARY KEY AUTOINCREMENT,
    profile_id            INTEGER NOT NULL,
    name                  TEXT NOT NULL,
    endpoint_id           INTEGER,
    endpoint_slug         TEXT NOT NULL DEFAULT '',
    health_check_url      TEXT NOT NULL DEFAULT '',
    health_check_interval INTEGER NOT NULL DEFAULT 300,
    position              INTEGER NOT NULL DEFAULT 0,
    created_at            DATETIME NOT NULL,
    updated_at            DATETIME NOT NULL,
    FOREIGN KEY (profile_id) REFERENCES profiles(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_profile_node_pools_profile ON profile_node_pools(profile_id);

CREATE TABLE IF NOT EXISTS profile_rule_sets (
    id             INTEGER PRIMARY KEY AUTOINCREMENT,
    profile_id     INTEGER NOT NULL,
    name           TEXT NOT NULL,
    endpoint_id    INTEGER,
    endpoint_slug  TEXT NOT NULL DEFAULT '',
    external_url   TEXT NOT NULL DEFAULT '',
    metadata       TEXT NOT NULL DEFAULT '{}',
    interval       INTEGER NOT NULL DEFAULT 86400,
    position       INTEGER NOT NULL DEFAULT 0,
    created_at     DATETIME NOT NULL,
    updated_at     DATETIME NOT NULL,
    FOREIGN KEY (profile_id) REFERENCES profiles(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_profile_rule_sets_profile ON profile_rule_sets(profile_id);

CREATE TABLE IF NOT EXISTS profile_strategies (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    profile_id INTEGER NOT NULL,
    name       TEXT NOT NULL,
    strategy   TEXT NOT NULL DEFAULT 'select',
    pools      TEXT NOT NULL DEFAULT '[]',
    proxies    TEXT NOT NULL DEFAULT '[]',
    config     TEXT NOT NULL DEFAULT '{}',
    position   INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL,
    FOREIGN KEY (profile_id) REFERENCES profiles(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_profile_strategies_profile ON profile_strategies(profile_id);

CREATE TABLE IF NOT EXISTS profile_routing_rules (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    profile_id INTEGER NOT NULL,
    type       TEXT NOT NULL,
    payload    TEXT NOT NULL DEFAULT '',
    target     TEXT NOT NULL DEFAULT '',
    no_resolve INTEGER NOT NULL DEFAULT 0,
    position   INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL,
    FOREIGN KEY (profile_id) REFERENCES profiles(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_profile_routing_rules_profile ON profile_routing_rules(profile_id);

CREATE TABLE IF NOT EXISTS provider_defs (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    subscription_id INTEGER NOT NULL,
    name            TEXT NOT NULL,
    kind            TEXT NOT NULL DEFAULT '',
    type            TEXT NOT NULL DEFAULT '',
    behavior        TEXT NOT NULL DEFAULT '',
    url             TEXT NOT NULL DEFAULT '',
    interval        INTEGER NOT NULL DEFAULT 0,
    path            TEXT NOT NULL DEFAULT '',
    created_at      DATETIME NOT NULL,
    updated_at      DATETIME NOT NULL,
    FOREIGN KEY (subscription_id) REFERENCES subscriptions(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_provider_defs_subscription ON provider_defs(subscription_id);
`)
	if err != nil {
		return err
	}

	// Step 2: Add collection_id column to tables that may not have it yet (before table recreation).
	for _, table := range []string{"proxies", "rules", "endpoints"} {
		s.addColumnIfMissing(table, "collection_id", "INTEGER")
	}

	// Step 3: Add settings column to profiles if missing.
	s.addColumnIfMissing("profiles", "settings", "TEXT DEFAULT '{}'")

	// Step 3b: Add proxy_groups_data column to subscriptions if missing.
	s.addColumnIfMissing("subscriptions", "proxy_groups_data", "TEXT NOT NULL DEFAULT '{}'")

	// Step 4: Drop legacy profile_blocks table if it exists.
	if _, err := s.db.Exec(`DROP TABLE IF EXISTS profile_blocks`); err != nil {
		return fmt.Errorf("drop profile_blocks: %w", err)
	}

	// Step 5: Idempotent migration — make subscription_id nullable on proxies, rules, endpoints.
	if err := s.makeNullableIfNeeded("proxies", `
CREATE TABLE new_proxies (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    subscription_id INTEGER,
    name            TEXT NOT NULL,
    type            TEXT NOT NULL,
    server          TEXT NOT NULL,
    port            INTEGER NOT NULL,
    config          TEXT NOT NULL DEFAULT '{}',
    region          TEXT NOT NULL DEFAULT '',
    latency         INTEGER,
    alive           INTEGER,
    last_check_at   DATETIME,
    created_at      DATETIME NOT NULL,
    updated_at      DATETIME NOT NULL,
    collection_id   INTEGER REFERENCES collections(id) ON DELETE CASCADE
)`, "new_proxies",
		"DROP INDEX IF EXISTS idx_proxies_subscription",
		"CREATE INDEX IF NOT EXISTS idx_proxies_subscription ON proxies(subscription_id)",
	); err != nil {
		return fmt.Errorf("nullable proxies migration: %w", err)
	}

	if err := s.makeNullableIfNeeded("rules", `
CREATE TABLE new_rules (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    subscription_id INTEGER,
    provider_name   TEXT NOT NULL,
    type            TEXT NOT NULL,
    payload         TEXT NOT NULL,
    target          TEXT NOT NULL,
    collection_id   INTEGER REFERENCES collections(id) ON DELETE CASCADE
)`, "new_rules",
		"DROP INDEX IF EXISTS idx_rules_subscription",
		"CREATE INDEX IF NOT EXISTS idx_rules_subscription ON rules(subscription_id)",
	); err != nil {
		return fmt.Errorf("nullable rules migration: %w", err)
	}

	if err := s.makeNullableIfNeeded("endpoints", `
CREATE TABLE new_endpoints (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    name            TEXT NOT NULL,
    slug            TEXT NOT NULL UNIQUE,
    subscription_id INTEGER,
    output_type     TEXT NOT NULL DEFAULT 'proxy',
    format          TEXT NOT NULL DEFAULT 'clash',
    filters         TEXT NOT NULL DEFAULT '{}',
    created_at      DATETIME NOT NULL,
    updated_at      DATETIME NOT NULL,
    collection_id   INTEGER REFERENCES collections(id) ON DELETE SET NULL
)`, "new_endpoints",
		"DROP INDEX IF EXISTS idx_endpoints_slug",
		"CREATE INDEX IF NOT EXISTS idx_endpoints_slug ON endpoints(slug)",
	); err != nil {
		return fmt.Errorf("nullable endpoints migration: %w", err)
	}

	// Step 6: System settings KV table.
	if _, err := s.db.Exec(`
CREATE TABLE IF NOT EXISTS system_settings (
    key   TEXT PRIMARY KEY,
    value TEXT NOT NULL DEFAULT ''
)`); err != nil {
		return fmt.Errorf("create system_settings: %w", err)
	}

	return nil
}

// GetSystemSetting returns a system setting value by key, or fallback if not set.
func (s *sqliteStore) GetSystemSetting(ctx context.Context, key, fallback string) string {
	var val string
	err := s.db.QueryRowContext(ctx, `SELECT value FROM system_settings WHERE key=?`, key).Scan(&val)
	if err != nil || val == "" {
		return fallback
	}
	return val
}

// SetSystemSetting upserts a system setting.
func (s *sqliteStore) SetSystemSetting(ctx context.Context, key, value string) error {
	_, err := s.db.ExecContext(ctx, `
INSERT INTO system_settings (key, value) VALUES (?, ?)
ON CONFLICT(key) DO UPDATE SET value=excluded.value`, key, value)
	return err
}

// makeNullableIfNeeded checks if subscription_id on tableName is NOT NULL.
// If it is, it recreates the table using createNewTableSQL (with nullable subscription_id),
// copying all existing data, then drops the old index, renames, and recreates the index.
func (s *sqliteStore) makeNullableIfNeeded(tableName, createNewTableSQL, newTableName, dropIndexSQL, createIndexSQL string) error {
	// Query PRAGMA table_info to check notnull on subscription_id.
	rows, err := s.db.Query(fmt.Sprintf("PRAGMA table_info(%s)", tableName))
	if err != nil {
		return err
	}
	defer rows.Close()

	isNotNull := false
	for rows.Next() {
		var cid int
		var name, colType string
		var notNull int
		var dfltValue sql.NullString
		var pk int
		if err := rows.Scan(&cid, &name, &colType, &notNull, &dfltValue, &pk); err != nil {
			return err
		}
		if name == "subscription_id" && notNull == 1 {
			isNotNull = true
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}

	if !isNotNull {
		// Already nullable — nothing to do.
		return nil
	}

	// Recreate table with nullable subscription_id using SQLite rename trick.
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Disable FK enforcement during restructure (foreign_keys is per-connection but we need it off here).
	if _, err := tx.Exec("PRAGMA foreign_keys=OFF"); err != nil {
		return err
	}

	if _, err := tx.Exec(createNewTableSQL); err != nil {
		return err
	}

	if _, err := tx.Exec(fmt.Sprintf("INSERT INTO %s SELECT * FROM %s", newTableName, tableName)); err != nil {
		return err
	}

	if _, err := tx.Exec(dropIndexSQL); err != nil {
		return err
	}

	if _, err := tx.Exec(fmt.Sprintf("DROP TABLE %s", tableName)); err != nil {
		return err
	}

	if _, err := tx.Exec(fmt.Sprintf("ALTER TABLE %s RENAME TO %s", newTableName, tableName)); err != nil {
		return err
	}

	if _, err := tx.Exec(createIndexSQL); err != nil {
		return err
	}

	if _, err := tx.Exec("PRAGMA foreign_keys=ON"); err != nil {
		return err
	}

	return tx.Commit()
}

// addColumnIfMissing adds a column to a table if it does not already exist.
func (s *sqliteStore) addColumnIfMissing(table, column, colType string) {
	rows, err := s.db.Query(fmt.Sprintf("PRAGMA table_info(%s)", table))
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var cid int
		var name, ct string
		var notNull int
		var dfltValue sql.NullString
		var pk int
		if err := rows.Scan(&cid, &name, &ct, &notNull, &dfltValue, &pk); err != nil {
			return
		}
		if name == column {
			return // already exists
		}
	}

	s.db.Exec(fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", table, column, colType))
}

// ---- Collections ----

func (s *sqliteStore) CreateCollection(ctx context.Context, c *model.Collection) error {
	now := time.Now()
	c.CreatedAt = now
	c.UpdatedAt = now
	res, err := s.db.ExecContext(ctx, `
INSERT INTO collections (name, content_type, description, created_at, updated_at)
VALUES (?, ?, ?, ?, ?)`,
		c.Name, c.ContentType, c.Description, c.CreatedAt, c.UpdatedAt)
	if err != nil {
		return err
	}
	c.ID, _ = res.LastInsertId()
	return nil
}

func (s *sqliteStore) GetCollection(ctx context.Context, id int64) (*model.Collection, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT id, name, content_type, description, created_at, updated_at
FROM collections WHERE id=?`, id)
	return scanCollection(row)
}

func (s *sqliteStore) ListCollections(ctx context.Context) ([]*model.Collection, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT id, name, content_type, description, created_at, updated_at
FROM collections ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*model.Collection
	for rows.Next() {
		c, err := scanCollection(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, c)
	}
	return result, rows.Err()
}

func (s *sqliteStore) UpdateCollection(ctx context.Context, c *model.Collection) error {
	c.UpdatedAt = time.Now()
	_, err := s.db.ExecContext(ctx, `
UPDATE collections SET name=?, content_type=?, description=?, updated_at=? WHERE id=?`,
		c.Name, c.ContentType, c.Description, c.UpdatedAt, c.ID)
	return err
}

func (s *sqliteStore) DeleteCollection(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM collections WHERE id=?`, id)
	return err
}

// ---- Subscriptions ----

func (s *sqliteStore) CreateSubscription(ctx context.Context, sub *model.Subscription) error {
	now := time.Now()
	sub.CreatedAt = now
	sub.UpdatedAt = now
	pgdJSON := marshalProxyGroupsData(sub.ProxyGroupsData)
	res, err := s.db.ExecContext(ctx, `
INSERT INTO subscriptions (name, url, type, auto_refresh, refresh_cron, node_count, status, error_msg, proxy_groups_data, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		sub.Name, sub.URL, sub.Type, boolToInt(sub.AutoRefresh), sub.RefreshCron,
		sub.NodeCount, sub.Status, sub.ErrorMsg, pgdJSON, sub.CreatedAt, sub.UpdatedAt)
	if err != nil {
		return err
	}
	sub.ID, _ = res.LastInsertId()
	return nil
}

func (s *sqliteStore) GetSubscription(ctx context.Context, id int64) (*model.Subscription, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT id, name, url, type, auto_refresh, refresh_cron, last_fetch_at, node_count, status, error_msg, proxy_groups_data, created_at, updated_at
FROM subscriptions WHERE id = ?`, id)
	return scanSubscription(row)
}

func (s *sqliteStore) ListSubscriptions(ctx context.Context) ([]*model.Subscription, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT id, name, url, type, auto_refresh, refresh_cron, last_fetch_at, node_count, status, error_msg, proxy_groups_data, created_at, updated_at
FROM subscriptions ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*model.Subscription
	for rows.Next() {
		sub, err := scanSubscription(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, sub)
	}
	return result, rows.Err()
}

func (s *sqliteStore) UpdateSubscription(ctx context.Context, sub *model.Subscription) error {
	sub.UpdatedAt = time.Now()
	pgdJSON := marshalProxyGroupsData(sub.ProxyGroupsData)
	_, err := s.db.ExecContext(ctx, `
UPDATE subscriptions SET name=?, url=?, type=?, auto_refresh=?, refresh_cron=?,
last_fetch_at=?, node_count=?, status=?, error_msg=?, proxy_groups_data=?, updated_at=?
WHERE id=?`,
		sub.Name, sub.URL, sub.Type, boolToInt(sub.AutoRefresh), sub.RefreshCron,
		sub.LastFetchAt, sub.NodeCount, sub.Status, sub.ErrorMsg, pgdJSON, sub.UpdatedAt, sub.ID)
	return err
}

func (s *sqliteStore) DeleteSubscription(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM subscriptions WHERE id=?`, id)
	return err
}

// ---- Proxies ----

func (s *sqliteStore) UpsertProxies(ctx context.Context, subscriptionID int64, proxies []*model.Proxy) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `DELETE FROM proxies WHERE subscription_id=?`, subscriptionID); err != nil {
		return err
	}

	now := time.Now()
	for _, p := range proxies {
		cfgJSON, _ := json.Marshal(p.Config)
		p.CreatedAt = now
		p.UpdatedAt = now
		res, err := tx.ExecContext(ctx, `
INSERT INTO proxies (subscription_id, name, type, server, port, config, region, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			subscriptionID, p.Name, p.Type, p.Server, p.Port, string(cfgJSON), p.Region, now, now)
		if err != nil {
			return err
		}
		p.ID, _ = res.LastInsertId()
		p.SubscriptionID = model.Int64Ptr(subscriptionID)
	}
	return tx.Commit()
}

func (s *sqliteStore) ListProxies(ctx context.Context, filter ProxyFilter) ([]*model.Proxy, error) {
	var conditions []string
	var args []interface{}

	if filter.SubscriptionID != 0 {
		conditions = append(conditions, "subscription_id=?")
		args = append(args, filter.SubscriptionID)
	}
	if filter.CollectionID != 0 {
		conditions = append(conditions, "collection_id=?")
		args = append(args, filter.CollectionID)
	}
	if filter.Region != "" {
		conditions = append(conditions, "region=?")
		args = append(args, filter.Region)
	}
	if filter.Type != "" {
		conditions = append(conditions, "type=?")
		args = append(args, filter.Type)
	}
	if filter.LatencyMax > 0 {
		conditions = append(conditions, "(latency IS NULL OR latency <= ?)")
		args = append(args, filter.LatencyMax)
	}
	if filter.Alive != nil {
		conditions = append(conditions, "alive=?")
		args = append(args, boolToInt(*filter.Alive))
	}
	if filter.NameContains != "" {
		conditions = append(conditions, "name LIKE ?")
		args = append(args, "%"+filter.NameContains+"%")
	}
	if len(filter.Names) > 0 {
		placeholders := make([]string, len(filter.Names))
		for i, n := range filter.Names {
			placeholders[i] = "?"
			args = append(args, n)
		}
		conditions = append(conditions, "name IN ("+strings.Join(placeholders, ",")+")")
	}
	// Groups: OR-join LIKE conditions across all group strings.
	// Proxy-group membership is not stored per row; we match by proxy name substring.
	if len(filter.Groups) > 0 {
		likes := make([]string, len(filter.Groups))
		for i, g := range filter.Groups {
			likes[i] = "name LIKE ?"
			args = append(args, "%"+g+"%")
		}
		conditions = append(conditions, "("+strings.Join(likes, " OR ")+")")
	}

	query := `SELECT id, subscription_id, collection_id, name, type, server, port, config, region, latency, alive, last_check_at, created_at, updated_at FROM proxies`
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	if filter.SortByLatency {
		query += " ORDER BY latency ASC NULLS LAST"
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*model.Proxy
	for rows.Next() {
		p, err := scanProxy(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, p)
	}
	return result, rows.Err()
}

func (s *sqliteStore) GetProxy(ctx context.Context, id int64) (*model.Proxy, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT id, subscription_id, collection_id, name, type, server, port, config, region, latency, alive, last_check_at, created_at, updated_at
FROM proxies WHERE id=?`, id)
	return scanProxy(row)
}

func (s *sqliteStore) UpdateProxyHealth(ctx context.Context, id int64, alive bool, latencyMs int) error {
	now := time.Now()
	_, err := s.db.ExecContext(ctx, `
UPDATE proxies SET alive=?, latency=?, last_check_at=?, updated_at=? WHERE id=?`,
		boolToInt(alive), latencyMs, now, now, id)
	return err
}

func (s *sqliteStore) DeleteProxiesBySubscription(ctx context.Context, subscriptionID int64) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM proxies WHERE subscription_id=?`, subscriptionID)
	return err
}

// CreateProxy inserts a self-managed proxy (subscription_id may be nil).
func (s *sqliteStore) CreateProxy(ctx context.Context, p *model.Proxy) error {
	now := time.Now()
	p.CreatedAt = now
	p.UpdatedAt = now
	cfgJSON, _ := json.Marshal(p.Config)
	res, err := s.db.ExecContext(ctx, `
INSERT INTO proxies (subscription_id, collection_id, name, type, server, port, config, region, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		nullableInt64(p.SubscriptionID), nullableInt64(p.CollectionID), p.Name, p.Type, p.Server, p.Port,
		string(cfgJSON), p.Region, now, now)
	if err != nil {
		return err
	}
	p.ID, _ = res.LastInsertId()
	return nil
}

// UpdateProxy updates mutable fields on an existing proxy.
func (s *sqliteStore) UpdateProxy(ctx context.Context, p *model.Proxy) error {
	p.UpdatedAt = time.Now()
	cfgJSON, _ := json.Marshal(p.Config)
	_, err := s.db.ExecContext(ctx, `
UPDATE proxies SET name=?, type=?, server=?, port=?, config=?, region=?, collection_id=?, updated_at=? WHERE id=?`,
		p.Name, p.Type, p.Server, p.Port, string(cfgJSON), p.Region, nullableInt64(p.CollectionID), p.UpdatedAt, p.ID)
	return err
}

// DeleteProxy removes a proxy by ID.
func (s *sqliteStore) DeleteProxy(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM proxies WHERE id=?`, id)
	return err
}

// ---- Rules ----

func (s *sqliteStore) UpsertRules(ctx context.Context, subscriptionID int64, rules []*model.Rule) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `DELETE FROM rules WHERE subscription_id=?`, subscriptionID); err != nil {
		return err
	}

	for _, r := range rules {
		res, err := tx.ExecContext(ctx, `
INSERT INTO rules (subscription_id, provider_name, type, payload, target)
VALUES (?, ?, ?, ?, ?)`,
			subscriptionID, r.ProviderName, r.Type, r.Payload, r.Target)
		if err != nil {
			return err
		}
		r.ID, _ = res.LastInsertId()
		r.SubscriptionID = model.Int64Ptr(subscriptionID)
	}
	return tx.Commit()
}

func (s *sqliteStore) ListRules(ctx context.Context, filter RuleFilter) ([]*model.Rule, error) {
	var conditions []string
	var args []interface{}

	if filter.SubscriptionID != 0 {
		conditions = append(conditions, "subscription_id=?")
		args = append(args, filter.SubscriptionID)
	}
	if filter.CollectionID != 0 {
		conditions = append(conditions, "collection_id=?")
		args = append(args, filter.CollectionID)
	}
	if filter.ProviderName != "" {
		conditions = append(conditions, "provider_name=?")
		args = append(args, filter.ProviderName)
	}
	if filter.Type != "" {
		conditions = append(conditions, "type=?")
		args = append(args, filter.Type)
	}
	if filter.Target != "" {
		conditions = append(conditions, "target=?")
		args = append(args, filter.Target)
	}
	if filter.Keyword != "" {
		conditions = append(conditions, "(payload LIKE ? OR type LIKE ?)")
		args = append(args, "%"+filter.Keyword+"%", "%"+filter.Keyword+"%")
	}

	query := `SELECT id, subscription_id, collection_id, provider_name, type, payload, target FROM rules`
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*model.Rule
	for rows.Next() {
		r := &model.Rule{}
		var subID sql.NullInt64
		var colID sql.NullInt64
		if err := rows.Scan(&r.ID, &subID, &colID, &r.ProviderName, &r.Type, &r.Payload, &r.Target); err != nil {
			return nil, err
		}
		if subID.Valid {
			r.SubscriptionID = &subID.Int64
		}
		if colID.Valid {
			r.CollectionID = &colID.Int64
		}
		result = append(result, r)
	}
	return result, rows.Err()
}

func (s *sqliteStore) DeleteRulesBySubscription(ctx context.Context, subscriptionID int64) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM rules WHERE subscription_id=?`, subscriptionID)
	return err
}

// GetRule fetches a single rule by ID.
func (s *sqliteStore) GetRule(ctx context.Context, id int64) (*model.Rule, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT id, subscription_id, collection_id, provider_name, type, payload, target FROM rules WHERE id=?`, id)
	r := &model.Rule{}
	var subID sql.NullInt64
	var colID sql.NullInt64
	if err := row.Scan(&r.ID, &subID, &colID, &r.ProviderName, &r.Type, &r.Payload, &r.Target); err != nil {
		return nil, err
	}
	if subID.Valid {
		r.SubscriptionID = &subID.Int64
	}
	if colID.Valid {
		r.CollectionID = &colID.Int64
	}
	return r, nil
}

// CreateRule inserts a self-managed rule (subscription_id may be nil).
func (s *sqliteStore) CreateRule(ctx context.Context, r *model.Rule) error {
	res, err := s.db.ExecContext(ctx, `
INSERT INTO rules (subscription_id, collection_id, provider_name, type, payload, target)
VALUES (?, ?, ?, ?, ?, ?)`,
		nullableInt64(r.SubscriptionID), nullableInt64(r.CollectionID), r.ProviderName, r.Type, r.Payload, r.Target)
	if err != nil {
		return err
	}
	r.ID, _ = res.LastInsertId()
	return nil
}

// UpdateRule updates mutable fields on an existing rule.
func (s *sqliteStore) UpdateRule(ctx context.Context, r *model.Rule) error {
	_, err := s.db.ExecContext(ctx, `
UPDATE rules SET provider_name=?, type=?, payload=?, target=?, collection_id=? WHERE id=?`,
		r.ProviderName, r.Type, r.Payload, r.Target, nullableInt64(r.CollectionID), r.ID)
	return err
}

// DeleteRule removes a rule by ID.
func (s *sqliteStore) DeleteRule(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM rules WHERE id=?`, id)
	return err
}

func (s *sqliteStore) Close() error {
	return s.db.Close()
}

// ---- Endpoints ----

func (s *sqliteStore) CreateEndpoint(ctx context.Context, e *model.Endpoint) error {
	now := time.Now()
	e.CreatedAt = now
	e.UpdatedAt = now
	filtersJSON, _ := json.Marshal(e.Filters)
	res, err := s.db.ExecContext(ctx, `
INSERT INTO endpoints (name, slug, subscription_id, collection_id, output_type, format, filters, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		e.Name, e.Slug, nullableInt64(e.SubscriptionID), nullableInt64(e.CollectionID), e.OutputType, e.Format, string(filtersJSON), e.CreatedAt, e.UpdatedAt)
	if err != nil {
		return err
	}
	e.ID, _ = res.LastInsertId()
	return nil
}

func (s *sqliteStore) GetEndpoint(ctx context.Context, id int64) (*model.Endpoint, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT id, name, slug, subscription_id, collection_id, output_type, format, filters, created_at, updated_at
FROM endpoints WHERE id=?`, id)
	return scanEndpoint(row)
}

func (s *sqliteStore) GetEndpointBySlug(ctx context.Context, slug string) (*model.Endpoint, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT id, name, slug, subscription_id, collection_id, output_type, format, filters, created_at, updated_at
FROM endpoints WHERE slug=?`, slug)
	return scanEndpoint(row)
}

func (s *sqliteStore) ListEndpoints(ctx context.Context) ([]*model.Endpoint, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT id, name, slug, subscription_id, collection_id, output_type, format, filters, created_at, updated_at
FROM endpoints ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*model.Endpoint
	for rows.Next() {
		e, err := scanEndpoint(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, e)
	}
	return result, rows.Err()
}

func (s *sqliteStore) UpdateEndpoint(ctx context.Context, e *model.Endpoint) error {
	e.UpdatedAt = time.Now()
	filtersJSON, _ := json.Marshal(e.Filters)
	_, err := s.db.ExecContext(ctx, `
UPDATE endpoints SET name=?, slug=?, output_type=?, format=?, filters=?, subscription_id=?, collection_id=?, updated_at=? WHERE id=?`,
		e.Name, e.Slug, e.OutputType, e.Format, string(filtersJSON), nullableInt64(e.SubscriptionID), nullableInt64(e.CollectionID), e.UpdatedAt, e.ID)
	return err
}

func (s *sqliteStore) DeleteEndpoint(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM endpoints WHERE id=?`, id)
	return err
}

// ---- Profiles ----

func (s *sqliteStore) CreateProfile(ctx context.Context, p *model.Profile) error {
	now := time.Now()
	p.CreatedAt = now
	p.UpdatedAt = now
	settingsJSON, _ := json.Marshal(p.Settings)
	if settingsJSON == nil {
		settingsJSON = []byte("{}")
	}
	res, err := s.db.ExecContext(ctx, `
INSERT INTO profiles (name, slug, settings, created_at, updated_at)
VALUES (?, ?, ?, ?, ?)`,
		p.Name, p.Slug, string(settingsJSON), p.CreatedAt, p.UpdatedAt)
	if err != nil {
		return err
	}
	p.ID, _ = res.LastInsertId()
	return nil
}

func (s *sqliteStore) GetProfile(ctx context.Context, id int64) (*model.Profile, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT p.id, p.name, p.slug, p.settings,
  (SELECT COUNT(*) FROM profile_node_pools WHERE profile_id = p.id) AS node_pool_count,
  (SELECT COUNT(*) FROM profile_rule_sets WHERE profile_id = p.id) AS rule_set_count,
  (SELECT COUNT(*) FROM profile_strategies WHERE profile_id = p.id) AS strategy_count,
  (SELECT COUNT(*) FROM profile_routing_rules WHERE profile_id = p.id) AS routing_rule_count,
  p.created_at, p.updated_at
FROM profiles p WHERE p.id=?`, id)
	return scanProfile(row)
}

func (s *sqliteStore) GetProfileBySlug(ctx context.Context, slug string) (*model.Profile, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT p.id, p.name, p.slug, p.settings,
  (SELECT COUNT(*) FROM profile_node_pools WHERE profile_id = p.id) AS node_pool_count,
  (SELECT COUNT(*) FROM profile_rule_sets WHERE profile_id = p.id) AS rule_set_count,
  (SELECT COUNT(*) FROM profile_strategies WHERE profile_id = p.id) AS strategy_count,
  (SELECT COUNT(*) FROM profile_routing_rules WHERE profile_id = p.id) AS routing_rule_count,
  p.created_at, p.updated_at
FROM profiles p WHERE p.slug=?`, slug)
	return scanProfile(row)
}

func (s *sqliteStore) ListProfiles(ctx context.Context) ([]*model.Profile, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT p.id, p.name, p.slug, p.settings,
  (SELECT COUNT(*) FROM profile_node_pools WHERE profile_id = p.id) AS node_pool_count,
  (SELECT COUNT(*) FROM profile_rule_sets WHERE profile_id = p.id) AS rule_set_count,
  (SELECT COUNT(*) FROM profile_strategies WHERE profile_id = p.id) AS strategy_count,
  (SELECT COUNT(*) FROM profile_routing_rules WHERE profile_id = p.id) AS routing_rule_count,
  p.created_at, p.updated_at
FROM profiles p ORDER BY p.id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*model.Profile
	for rows.Next() {
		p, err := scanProfile(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, p)
	}
	return result, rows.Err()
}

func (s *sqliteStore) UpdateProfile(ctx context.Context, p *model.Profile) error {
	p.UpdatedAt = time.Now()
	settingsJSON, _ := json.Marshal(p.Settings)
	if settingsJSON == nil {
		settingsJSON = []byte("{}")
	}
	_, err := s.db.ExecContext(ctx, `
UPDATE profiles SET name=?, slug=?, settings=?, updated_at=? WHERE id=?`,
		p.Name, p.Slug, string(settingsJSON), p.UpdatedAt, p.ID)
	return err
}

func (s *sqliteStore) DeleteProfile(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM profiles WHERE id=?`, id)
	return err
}

// ---- ProfileNodePools ----

func (s *sqliteStore) CreateProfileNodePool(ctx context.Context, np *model.ProfileNodePool) error {
	now := time.Now()
	np.CreatedAt = now
	np.UpdatedAt = now
	res, err := s.db.ExecContext(ctx, `
INSERT INTO profile_node_pools (profile_id, name, endpoint_id, endpoint_slug, health_check_url, health_check_interval, position, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		np.ProfileID, np.NameStr, nullableInt64(np.EndpointID), np.EndpointSlugValue,
		np.HealthCheckURL, np.HealthCheckInterval, np.Position, now, now)
	if err != nil {
		return err
	}
	np.ID, _ = res.LastInsertId()
	return nil
}

func (s *sqliteStore) GetProfileNodePool(ctx context.Context, id int64) (*model.ProfileNodePool, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT id, profile_id, name, endpoint_id, endpoint_slug, health_check_url, health_check_interval, position, created_at, updated_at
FROM profile_node_pools WHERE id=?`, id)
	return scanProfileNodePool(row)
}

func (s *sqliteStore) ListProfileNodePools(ctx context.Context, profileID int64) ([]*model.ProfileNodePool, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT id, profile_id, name, endpoint_id, endpoint_slug, health_check_url, health_check_interval, position, created_at, updated_at
FROM profile_node_pools WHERE profile_id=? ORDER BY position ASC`, profileID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*model.ProfileNodePool
	for rows.Next() {
		np, err := scanProfileNodePool(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, np)
	}
	return result, rows.Err()
}

func (s *sqliteStore) UpdateProfileNodePool(ctx context.Context, np *model.ProfileNodePool) error {
	np.UpdatedAt = time.Now()
	_, err := s.db.ExecContext(ctx, `
UPDATE profile_node_pools SET name=?, endpoint_id=?, endpoint_slug=?, health_check_url=?, health_check_interval=?, position=?, updated_at=? WHERE id=?`,
		np.NameStr, nullableInt64(np.EndpointID), np.EndpointSlugValue,
		np.HealthCheckURL, np.HealthCheckInterval, np.Position, np.UpdatedAt, np.ID)
	return err
}

func (s *sqliteStore) DeleteProfileNodePool(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM profile_node_pools WHERE id=?`, id)
	return err
}

// ---- ProfileRuleSets ----

func (s *sqliteStore) CreateProfileRuleSet(ctx context.Context, rs *model.ProfileRuleSet) error {
	now := time.Now()
	rs.CreatedAt = now
	rs.UpdatedAt = now
	metaJSON, _ := json.Marshal(rs.MetadataJSON)
	if metaJSON == nil {
		metaJSON = []byte("{}")
	}
	res, err := s.db.ExecContext(ctx, `
INSERT INTO profile_rule_sets (profile_id, name, endpoint_id, endpoint_slug, external_url, metadata, interval, position, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		rs.ProfileID, rs.NameStr, nullableInt64(rs.EndpointID), rs.EndpointSlugValue,
		rs.ExternalURL, string(metaJSON), rs.Interval, rs.Position, now, now)
	if err != nil {
		return err
	}
	rs.ID, _ = res.LastInsertId()
	return nil
}

func (s *sqliteStore) GetProfileRuleSet(ctx context.Context, id int64) (*model.ProfileRuleSet, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT id, profile_id, name, endpoint_id, endpoint_slug, external_url, metadata, interval, position, created_at, updated_at
FROM profile_rule_sets WHERE id=?`, id)
	return scanProfileRuleSet(row)
}

func (s *sqliteStore) ListProfileRuleSets(ctx context.Context, profileID int64) ([]*model.ProfileRuleSet, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT id, profile_id, name, endpoint_id, endpoint_slug, external_url, metadata, interval, position, created_at, updated_at
FROM profile_rule_sets WHERE profile_id=? ORDER BY position ASC`, profileID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*model.ProfileRuleSet
	for rows.Next() {
		rs, err := scanProfileRuleSet(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, rs)
	}
	return result, rows.Err()
}

func (s *sqliteStore) UpdateProfileRuleSet(ctx context.Context, rs *model.ProfileRuleSet) error {
	rs.UpdatedAt = time.Now()
	metaJSON, _ := json.Marshal(rs.MetadataJSON)
	if metaJSON == nil {
		metaJSON = []byte("{}")
	}
	_, err := s.db.ExecContext(ctx, `
UPDATE profile_rule_sets SET name=?, endpoint_id=?, endpoint_slug=?, external_url=?, metadata=?, interval=?, position=?, updated_at=? WHERE id=?`,
		rs.NameStr, nullableInt64(rs.EndpointID), rs.EndpointSlugValue,
		rs.ExternalURL, string(metaJSON), rs.Interval, rs.Position, rs.UpdatedAt, rs.ID)
	return err
}

func (s *sqliteStore) DeleteProfileRuleSet(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM profile_rule_sets WHERE id=?`, id)
	return err
}

// ---- ProfileStrategies ----

func (s *sqliteStore) CreateProfileStrategy(ctx context.Context, st *model.ProfileStrategy) error {
	now := time.Now()
	st.CreatedAt = now
	st.UpdatedAt = now
	poolsJSON, _ := json.Marshal(st.PoolNames)
	proxiesJSON, _ := json.Marshal(st.ProxyNames)
	configJSON, _ := json.Marshal(st.ConfigJSON)
	if poolsJSON == nil {
		poolsJSON = []byte("[]")
	}
	if proxiesJSON == nil {
		proxiesJSON = []byte("[]")
	}
	if configJSON == nil {
		configJSON = []byte("{}")
	}
	res, err := s.db.ExecContext(ctx, `
INSERT INTO profile_strategies (profile_id, name, strategy, pools, proxies, config, position, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		st.ProfileID, st.NameStr, st.StrategyV,
		string(poolsJSON), string(proxiesJSON), string(configJSON),
		st.Position, now, now)
	if err != nil {
		return err
	}
	st.ID, _ = res.LastInsertId()
	return nil
}

func (s *sqliteStore) GetProfileStrategy(ctx context.Context, id int64) (*model.ProfileStrategy, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT id, profile_id, name, strategy, pools, proxies, config, position, created_at, updated_at
FROM profile_strategies WHERE id=?`, id)
	return scanProfileStrategy(row)
}

func (s *sqliteStore) ListProfileStrategies(ctx context.Context, profileID int64) ([]*model.ProfileStrategy, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT id, profile_id, name, strategy, pools, proxies, config, position, created_at, updated_at
FROM profile_strategies WHERE profile_id=? ORDER BY position ASC`, profileID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*model.ProfileStrategy
	for rows.Next() {
		st, err := scanProfileStrategy(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, st)
	}
	return result, rows.Err()
}

func (s *sqliteStore) UpdateProfileStrategy(ctx context.Context, st *model.ProfileStrategy) error {
	st.UpdatedAt = time.Now()
	poolsJSON, _ := json.Marshal(st.PoolNames)
	proxiesJSON, _ := json.Marshal(st.ProxyNames)
	configJSON, _ := json.Marshal(st.ConfigJSON)
	if poolsJSON == nil {
		poolsJSON = []byte("[]")
	}
	if proxiesJSON == nil {
		proxiesJSON = []byte("[]")
	}
	if configJSON == nil {
		configJSON = []byte("{}")
	}
	_, err := s.db.ExecContext(ctx, `
UPDATE profile_strategies SET name=?, strategy=?, pools=?, proxies=?, config=?, position=?, updated_at=? WHERE id=?`,
		st.NameStr, st.StrategyV,
		string(poolsJSON), string(proxiesJSON), string(configJSON),
		st.Position, st.UpdatedAt, st.ID)
	return err
}

func (s *sqliteStore) DeleteProfileStrategy(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM profile_strategies WHERE id=?`, id)
	return err
}

// ---- ProfileRoutingRules ----

func (s *sqliteStore) CreateProfileRoutingRule(ctx context.Context, rr *model.ProfileRoutingRule) error {
	now := time.Now()
	rr.CreatedAt = now
	rr.UpdatedAt = now
	res, err := s.db.ExecContext(ctx, `
INSERT INTO profile_routing_rules (profile_id, type, payload, target, no_resolve, position, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		rr.ProfileID, rr.Type, rr.Payload, rr.TargetStr,
		boolToInt(rr.NoResolveV), rr.PositionV, now, now)
	if err != nil {
		return err
	}
	rr.ID, _ = res.LastInsertId()
	return nil
}

func (s *sqliteStore) GetProfileRoutingRule(ctx context.Context, id int64) (*model.ProfileRoutingRule, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT id, profile_id, type, payload, target, no_resolve, position, created_at, updated_at
FROM profile_routing_rules WHERE id=?`, id)
	return scanProfileRoutingRule(row)
}

func (s *sqliteStore) ListProfileRoutingRules(ctx context.Context, profileID int64) ([]*model.ProfileRoutingRule, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT id, profile_id, type, payload, target, no_resolve, position, created_at, updated_at
FROM profile_routing_rules WHERE profile_id=? ORDER BY position ASC`, profileID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*model.ProfileRoutingRule
	for rows.Next() {
		rr, err := scanProfileRoutingRule(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, rr)
	}
	return result, rows.Err()
}

func (s *sqliteStore) UpdateProfileRoutingRule(ctx context.Context, rr *model.ProfileRoutingRule) error {
	rr.UpdatedAt = time.Now()
	_, err := s.db.ExecContext(ctx, `
UPDATE profile_routing_rules SET type=?, payload=?, target=?, no_resolve=?, position=?, updated_at=? WHERE id=?`,
		rr.Type, rr.Payload, rr.TargetStr,
		boolToInt(rr.NoResolveV), rr.PositionV, rr.UpdatedAt, rr.ID)
	return err
}

func (s *sqliteStore) DeleteProfileRoutingRule(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM profile_routing_rules WHERE id=?`, id)
	return err
}

// ---- ProviderDefs ----

func (s *sqliteStore) UpsertProviderDefs(ctx context.Context, subscriptionID int64, defs []*model.ProviderDefinition) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `DELETE FROM provider_defs WHERE subscription_id=?`, subscriptionID); err != nil {
		return err
	}

	now := time.Now()
	for _, d := range defs {
		d.CreatedAt = now
		d.UpdatedAt = now
		res, err := tx.ExecContext(ctx, `
INSERT INTO provider_defs (subscription_id, name, kind, type, behavior, url, interval, path, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			subscriptionID, d.Name, d.Kind, d.Type, d.Behavior, d.URL, d.Interval, d.Path, now, now)
		if err != nil {
			return err
		}
		d.ID, _ = res.LastInsertId()
		d.SubscriptionID = subscriptionID
	}
	return tx.Commit()
}

func (s *sqliteStore) ListProviderDefs(ctx context.Context, subscriptionID int64, kind string) ([]*model.ProviderDefinition, error) {
	var conditions []string
	var args []interface{}

	conditions = append(conditions, "subscription_id=?")
	args = append(args, subscriptionID)

	if kind != "" {
		conditions = append(conditions, "kind=?")
		args = append(args, kind)
	}

	query := `SELECT id, subscription_id, name, kind, type, behavior, url, interval, path, created_at, updated_at FROM provider_defs WHERE ` + strings.Join(conditions, " AND ")

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*model.ProviderDefinition
	for rows.Next() {
		d, err := scanProviderDef(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, d)
	}
	return result, rows.Err()
}

func (s *sqliteStore) DeleteProviderDefsBySubscription(ctx context.Context, subscriptionID int64) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM provider_defs WHERE subscription_id=?`, subscriptionID)
	return err
}

// ---- helpers ----

type scanner interface {
	Scan(dest ...any) error
}

func scanCollection(s scanner) (*model.Collection, error) {
	c := &model.Collection{}
	err := s.Scan(&c.ID, &c.Name, &c.ContentType, &c.Description, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return c, nil
}

func scanSubscription(s scanner) (*model.Subscription, error) {
	sub := &model.Subscription{}
	var autoRefresh int
	var lastFetchAt sql.NullTime
	var pgdJSON string
	err := s.Scan(
		&sub.ID, &sub.Name, &sub.URL, &sub.Type,
		&autoRefresh, &sub.RefreshCron,
		&lastFetchAt, &sub.NodeCount, &sub.Status, &sub.ErrorMsg,
		&pgdJSON,
		&sub.CreatedAt, &sub.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	sub.AutoRefresh = autoRefresh != 0
	if lastFetchAt.Valid {
		sub.LastFetchAt = &lastFetchAt.Time
	}
	if pgdJSON != "" && pgdJSON != "{}" {
		var pgd model.ProxyGroupData
		if err := json.Unmarshal([]byte(pgdJSON), &pgd); err == nil {
			sub.ProxyGroupsData = &pgd
		}
	}
	return sub, nil
}

func scanProxy(s scanner) (*model.Proxy, error) {
	p := &model.Proxy{}
	var cfgJSON string
	var subID sql.NullInt64
	var colID sql.NullInt64
	var latency sql.NullInt64
	var alive sql.NullInt64
	var lastCheckAt sql.NullTime
	err := s.Scan(
		&p.ID, &subID, &colID, &p.Name, &p.Type, &p.Server, &p.Port,
		&cfgJSON, &p.Region, &latency, &alive, &lastCheckAt,
		&p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if subID.Valid {
		p.SubscriptionID = &subID.Int64
	}
	if colID.Valid {
		p.CollectionID = &colID.Int64
	}
	if cfgJSON != "" {
		_ = json.Unmarshal([]byte(cfgJSON), &p.Config)
	}
	if latency.Valid {
		v := int(latency.Int64)
		p.Latency = &v
	}
	if alive.Valid {
		v := alive.Int64 != 0
		p.Alive = &v
	}
	if lastCheckAt.Valid {
		p.LastCheckAt = &lastCheckAt.Time
	}
	return p, nil
}

func scanEndpoint(s scanner) (*model.Endpoint, error) {
	e := &model.Endpoint{}
	var filtersJSON string
	var subID sql.NullInt64
	var colID sql.NullInt64
	err := s.Scan(&e.ID, &e.Name, &e.Slug, &subID, &colID, &e.OutputType, &e.Format, &filtersJSON, &e.CreatedAt, &e.UpdatedAt)
	if err != nil {
		return nil, err
	}
	if subID.Valid {
		e.SubscriptionID = &subID.Int64
	}
	if colID.Valid {
		e.CollectionID = &colID.Int64
	}
	_ = json.Unmarshal([]byte(filtersJSON), &e.Filters)
	return e, nil
}

func scanProfile(s scanner) (*model.Profile, error) {
	p := &model.Profile{}
	var settingsJSON sql.NullString
	err := s.Scan(&p.ID, &p.Name, &p.Slug, &settingsJSON,
		&p.NodePoolCount, &p.RuleSetCount, &p.StrategyCount, &p.RoutingRuleCount,
		&p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return nil, err
	}
	if settingsJSON.Valid && settingsJSON.String != "" {
		_ = json.Unmarshal([]byte(settingsJSON.String), &p.Settings)
	}
	if p.Settings == nil {
		p.Settings = make(map[string]any)
	}
	return p, nil
}

func scanProfileNodePool(s scanner) (*model.ProfileNodePool, error) {
	np := &model.ProfileNodePool{}
	var endpointID sql.NullInt64
	err := s.Scan(
		&np.ID, &np.ProfileID, &np.NameStr,
		&endpointID, &np.EndpointSlugValue,
		&np.HealthCheckURL, &np.HealthCheckInterval,
		&np.Position, &np.CreatedAt, &np.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if endpointID.Valid {
		np.EndpointID = &endpointID.Int64
	}
	return np, nil
}

func scanProfileRuleSet(s scanner) (*model.ProfileRuleSet, error) {
	rs := &model.ProfileRuleSet{}
	var endpointID sql.NullInt64
	var metaJSON string
	err := s.Scan(
		&rs.ID, &rs.ProfileID, &rs.NameStr,
		&endpointID, &rs.EndpointSlugValue,
		&rs.ExternalURL, &metaJSON,
		&rs.Interval, &rs.Position, &rs.CreatedAt, &rs.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if endpointID.Valid {
		rs.EndpointID = &endpointID.Int64
	}
	if metaJSON != "" {
		_ = json.Unmarshal([]byte(metaJSON), &rs.MetadataJSON)
	}
	if rs.MetadataJSON == nil {
		rs.MetadataJSON = make(map[string]any)
	}
	return rs, nil
}

func scanProfileStrategy(s scanner) (*model.ProfileStrategy, error) {
	st := &model.ProfileStrategy{}
	var poolsJSON, proxiesJSON, configJSON string
	err := s.Scan(
		&st.ID, &st.ProfileID, &st.NameStr,
		&st.StrategyV, &poolsJSON, &proxiesJSON, &configJSON,
		&st.Position, &st.CreatedAt, &st.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if poolsJSON != "" {
		_ = json.Unmarshal([]byte(poolsJSON), &st.PoolNames)
	}
	if st.PoolNames == nil {
		st.PoolNames = []string{}
	}
	if proxiesJSON != "" {
		_ = json.Unmarshal([]byte(proxiesJSON), &st.ProxyNames)
	}
	if st.ProxyNames == nil {
		st.ProxyNames = []string{}
	}
	if configJSON != "" {
		_ = json.Unmarshal([]byte(configJSON), &st.ConfigJSON)
	}
	if st.ConfigJSON == nil {
		st.ConfigJSON = make(map[string]any)
	}
	return st, nil
}

func scanProfileRoutingRule(s scanner) (*model.ProfileRoutingRule, error) {
	rr := &model.ProfileRoutingRule{}
	var noResolve int
	err := s.Scan(
		&rr.ID, &rr.ProfileID, &rr.Type,
		&rr.Payload, &rr.TargetStr,
		&noResolve, &rr.PositionV,
		&rr.CreatedAt, &rr.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	rr.NoResolveV = noResolve != 0
	return rr, nil
}

func scanProviderDef(s scanner) (*model.ProviderDefinition, error) {
	d := &model.ProviderDefinition{}
	err := s.Scan(
		&d.ID, &d.SubscriptionID, &d.Name,
		&d.Kind, &d.Type, &d.Behavior,
		&d.URL, &d.Interval, &d.Path,
		&d.CreatedAt, &d.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return d, nil
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// marshalProxyGroupsData serialises ProxyGroupsData to a JSON string for storage.
func marshalProxyGroupsData(pgd *model.ProxyGroupData) string {
	if pgd == nil {
		return "{}"
	}
	b, err := json.Marshal(pgd)
	if err != nil {
		return "{}"
	}
	return string(b)
}

// nullableInt64 converts *int64 to a value suitable for SQL — nil becomes nil (NULL).
func nullableInt64(v *int64) interface{} {
	if v == nil {
		return nil
	}
	return *v
}
