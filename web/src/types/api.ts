// API response type definitions

export interface DashboardStats {
  subscription_count: number;
  active_subscription_count: number;
  node_count: number;
  alive_node_count: number;
}

export interface Subscription {
  id: number;
  name: string;
  url: string;
  type: string;
  auto_refresh: boolean;
  refresh_cron: string;
  last_fetch_at: string | null;
  node_count: number;
  status: string;
  error_msg: string | null;
  created_at: string;
  updated_at: string;
}

export interface Proxy {
  id: number;
  subscription_id: number | null;
  collection_id: number | null;
  name: string;
  type: string;
  server: string;
  port: number;
  config: Record<string, unknown> | null;
  region: string | null;
  latency: number | null;
  alive: boolean | null;
  last_check_at: string | null;
  created_at: string;
  updated_at: string;
}

export interface CreateProxyInput {
  name: string;
  type: string;
  server: string;
  port: number;
  region?: string;
  config?: Record<string, unknown>;
  subscription_id?: number | null;
  collection_id?: number | null;
}

export interface UpdateProxyInput {
  name?: string;
  type?: string;
  server?: string;
  port?: number;
  region?: string;
  config?: Record<string, unknown>;
}

export interface HealthCheckResponse {
  checked: number;
  alive: number;
  results: Array<{
    proxy_id: number;
    alive: boolean;
    latency_ms: number;
  }>;
}

export interface CreateSubscriptionInput {
  name: string;
  url: string;
  type: string;
  auto_refresh: boolean;
  refresh_cron: string;
}

export interface UpdateSubscriptionInput {
  name?: string;
  url?: string;
  type?: string;
  auto_refresh?: boolean;
  refresh_cron?: string;
}

export interface ProxyFilter {
  subscription_id?: number;
  type?: string;
  alive?: boolean;
  region?: string;
}

// ── Endpoints ────────────────────────────────────────────────────────────────

export interface EndpointFilters {
  regions?: string[];
  types?: string[];
  groups?: string[];
  name_contains?: string;
  latency_max?: number;
  alive_only?: boolean;
  target?: string;
  source?: string;
  rule_type?: string;
  keyword?: string;
}

export interface Endpoint {
  id: number;
  name: string;
  slug: string;
  subscription_id: number | null;
  collection_id: number | null;
  output_type: string;  // "proxy" | "rule"
  format: string;       // "clash" | "surge" | "quantumult-x"
  filters: EndpointFilters;
  created_at: string;
  updated_at: string;
}

export interface CreateEndpointInput {
  name: string;
  slug?: string;
  subscription_id?: number | null;
  collection_id?: number | null;
  output_type: string;
  format: string;
  filters: EndpointFilters;
}

export interface UpdateEndpointInput {
  name?: string;
  slug?: string;
  output_type?: string;
  format?: string;
  subscription_id?: number | null;
  collection_id?: number | null;
  filters?: EndpointFilters;
}

// ── Rules ─────────────────────────────────────────────────────────────────────

export interface Rule {
  id: number;
  subscription_id: number | null;
  collection_id: number | null;
  provider_name: string;
  type: string;
  payload: string;
  target: string;
}

export interface CreateRuleInput {
  type: string;
  payload: string;
  target: string;
  provider_name?: string;
  subscription_id?: number | null;
  collection_id?: number | null;
}

export interface UpdateRuleInput {
  type?: string;
  payload?: string;
  target?: string;
  provider_name?: string;
}

export interface RuleFilter {
  subscription_id?: number;
  provider?: string;
  type?: string;
  target?: string;
  q?: string;
}

// ── Profiles ──────────────────────────────────────────────────────────────────

export interface Profile {
  id: number;
  name: string;
  slug: string;
  settings: Record<string, unknown> | null;
  node_pool_count: number;
  rule_set_count: number;
  strategy_count: number;
  routing_rule_count: number;
  created_at: string;
  updated_at: string;
}

export interface ProfileNodePool {
  id: number;
  profile_id: number;
  name: string;
  endpoint_slug: string;
  health_check_url: string;
  health_check_interval: number;
  position: number;
  created_at: string;
  updated_at: string;
}

export interface ProfileRuleSet {
  id: number;
  profile_id: number;
  name: string;
  endpoint_slug: string;
  url: string;
  metadata: Record<string, unknown> | null;
  interval: number;
  position: number;
  created_at: string;
  updated_at: string;
}

export interface ProfileStrategy {
  id: number;
  profile_id: number;
  name: string;
  strategy: string;
  pools: string[];
  proxies: string[];
  config: Record<string, unknown> | null;
  position: number;
  created_at: string;
  updated_at: string;
}

export interface ProfileRoutingRule {
  id: number;
  profile_id: number;
  type: string;
  payload: string;
  target: string;
  no_resolve: boolean;
  position: number;
  created_at: string;
  updated_at: string;
}

export interface CreateProfileNodePoolInput {
  name: string;
  endpoint_id?: number;
  endpoint_slug?: string;
  health_check_url?: string;
  health_check_interval?: number;
  position?: number;
}

export interface CreateProfileRuleSetInput {
  name: string;
  endpoint_slug?: string;
  url?: string;
  metadata?: Record<string, unknown>;
}

export interface CreateProfileStrategyInput {
  name: string;
  strategy?: string;
  pools?: string[];
  proxies?: string[];
  config?: Record<string, unknown>;
  position?: number;
}

export interface CreateProfileRoutingRuleInput {
  match: string;
  value?: string;
  target: string;
  position?: number;
  no_resolve?: boolean;
}

export interface CreateProfileInput {
  name: string;
  settings?: Record<string, unknown>;
}

export interface UpdateProfileInput {
  name?: string;
  slug?: string;
  settings?: Record<string, unknown>;
}

// ── Collections ───────────────────────────────────────────────────────────────

export interface Collection {
  id: number;
  name: string;
  content_type: "proxy" | "rule";
  description: string;
  created_at: string;
  updated_at: string;
}

export interface CreateCollectionInput {
  name: string;
  content_type: "proxy" | "rule";
  description?: string;
}

export interface UpdateCollectionInput {
  name?: string;
  description?: string;
}
