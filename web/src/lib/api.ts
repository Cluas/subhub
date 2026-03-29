import type {
  DashboardStats,
  Subscription,
  Proxy,
  HealthCheckResponse,
  CreateSubscriptionInput,
  UpdateSubscriptionInput,
  ProxyFilter,
  CreateProxyInput,
  UpdateProxyInput,
  Endpoint,
  CreateEndpointInput,
  UpdateEndpointInput,
  Rule,
  RuleFilter,
  CreateRuleInput,
  UpdateRuleInput,
  Profile,
  CreateProfileInput,
  UpdateProfileInput,
  ProfileNodePool,
  ProfileRuleSet,
  ProfileStrategy,
  ProfileRoutingRule,
  CreateProfileNodePoolInput,
  CreateProfileRuleSetInput,
  CreateProfileStrategyInput,
  CreateProfileRoutingRuleInput,
  Collection,
  CreateCollectionInput,
  UpdateCollectionInput,
} from "@/types/api";

/**
 * Thin fetch wrapper for SubHub API.
 * In production, the SPA is co-hosted with the Go backend so relative URLs work.
 * In dev, Vite proxies /api/* to http://localhost:9000 (configured in vite.config.ts).
 */

// ── 401 intercept hook ────────────────────────────────────────────────────────
// Auth401Dialog registers itself here on mount so apiFetch can call it
// imperatively without needing React hooks.

type TokenPromptFn = () => Promise<string | null>;
let _promptForToken: TokenPromptFn | null = null;

export function registerTokenPrompt(fn: TokenPromptFn): void {
  _promptForToken = fn;
}

export function unregisterTokenPrompt(): void {
  _promptForToken = null;
}

interface FetchOptions {
  /** Optional Bearer token for authenticated endpoints */
  token?: string;
  /** HTTP method (defaults to GET) */
  method?: string;
  /** Request body (will be JSON-serialised) */
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  body?: any;
}

export async function apiFetch<T>(path: string, opts: FetchOptions = {}): Promise<T> {
  const headers: Record<string, string> = {
    "Content-Type": "application/json",
  };
  if (opts.token) {
    headers["Authorization"] = `Bearer ${opts.token}`;
  }

  const res = await fetch(path, {
    method: opts.method ?? "GET",
    headers,
    body: opts.body !== undefined ? JSON.stringify(opts.body) : undefined,
  });

  if (!res.ok) {
    // 401: attempt to prompt user for a new token, then retry exactly once.
    // The retry is a direct fetch() call — NOT a recursive apiFetch() call.
    if (res.status === 401) {
      const newToken = _promptForToken ? await _promptForToken() : null;
      if (newToken) {
        const retryHeaders = { ...headers, Authorization: `Bearer ${newToken}` };
        const retryRes = await fetch(path, {
          method: opts.method ?? "GET",
          headers: retryHeaders,
          body: opts.body !== undefined ? JSON.stringify(opts.body) : undefined,
        });
        if (retryRes.ok) {
          if (retryRes.status === 204) return undefined as T;
          return retryRes.json() as Promise<T>;
        }
      }
      throw new Error("Unauthorized — please set your API token in Settings");
    }

    let message = `HTTP ${res.status}`;
    try {
      const body = await res.json();
      if (typeof body.error === "string") {
        message = body.error;
      }
    } catch {
      // ignore JSON parse errors on error responses
    }
    throw new Error(message);
  }

  // 204 No Content — nothing to parse
  if (res.status === 204) {
    return undefined as T;
  }

  return res.json() as Promise<T>;
}

/**
 * Fetch aggregate dashboard statistics (subscription and node counts).
 */
export function fetchDashboardStats(opts: FetchOptions = {}): Promise<DashboardStats> {
  return apiFetch<DashboardStats>("/api/dashboard/stats", opts);
}

// ── Subscriptions ────────────────────────────────────────────────────────────

export function fetchSubscriptions(opts: FetchOptions = {}): Promise<Subscription[]> {
  return apiFetch<Subscription[]>("/api/subscriptions", opts);
}

export function fetchSubscription(id: number, opts: FetchOptions = {}): Promise<Subscription> {
  return apiFetch<Subscription>(`/api/subscriptions/${id}`, opts);
}

export function createSubscription(
  data: CreateSubscriptionInput,
  opts: FetchOptions = {}
): Promise<Subscription> {
  return apiFetch<Subscription>("/api/subscriptions", {
    ...opts,
    method: "POST",
    body: data,
  });
}

export function updateSubscription(
  id: number,
  data: UpdateSubscriptionInput,
  opts: FetchOptions = {}
): Promise<Subscription> {
  return apiFetch<Subscription>(`/api/subscriptions/${id}`, {
    ...opts,
    method: "PUT",
    body: data,
  });
}

export function deleteSubscription(id: number, opts: FetchOptions = {}): Promise<void> {
  return apiFetch<void>(`/api/subscriptions/${id}`, {
    ...opts,
    method: "DELETE",
  });
}

export function triggerFetch(id: number, opts: FetchOptions = {}): Promise<Subscription> {
  return apiFetch<Subscription>(`/api/subscriptions/${id}/fetch`, {
    ...opts,
    method: "POST",
  });
}

export function triggerHealthCheck(
  id: number,
  opts: FetchOptions = {}
): Promise<HealthCheckResponse> {
  return apiFetch<HealthCheckResponse>(`/api/subscriptions/${id}/health-check`, {
    ...opts,
    method: "POST",
  });
}

// ── Proxies ──────────────────────────────────────────────────────────────────

export function fetchProxies(
  filter?: ProxyFilter,
  opts: FetchOptions = {}
): Promise<Proxy[]> {
  const params = new URLSearchParams();
  if (filter) {
    if (filter.subscription_id !== undefined) {
      params.set("subscription_id", String(filter.subscription_id));
    }
    if (filter.type) {
      params.set("type", filter.type);
    }
    if (filter.alive !== undefined) {
      params.set("alive", filter.alive ? "true" : "false");
    }
    if (filter.region) {
      params.set("region", filter.region);
    }
  }
  const qs = params.toString();
  return apiFetch<Proxy[]>(qs ? `/api/proxies?${qs}` : "/api/proxies", opts);
}

export function fetchProxy(id: number, opts: FetchOptions = {}): Promise<Proxy> {
  return apiFetch<Proxy>(`/api/proxies/${id}`, opts);
}

export function createProxy(
  data: CreateProxyInput,
  opts: FetchOptions = {}
): Promise<Proxy> {
  return apiFetch<Proxy>("/api/proxies", {
    ...opts,
    method: "POST",
    body: data,
  });
}

export function updateProxy(
  id: number,
  data: UpdateProxyInput,
  opts: FetchOptions = {}
): Promise<Proxy> {
  return apiFetch<Proxy>(`/api/proxies/${id}`, {
    ...opts,
    method: "PUT",
    body: data,
  });
}

export function deleteProxy(id: number, opts: FetchOptions = {}): Promise<void> {
  return apiFetch<void>(`/api/proxies/${id}`, {
    ...opts,
    method: "DELETE",
  });
}

// ── Endpoints ────────────────────────────────────────────────────────────────

export function fetchEndpoints(opts: FetchOptions = {}): Promise<Endpoint[]> {
  return apiFetch<Endpoint[]>("/api/endpoints", opts);
}

export function createEndpoint(
  data: CreateEndpointInput,
  opts: FetchOptions = {}
): Promise<Endpoint> {
  return apiFetch<Endpoint>("/api/endpoints", {
    ...opts,
    method: "POST",
    body: data,
  });
}

export function updateEndpoint(
  id: number,
  data: UpdateEndpointInput,
  opts: FetchOptions = {}
): Promise<Endpoint> {
  return apiFetch<Endpoint>(`/api/endpoints/${id}`, {
    ...opts,
    method: "PUT",
    body: data,
  });
}

export function deleteEndpoint(id: number, opts: FetchOptions = {}): Promise<void> {
  return apiFetch<void>(`/api/endpoints/${id}`, {
    ...opts,
    method: "DELETE",
  });
}

// ── Rules ─────────────────────────────────────────────────────────────────────

export function fetchRules(
  filter?: RuleFilter,
  opts: FetchOptions = {}
): Promise<Rule[]> {
  const params = new URLSearchParams();
  if (filter) {
    if (filter.subscription_id !== undefined) {
      params.set("subscription_id", String(filter.subscription_id));
    }
    if (filter.provider) params.set("provider", filter.provider);
    if (filter.type) params.set("type", filter.type);
    if (filter.target) params.set("target", filter.target);
    if (filter.q) params.set("q", filter.q);
  }
  const qs = params.toString();
  return apiFetch<Rule[]>(qs ? `/api/rules?${qs}` : "/api/rules", opts);
}

export function createRule(
  data: CreateRuleInput,
  opts: FetchOptions = {}
): Promise<Rule> {
  return apiFetch<Rule>("/api/rules", {
    ...opts,
    method: "POST",
    body: data,
  });
}

export function updateRule(
  id: number,
  data: UpdateRuleInput,
  opts: FetchOptions = {}
): Promise<Rule> {
  return apiFetch<Rule>(`/api/rules/${id}`, {
    ...opts,
    method: "PUT",
    body: data,
  });
}

export function deleteRule(id: number, opts: FetchOptions = {}): Promise<void> {
  return apiFetch<void>(`/api/rules/${id}`, {
    ...opts,
    method: "DELETE",
  });
}

// ── Profiles ──────────────────────────────────────────────────────────────────

export function fetchProfiles(opts: FetchOptions = {}): Promise<Profile[]> {
  return apiFetch<Profile[]>("/api/profiles", opts);
}

export function createProfile(
  data: CreateProfileInput,
  opts: FetchOptions = {}
): Promise<Profile> {
  return apiFetch<Profile>("/api/profiles", {
    ...opts,
    method: "POST",
    body: data,
  });
}

export function getProfile(id: number, opts: FetchOptions = {}): Promise<Profile> {
  return apiFetch<Profile>(`/api/profiles/${id}`, opts);
}

export function updateProfile(
  id: number,
  data: UpdateProfileInput,
  opts: FetchOptions = {}
): Promise<Profile> {
  return apiFetch<Profile>(`/api/profiles/${id}`, {
    ...opts,
    method: "PUT",
    body: data,
  });
}

export function deleteProfile(id: number, opts: FetchOptions = {}): Promise<void> {
  return apiFetch<void>(`/api/profiles/${id}`, {
    ...opts,
    method: "DELETE",
  });
}

/**
 * Fetch the rendered Clash YAML for a profile by slug.
 * Hits GET /profile/{slug} — the public (no-auth) endpoint that returns text/yaml.
 * Returns the raw YAML string.
 */
export async function fetchProfileYAML(
  slug: string,
  opts: FetchOptions = {}
): Promise<string> {
  const headers: Record<string, string> = {};
  if (opts.token) {
    headers["Authorization"] = `Bearer ${opts.token}`;
  }

  const res = await fetch(`/profile/${slug}`, {
    method: "GET",
    headers,
  });

  if (!res.ok) {
    let message = `HTTP ${res.status}`;
    try {
      const body = await res.json();
      if (typeof body.error === "string") {
        message = body.error;
      }
    } catch {
      // ignore JSON parse errors
    }
    throw new Error(message);
  }

  return res.text();
}

// ── Collections ───────────────────────────────────────────────────────────────

export function fetchCollections(opts: FetchOptions = {}): Promise<Collection[]> {
  return apiFetch<Collection[]>("/api/collections", opts);
}

export function createCollection(
  data: CreateCollectionInput,
  opts: FetchOptions = {}
): Promise<Collection> {
  return apiFetch<Collection>("/api/collections", {
    ...opts,
    method: "POST",
    body: data,
  });
}

export function updateCollection(
  id: number,
  data: UpdateCollectionInput,
  opts: FetchOptions = {}
): Promise<Collection> {
  return apiFetch<Collection>(`/api/collections/${id}`, {
    ...opts,
    method: "PUT",
    body: data,
  });
}

export function deleteCollection(id: number, opts: FetchOptions = {}): Promise<void> {
  return apiFetch<void>(`/api/collections/${id}`, {
    ...opts,
    method: "DELETE",
  });
}

export function fetchCollectionProxies(
  id: number,
  opts: FetchOptions = {}
): Promise<Proxy[]> {
  return apiFetch<Proxy[]>(`/api/collections/${id}/proxies`, opts);
}

export function fetchCollectionRules(
  id: number,
  opts: FetchOptions = {}
): Promise<Rule[]> {
  return apiFetch<Rule[]>(`/api/collections/${id}/rules`, opts);
}

// ── Profile Sub-resources ─────────────────────────────────────────────────────

export function fetchProfileNodePools(
  profileId: number,
  opts: FetchOptions = {}
): Promise<ProfileNodePool[]> {
  return apiFetch<ProfileNodePool[]>(`/api/profiles/${profileId}/node-pools`, opts);
}

export function createProfileNodePool(
  profileId: number,
  data: CreateProfileNodePoolInput,
  opts: FetchOptions = {}
): Promise<ProfileNodePool> {
  return apiFetch<ProfileNodePool>(`/api/profiles/${profileId}/node-pools`, {
    ...opts,
    method: "POST",
    body: data,
  });
}

export function fetchProfileRuleSets(
  profileId: number,
  opts: FetchOptions = {}
): Promise<ProfileRuleSet[]> {
  return apiFetch<ProfileRuleSet[]>(`/api/profiles/${profileId}/rule-sets`, opts);
}

export function createProfileRuleSet(
  profileId: number,
  data: CreateProfileRuleSetInput,
  opts: FetchOptions = {}
): Promise<ProfileRuleSet> {
  return apiFetch<ProfileRuleSet>(`/api/profiles/${profileId}/rule-sets`, {
    ...opts,
    method: "POST",
    body: data,
  });
}

export function fetchProfileStrategies(
  profileId: number,
  opts: FetchOptions = {}
): Promise<ProfileStrategy[]> {
  return apiFetch<ProfileStrategy[]>(`/api/profiles/${profileId}/strategies`, opts);
}

export function createProfileStrategy(
  profileId: number,
  data: CreateProfileStrategyInput,
  opts: FetchOptions = {}
): Promise<ProfileStrategy> {
  return apiFetch<ProfileStrategy>(`/api/profiles/${profileId}/strategies`, {
    ...opts,
    method: "POST",
    body: data,
  });
}

export function fetchProfileRoutingRules(
  profileId: number,
  opts: FetchOptions = {}
): Promise<ProfileRoutingRule[]> {
  return apiFetch<ProfileRoutingRule[]>(`/api/profiles/${profileId}/routing-rules`, opts);
}

export function createProfileRoutingRule(
  profileId: number,
  data: CreateProfileRoutingRuleInput,
  opts: FetchOptions = {}
): Promise<ProfileRoutingRule> {
  return apiFetch<ProfileRoutingRule>(`/api/profiles/${profileId}/routing-rules`, {
    ...opts,
    method: "POST",
    body: data,
  });
}

export function deleteProfileNodePool(
  profileId: number,
  nodePoolId: number,
  opts: FetchOptions = {}
): Promise<void> {
  return apiFetch<void>(`/api/profiles/${profileId}/node-pools/${nodePoolId}`, {
    ...opts,
    method: "DELETE",
  });
}

export function deleteProfileRuleSet(
  profileId: number,
  ruleSetId: number,
  opts: FetchOptions = {}
): Promise<void> {
  return apiFetch<void>(`/api/profiles/${profileId}/rule-sets/${ruleSetId}`, {
    ...opts,
    method: "DELETE",
  });
}

export function deleteProfileStrategy(
  profileId: number,
  strategyId: number,
  opts: FetchOptions = {}
): Promise<void> {
  return apiFetch<void>(`/api/profiles/${profileId}/strategies/${strategyId}`, {
    ...opts,
    method: "DELETE",
  });
}

export function deleteProfileRoutingRule(
  profileId: number,
  ruleId: number,
  opts: FetchOptions = {}
): Promise<void> {
  return apiFetch<void>(`/api/profiles/${profileId}/routing-rules/${ruleId}`, {
    ...opts,
    method: "DELETE",
  });
}

export function updateProfileSettings(
  id: number,
  settings: Record<string, unknown>,
  opts: FetchOptions = {}
): Promise<Profile> {
  return apiFetch<Profile>(`/api/profiles/${id}`, {
    ...opts,
    method: "PUT",
    body: { settings },
  });
}
