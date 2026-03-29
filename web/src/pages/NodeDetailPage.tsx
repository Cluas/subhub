import { useCallback, useEffect, useState } from "react";
import { useParams, Link, useNavigate } from "react-router-dom";
import { ArrowLeft, Loader2, Trash2, Activity } from "lucide-react";
import { useAuth } from "@/context/AuthContext";
import { useToast } from "@/components/ui/toast";
import { fetchProxy, deleteProxy, triggerHealthCheck } from "@/lib/api";
import type { Proxy } from "@/types/api";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { ConfirmDialog } from "@/components/ui/confirm-dialog";

// ── Helpers ───────────────────────────────────────────────────────────────────

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

function maskSecret(value: string | undefined): string {
  if (!value) return "—";
  if (value.length <= 8) return "••••••••";
  return value.slice(0, 4) + "••••••••" + value.slice(-4);
}

// ── Latency bar chart (div-based, no chart library) ───────────────────────────

// The backend has no history endpoint, so we display 7 placeholder bars
// with the current latency filled in the rightmost bar if available.
function LatencyChart({ latency }: { latency: number | null }) {
  const MAX_MS = 300;

  // Build 7 "bars": the last bar uses actual latency, others are empty
  const bars: Array<{ ms: number | null; label: string }> = [
    { ms: null, label: "6d ago" },
    { ms: null, label: "5d ago" },
    { ms: null, label: "4d ago" },
    { ms: null, label: "3d ago" },
    { ms: null, label: "2d ago" },
    { ms: null, label: "Yesterday" },
    { ms: latency, label: "Today" },
  ];

  return (
    <div
      className="rounded-[var(--radius-lg)] border p-5"
      style={{
        backgroundColor: "var(--color-bg-card)",
        borderColor: "var(--color-border)",
        boxShadow: "var(--shadow-card)",
      }}
    >
      <h3
        className="text-[13px] font-semibold mb-4"
        style={{ color: "var(--color-text-muted)" }}
      >
        LATENCY HISTORY (7 DAYS)
      </h3>
      <div className="flex items-end gap-2 h-20">
        {bars.map((bar, i) => {
          const heightPct = bar.ms != null ? Math.min(bar.ms / MAX_MS, 1) * 100 : 0;
          const color =
            bar.ms == null
              ? "var(--color-bg-accent)"
              : bar.ms < 100
              ? "var(--color-success)"
              : bar.ms < 200
              ? "var(--color-warning)"
              : "var(--color-danger)";

          return (
            <div key={i} className="flex flex-col items-center gap-1 flex-1">
              <div className="w-full flex items-end" style={{ height: "60px" }}>
                <div
                  className="w-full rounded-t-sm transition-all"
                  style={{
                    height: bar.ms != null ? `${Math.max(heightPct, 8)}%` : "8%",
                    backgroundColor: color,
                    opacity: bar.ms == null ? 0.3 : 1,
                  }}
                />
              </div>
              {bar.ms != null && (
                <span
                  className="text-[10px] tabular-nums"
                  style={{ color: "var(--color-text-muted)" }}
                >
                  {bar.ms}ms
                </span>
              )}
            </div>
          );
        })}
      </div>
      {latency == null && (
        <p
          className="text-center text-[12px] mt-2"
          style={{ color: "var(--color-text-muted)" }}
        >
          No latency data available — run a health check to populate
        </p>
      )}
    </div>
  );
}

// ── Detail row ────────────────────────────────────────────────────────────────

function DetailRow({ label, value }: { label: string; value: React.ReactNode }) {
  return (
    <div className="flex flex-col gap-0.5">
      <span
        className="text-[11px] font-medium uppercase tracking-wide"
        style={{ color: "var(--color-text-muted)" }}
      >
        {label}
      </span>
      <span
        className="text-[13px] font-mono"
        style={{ color: "var(--color-text-secondary)" }}
      >
        {value}
      </span>
    </div>
  );
}

// ── NodeDetailPage ────────────────────────────────────────────────────────────

export function NodeDetailPage() {
  const { id } = useParams<{ id: string }>();
  const proxyId = Number(id);
  const { token } = useAuth();
  const { showSuccess, showError } = useToast();
  const navigate = useNavigate();

  const [proxy, setProxy] = useState<Proxy | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [pinging, setPinging] = useState(false);
  const [confirmDelete, setConfirmDelete] = useState(false);

  const loadProxy = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const data = await fetchProxy(proxyId, { token });
      setProxy(data);
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : "Failed to load node");
    } finally {
      setLoading(false);
    }
  }, [proxyId, token]);

  useEffect(() => {
    loadProxy();
  }, [loadProxy]);

  async function handlePing() {
    if (!proxy?.subscription_id) {
      showError("Health check only available for subscription nodes");
      return;
    }
    setPinging(true);
    try {
      const result = await triggerHealthCheck(proxy.subscription_id, { token });
      const nodeResult = result.results.find((r) => r.proxy_id === proxyId);
      if (nodeResult) {
        showSuccess(
          nodeResult.alive
            ? `Node alive — ${nodeResult.latency_ms}ms`
            : "Node is unreachable"
        );
        loadProxy();
      } else {
        showSuccess(`Health check complete: ${result.alive}/${result.checked} alive`);
      }
    } catch (err: unknown) {
      showError(err instanceof Error ? err.message : "Health check failed");
    } finally {
      setPinging(false);
    }
  }

  async function handleDelete() {
    setConfirmDelete(false);
    try {
      await deleteProxy(proxyId, { token });
      showSuccess("Node deleted");
      navigate("/nodes");
    } catch (err: unknown) {
      showError(err instanceof Error ? err.message : "Failed to delete node");
    }
  }

  // Extract connection config fields
  const cfg = proxy?.config ?? {};
  const uuid = typeof cfg.uuid === "string" ? cfg.uuid : null;
  const password = typeof cfg.password === "string" ? cfg.password : null;
  const sni = typeof cfg.sni === "string" ? cfg.sni : null;
  const network = typeof cfg.network === "string" ? cfg.network : null;

  return (
    <div className="flex flex-col gap-6">
      {/* Breadcrumb */}
      <div className="flex items-center gap-2 text-[13px]">
        <Link
          to="/nodes"
          className="flex items-center gap-1 font-medium transition-colors hover:underline"
          style={{ color: "var(--color-primary)" }}
        >
          <ArrowLeft className="h-3.5 w-3.5" />
          Nodes
        </Link>
      </div>

      {loading ? (
        <div
          className="flex items-center gap-2 py-8 text-[13px]"
          style={{ color: "var(--color-text-muted)" }}
        >
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
      ) : proxy ? (
        <>
          {/* Page header */}
          <div className="flex flex-wrap items-center gap-3">
            <h1
              className="text-[26px] font-bold leading-tight"
              style={{ color: "var(--color-text-primary)" }}
            >
              {proxy.name}
            </h1>
            <Badge variant={proxy.alive ? "success" : "danger"}>
              {proxy.alive ? "Alive" : "Dead"}
            </Badge>
            <div className="flex items-center gap-2 ml-auto">
              <Button
                size="sm"
                variant="secondary"
                disabled={pinging || !proxy.subscription_id}
                onClick={handlePing}
                title={!proxy.subscription_id ? "Health check requires a subscription node" : undefined}
              >
                {pinging ? (
                  <Loader2 className="h-3.5 w-3.5 animate-spin" />
                ) : (
                  <Activity className="h-3.5 w-3.5" />
                )}
                Ping
              </Button>
              <Button
                size="sm"
                variant="destructive"
                onClick={() => setConfirmDelete(true)}
              >
                <Trash2 className="h-3.5 w-3.5" />
                Delete
              </Button>
            </div>
          </div>

          {/* Two-column grid: Connection Details + right column */}
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
            {/* Connection Details card */}
            <div
              className="rounded-[var(--radius-lg)] border p-5 flex flex-col gap-4"
              style={{
                backgroundColor: "var(--color-bg-card)",
                borderColor: "var(--color-border)",
                boxShadow: "var(--shadow-card)",
              }}
            >
              <h3
                className="text-[13px] font-semibold uppercase tracking-wide"
                style={{ color: "var(--color-text-muted)" }}
              >
                Connection Details
              </h3>
              <div className="flex flex-col gap-3">
                <DetailRow label="Type" value={proxy.type.toUpperCase()} />
                <DetailRow label="Server" value={proxy.server} />
                <DetailRow label="Port" value={proxy.port} />
                {uuid && <DetailRow label="UUID" value={maskSecret(uuid)} />}
                {password && <DetailRow label="Password" value={maskSecret(password)} />}
                {sni && <DetailRow label="SNI" value={sni} />}
                {network && <DetailRow label="Network" value={network} />}
              </div>
            </div>

            {/* Right column: Metadata + Health Status */}
            <div className="flex flex-col gap-4">
              {/* Metadata card */}
              <div
                className="rounded-[var(--radius-lg)] border p-5 flex flex-col gap-4"
                style={{
                  backgroundColor: "var(--color-bg-card)",
                  borderColor: "var(--color-border)",
                  boxShadow: "var(--shadow-card)",
                }}
              >
                <h3
                  className="text-[13px] font-semibold uppercase tracking-wide"
                  style={{ color: "var(--color-text-muted)" }}
                >
                  Metadata
                </h3>
                <div className="flex flex-col gap-3">
                  <DetailRow
                    label="Source"
                    value={
                      proxy.subscription_id
                        ? `Subscription #${proxy.subscription_id}`
                        : proxy.collection_id
                        ? `Collection #${proxy.collection_id}`
                        : "—"
                    }
                  />
                  {proxy.region && <DetailRow label="Region" value={proxy.region} />}
                  <DetailRow label="Updated" value={formatDate(proxy.updated_at)} />
                </div>
              </div>

              {/* Health Status card */}
              <div
                className="rounded-[var(--radius-lg)] border p-5 flex flex-col gap-4"
                style={{
                  backgroundColor: "var(--color-bg-card)",
                  borderColor: "var(--color-border)",
                  boxShadow: "var(--shadow-card)",
                }}
              >
                <h3
                  className="text-[13px] font-semibold uppercase tracking-wide"
                  style={{ color: "var(--color-text-muted)" }}
                >
                  Health Status
                </h3>
                <div className="flex flex-col gap-3">
                  <div className="flex flex-col gap-0.5">
                    <span
                      className="text-[11px] font-medium uppercase tracking-wide"
                      style={{ color: "var(--color-text-muted)" }}
                    >
                      Status
                    </span>
                    <Badge
                      variant={
                        proxy.alive === true
                          ? "success"
                          : proxy.alive === false
                          ? "danger"
                          : "default"
                      }
                    >
                      {proxy.alive === true
                        ? "Alive"
                        : proxy.alive === false
                        ? "Dead"
                        : "Unknown"}
                    </Badge>
                  </div>
                  <DetailRow label="Last Check" value={formatDate(proxy.last_check_at)} />
                  <div className="flex flex-col gap-0.5">
                    <span
                      className="text-[11px] font-medium uppercase tracking-wide"
                      style={{ color: "var(--color-text-muted)" }}
                    >
                      Latency
                    </span>
                    <span
                      className="text-[20px] font-bold tabular-nums"
                      style={{ color: latencyColor(proxy.latency ?? null) }}
                    >
                      {proxy.latency != null ? `${proxy.latency} ms` : "—"}
                    </span>
                  </div>
                </div>
              </div>
            </div>
          </div>

          {/* Latency chart */}
          <LatencyChart latency={proxy.latency ?? null} />

          <ConfirmDialog
            open={confirmDelete}
            title="Delete Node"
            message={`Delete "${proxy.name}"? This action cannot be undone.`}
            confirmLabel="Delete"
            onConfirm={handleDelete}
            onCancel={() => setConfirmDelete(false)}
          />
        </>
      ) : null}
    </div>
  );
}
