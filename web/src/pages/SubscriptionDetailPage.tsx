import { useCallback, useEffect, useMemo, useState } from "react";
import { useParams, Link } from "react-router-dom";
import { ArrowLeft, Loader2, RefreshCw } from "lucide-react";
import { useAuth } from "@/context/AuthContext";
import { useToast } from "@/components/ui/toast";
import {
  fetchSubscription,
  fetchProxies,
  fetchRules,
  fetchEndpoints,
  triggerFetch,
} from "@/lib/api";
import type { Subscription, Proxy, Rule, Endpoint } from "@/types/api";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Table,
  TableHeader,
  TableBody,
  TableRow,
  TableHead,
  TableCell,
} from "@/components/ui/table";

type Tab = "nodes" | "rules" | "endpoints";

function latencyColor(ms: number | null): string {
  if (ms === null) return "var(--color-text-muted)";
  if (ms < 100) return "var(--color-success)";
  if (ms < 200) return "var(--color-warning)";
  return "var(--color-danger)";
}

function formatDate(dt: string | null) {
  if (!dt) return "—";
  return new Date(dt).toLocaleString();
}

export function SubscriptionDetailPage() {
  const { id } = useParams<{ id: string }>();
  const subId = Number(id);
  const { token } = useAuth();
  const { showSuccess, showError } = useToast();

  const [subscription, setSubscription] = useState<Subscription | null>(null);
  const [proxies, setProxies] = useState<Proxy[]>([]);
  const [rules, setRules] = useState<Rule[]>([]);
  const [endpoints, setEndpoints] = useState<Endpoint[]>([]);
  const [tab, setTab] = useState<Tab>("nodes");
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [fetching, setFetching] = useState(false);

  const loadData = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const [sub, nodes, ruleList, endpointList] = await Promise.all([
        fetchSubscription(subId, { token }),
        fetchProxies({ subscription_id: subId }, { token }),
        fetchRules({ subscription_id: subId }, { token }),
        fetchEndpoints({ token }),
      ]);
      setSubscription(sub);
      setProxies(nodes);
      setRules(ruleList);
      setEndpoints(endpointList.filter((e) => e.subscription_id === subId));
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : "Failed to load subscription data");
    } finally {
      setLoading(false);
    }
  }, [subId, token]);

  useEffect(() => {
    loadData();
  }, [loadData]);

  async function handleFetch() {
    setFetching(true);
    try {
      const updated = await triggerFetch(subId, { token });
      setSubscription(updated);
      showSuccess("Subscription fetched successfully");
      loadData();
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : "Failed to fetch subscription";
      showError(msg);
    } finally {
      setFetching(false);
    }
  }

  const tabs: { key: Tab; label: string; count: number }[] = [
    { key: "nodes", label: "Nodes", count: proxies.length },
    { key: "rules", label: "Rules", count: rules.length },
    { key: "endpoints", label: "Endpoints", count: endpoints.length },
  ];

  return (
    <div className="flex flex-col gap-6">
      {/* Breadcrumb */}
      <div className="flex items-center gap-2 text-[13px]">
        <Link
          to="/subscriptions"
          className="flex items-center gap-1 font-medium transition-colors hover:underline"
          style={{ color: "var(--color-primary)" }}
        >
          <ArrowLeft className="h-3.5 w-3.5" />
          Subscriptions
        </Link>
      </div>

      {loading ? (
        <div className="flex items-center gap-2 text-[13px]" style={{ color: "var(--color-text-muted)" }}>
          <Loader2 className="h-4 w-4 animate-spin" />
          Loading…
        </div>
      ) : error ? (
        <div
          className="rounded-[var(--radius-lg)] border px-4 py-3 text-[13px]"
          style={{
            backgroundColor: "var(--color-danger-bg)",
            color: "var(--color-danger)",
            borderColor: "var(--color-danger)",
          }}
        >
          {error}
        </div>
      ) : subscription ? (
        <>
          {/* Subscription header */}
          <div className="flex flex-col gap-2">
            <div className="flex flex-wrap items-center gap-3">
              <h1
                className="text-[26px] font-bold leading-tight"
                style={{ color: "var(--color-text-primary)" }}
              >
                {subscription.name}
              </h1>
              <Badge
                variant={
                  subscription.status === "active"
                    ? "success"
                    : subscription.status === "error"
                    ? "danger"
                    : "warning"
                }
              >
                {subscription.status}
              </Badge>
              <Button
                size="sm"
                variant="secondary"
                disabled={fetching}
                onClick={handleFetch}
              >
                {fetching ? (
                  <>
                    <Loader2 className="h-3.5 w-3.5 animate-spin" />
                    Fetching…
                  </>
                ) : (
                  <>
                    <RefreshCw className="h-3.5 w-3.5" />
                    Fetch
                  </>
                )}
              </Button>
            </div>
            <p
              className="text-[13px] font-mono max-w-[600px] truncate"
              title={subscription.url}
              style={{ color: "var(--color-text-muted)" }}
            >
              {subscription.url}
            </p>
            <p className="text-[12px]" style={{ color: "var(--color-text-muted)" }}>
              Last fetched: {formatDate(subscription.last_fetch_at)} · {subscription.node_count} nodes
            </p>
          </div>

          {/* Tabs */}
          <div
            className="flex gap-0 border-b"
            style={{ borderColor: "var(--color-border)" }}
          >
            {tabs.map(({ key, label, count }) => (
              <button
                key={key}
                onClick={() => setTab(key)}
                className="px-4 py-2.5 text-[13px] font-medium border-b-2 -mb-px transition-colors"
                style={{
                  borderBottomColor:
                    tab === key ? "var(--color-primary)" : "transparent",
                  color:
                    tab === key
                      ? "var(--color-text-primary)"
                      : "var(--color-text-muted)",
                }}
              >
                {label}
                <span
                  className="ml-1.5 rounded-full px-1.5 py-0.5 text-[11px] tabular-nums"
                  style={{
                    backgroundColor: "var(--color-bg-accent)",
                    color: "var(--color-text-muted)",
                  }}
                >
                  {count}
                </span>
              </button>
            ))}
          </div>

          {/* Tab content */}
          {tab === "nodes" && <NodesTab proxies={proxies} />}
          {tab === "rules" && <RulesTab rules={rules} />}
          {tab === "endpoints" && <EndpointsTab endpoints={endpoints} />}
        </>
      ) : null}
    </div>
  );
}

// ── Nodes tab ──────────────────────────────────────────────────────────────

function NodesTab({ proxies }: { proxies: Proxy[] }) {
  if (proxies.length === 0) {
    return (
      <div
        className="rounded-[var(--radius-lg)] border px-6 py-12 text-center text-[13px]"
        style={{ color: "var(--color-text-muted)", borderColor: "var(--color-border)" }}
      >
        No nodes for this subscription.
      </div>
    );
  }
  return (
    <Table>
      <TableHeader>
        <TableRow>
          <TableHead>Name</TableHead>
          <TableHead>Type</TableHead>
          <TableHead>Server</TableHead>
          <TableHead>Port</TableHead>
          <TableHead>Alive</TableHead>
          <TableHead>Latency</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {proxies.map((p) => (
          <TableRow key={p.id}>
            <TableCell className="font-medium" style={{ color: "var(--color-text-primary)" }}>
              {p.name}
            </TableCell>
            <TableCell>
              <Badge>{p.type}</Badge>
            </TableCell>
            <TableCell className="font-mono text-[12px]" style={{ color: "var(--color-text-secondary)" }}>
              {p.server}
            </TableCell>
            <TableCell className="tabular-nums" style={{ color: "var(--color-text-secondary)" }}>
              {p.port}
            </TableCell>
            <TableCell>
              <Badge variant={p.alive ? "success" : "danger"}>
                {p.alive ? "Alive" : "Dead"}
              </Badge>
            </TableCell>
            <TableCell
              className="tabular-nums text-[13px] font-medium"
              style={{ color: latencyColor(p.latency ?? null) }}
            >
              {p.latency != null ? `${p.latency} ms` : "—"}
            </TableCell>
          </TableRow>
        ))}
      </TableBody>
    </Table>
  );
}

// ── Rules tab ──────────────────────────────────────────────────────────────

function RulesTab({ rules }: { rules: Rule[] }) {
  const [filterTarget, setFilterTarget] = useState("");

  const targets = useMemo(() => {
    const set = new Set(rules.map((r) => r.target));
    return Array.from(set).sort();
  }, [rules]);

  const filtered = useMemo(() => {
    if (!filterTarget) return rules;
    return rules.filter((r) => r.target === filterTarget);
  }, [rules, filterTarget]);

  if (rules.length === 0) {
    return (
      <div
        className="rounded-[var(--radius-lg)] border px-6 py-12 text-center text-[13px]"
        style={{ color: "var(--color-text-muted)", borderColor: "var(--color-border)" }}
      >
        No rules for this subscription.
      </div>
    );
  }
  return (
    <div className="flex flex-col gap-3">
      <div className="flex items-center gap-3">
        <select
          value={filterTarget}
          onChange={(e) => setFilterTarget(e.target.value)}
          className="h-9 rounded-[var(--radius)] border px-3 text-[13px] outline-none appearance-none"
          style={{
            borderColor: "var(--color-border)",
            backgroundColor: "var(--color-bg-card)",
            color: "var(--color-text-secondary)",
          }}
          aria-label="Filter by target"
        >
          <option value="">All targets</option>
          {targets.map((t) => (
            <option key={t} value={t}>{t}</option>
          ))}
        </select>
        <span
          className="text-[13px] tabular-nums"
          style={{ color: "var(--color-text-muted)" }}
        >
          {filtered.length} / {rules.length} rules
        </span>
      </div>
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>Type</TableHead>
            <TableHead>Payload</TableHead>
            <TableHead>Target</TableHead>
            <TableHead>Provider</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {filtered.map((r) => (
            <TableRow key={r.id}>
              <TableCell>
                <Badge>{r.type}</Badge>
              </TableCell>
              <TableCell className="font-mono text-[12px]" style={{ color: "var(--color-text-secondary)" }}>
                {r.payload || "—"}
              </TableCell>
              <TableCell className="text-[13px]" style={{ color: "var(--color-text-primary)" }}>
                {r.target}
              </TableCell>
              <TableCell className="text-[12px]" style={{ color: "var(--color-text-muted)" }}>
                {r.provider_name || "—"}
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </div>
  );
}

// ── Endpoints tab ──────────────────────────────────────────────────────────

function EndpointsTab({ endpoints }: { endpoints: Endpoint[] }) {
  if (endpoints.length === 0) {
    return (
      <div
        className="rounded-[var(--radius-lg)] border px-6 py-12 text-center text-[13px]"
        style={{ color: "var(--color-text-muted)", borderColor: "var(--color-border)" }}
      >
        No endpoints linked to this subscription.
      </div>
    );
  }
  return (
    <Table>
      <TableHeader>
        <TableRow>
          <TableHead>Name</TableHead>
          <TableHead>Slug</TableHead>
          <TableHead>Output</TableHead>
          <TableHead>Format</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {endpoints.map((e) => (
          <TableRow key={e.id}>
            <TableCell className="font-medium" style={{ color: "var(--color-text-primary)" }}>
              {e.name}
            </TableCell>
            <TableCell className="font-mono text-[12px]" style={{ color: "var(--color-text-muted)" }}>
              {e.slug}
            </TableCell>
            <TableCell>
              <Badge>{e.output_type}</Badge>
            </TableCell>
            <TableCell>
              <Badge>{e.format}</Badge>
            </TableCell>
          </TableRow>
        ))}
      </TableBody>
    </Table>
  );
}
