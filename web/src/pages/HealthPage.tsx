import { useEffect, useState } from "react";
import { Loader2 } from "lucide-react";
import {
  fetchDashboardStats,
  fetchSubscriptions,
  triggerHealthCheck,
} from "@/lib/api";
import { useAuth } from "@/context/AuthContext";
import { useToast } from "@/components/ui/toast";
import type { DashboardStats, Subscription, HealthCheckResponse } from "@/types/api";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";

// ── helpers ──────────────────────────────────────────────────────────────────

function aliveRatePct(alive: number, total: number): number {
  if (total === 0) return 0;
  return Math.round((alive / total) * 100);
}

function statusVariant(status: string): "success" | "danger" | "warning" | "default" {
  if (status === "active") return "success";
  if (status === "error") return "danger";
  return "warning";
}

// ── HealthCheckResult — per-subscription inline result ───────────────────────

interface HealthResultRow {
  checked: number;
  alive: number;
  error?: string;
}

// ── HealthPage ────────────────────────────────────────────────────────────────

export function HealthPage() {
  const { token } = useAuth();
  const { showSuccess, showError } = useToast();
  const [stats, setStats] = useState<DashboardStats | null>(null);
  const [subscriptions, setSubscriptions] = useState<Subscription[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Per-subscription check state: id → "running" | HealthResultRow
  const [checkState, setCheckState] = useState<Record<number, "running" | HealthResultRow>>({});

  useEffect(() => {
    let cancelled = false;
    const fetchData = () => {
      setLoading(true);
      setError(null);
      Promise.all([fetchDashboardStats({ token }), fetchSubscriptions({ token })])
        .then(([s, subs]) => {
          if (!cancelled) {
            setStats(s);
            setSubscriptions(subs);
            setLoading(false);
          }
        })
        .catch((err: unknown) => {
          if (!cancelled) {
            setError(err instanceof Error ? err.message : "Failed to load health data");
            setLoading(false);
          }
        });
    };
    fetchData();
    return () => {
      cancelled = true;
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  async function handleCheck(id: number) {
    setCheckState((prev) => ({ ...prev, [id]: "running" }));
    try {
      const result: HealthCheckResponse = await triggerHealthCheck(id, { token });
      setCheckState((prev) => ({
        ...prev,
        [id]: { checked: result.checked, alive: result.alive },
      }));
      showSuccess(`Health check complete — ${result.checked} nodes checked`);
      // Refresh aggregate stats
      fetchDashboardStats({ token })
        .then(setStats)
        .catch(() => {/* non-critical */});
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : "Check failed";
      setCheckState((prev) => ({
        ...prev,
        [id]: { checked: 0, alive: 0, error: msg },
      }));
      showError(msg);
    }
  }

  async function handleRunAllChecks() {
    for (const sub of subscriptions) {
      await handleCheck(sub.id);
    }
  }

  // ── Derived stats ─────────────────────────────────────────────────────────

  const totalNodes = stats?.node_count ?? 0;
  const aliveNodes = stats?.alive_node_count ?? 0;
  const deadNodes = Math.max(0, totalNodes - aliveNodes);

  const summaryStats = [
    { label: "TOTAL NODES", value: String(totalNodes), color: "var(--color-text-primary)" },
    { label: "ALIVE", value: String(aliveNodes), color: "var(--color-success)" },
    { label: "DEAD", value: String(deadNodes), color: "var(--color-danger)" },
    { label: "UNKNOWN", value: "0", color: "var(--color-text-muted)" },
    { label: "AVG LATENCY", value: "—", color: "var(--color-warning)" },
  ];

  // ── Render ────────────────────────────────────────────────────────────────

  return (
    <div className="flex flex-col gap-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-[26px] font-bold text-[var(--color-text-primary)]">Health</h1>
          <p className="text-[13px] text-[var(--color-text-secondary)]">Monitor node health and run diagnostics</p>
        </div>
        <Button onClick={handleRunAllChecks} disabled={loading || subscriptions.length === 0}>
          + Run All Checks
        </Button>
      </div>

      {/* Error banner */}
      {error && (
        <div
          className="rounded-md border px-4 py-3 text-[13px]"
          style={{
            backgroundColor: "var(--color-danger-bg)",
            color: "var(--color-danger)",
            borderColor: "var(--color-danger)",
          }}
        >
          <strong>Error:</strong> {error}
        </div>
      )}

      {/* Summary stat cards */}
      <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-5 gap-4 mb-8">
        {summaryStats.map((stat) => (
          <div
            key={stat.label}
            className="rounded-lg border p-4"
            style={{
              backgroundColor: "var(--color-bg-card)",
              borderColor: "var(--color-border)",
            }}
          >
            <div className="text-[11px] text-[var(--color-text-muted)] uppercase tracking-wide mb-1">
              {stat.label}
            </div>
            {loading ? (
              <div
                className="h-7 w-14 rounded animate-pulse"
                style={{ backgroundColor: "var(--color-bg-accent)" }}
              />
            ) : (
              <div className="text-[22px] font-bold" style={{ color: stat.color }}>
                {stat.value}
              </div>
            )}
          </div>
        ))}
      </div>

      {/* By Subscription section */}
      <h2 className="text-[15px] font-semibold text-[var(--color-text-primary)] mb-4">By Subscription</h2>

      {loading ? (
        <div className="flex flex-col gap-3">
          {[...Array(3)].map((_, i) => (
            <div
              key={i}
              className="h-24 rounded-xl animate-pulse"
              style={{ backgroundColor: "var(--color-bg-card)" }}
            />
          ))}
        </div>
      ) : subscriptions.length === 0 ? (
        <div
          className="rounded-xl border px-6 py-12 text-center text-[13px]"
          style={{
            color: "var(--color-text-muted)",
            borderColor: "var(--color-border)",
            backgroundColor: "var(--color-bg-card)",
          }}
        >
          No subscriptions yet. Create one to monitor node health.
        </div>
      ) : (
        <div className="flex flex-col gap-3">
          {subscriptions.map((sub) => {
            const cs = checkState[sub.id];
            const isRunning = cs === "running";
            const result = cs && cs !== "running" ? (cs as HealthResultRow) : null;

            const alive = result?.alive ?? 0;
            const total = result?.checked ?? sub.node_count;
            const dead = Math.max(0, total - alive);
            const pct = aliveRatePct(alive, total);

            return (
              <div
                key={sub.id}
                className="rounded-xl border p-5"
                style={{
                  backgroundColor: "var(--color-bg-card)",
                  borderColor: "var(--color-border)",
                }}
              >
                {/* Top row */}
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-3">
                    <span className="text-[15px] font-semibold text-[var(--color-text-primary)]">{sub.name}</span>
                    <Badge variant={statusVariant(sub.status)}>{sub.status}</Badge>
                  </div>
                  <Button
                    variant="secondary"
                    size="sm"
                    disabled={isRunning}
                    onClick={() => handleCheck(sub.id)}
                  >
                    {isRunning ? (
                      <><Loader2 className="h-3 w-3 animate-spin" />Checking…</>
                    ) : (
                      "Run Check"
                    )}
                  </Button>
                </div>

                {/* Health bar */}
                <div
                  className="mt-3 h-2 rounded-full overflow-hidden"
                  style={{ backgroundColor: "var(--color-bg-accent)" }}
                >
                  <div
                    className="h-full rounded-full transition-all"
                    style={{
                      width: `${pct}%`,
                      backgroundColor: "var(--color-success)",
                    }}
                  />
                </div>

                {/* Stats row */}
                <div className="mt-2 flex gap-4 text-[12px] text-[var(--color-text-secondary)]">
                  <span style={{ color: "var(--color-success)" }}>{alive} alive</span>
                  <span style={{ color: "var(--color-danger)" }}>{dead} dead</span>
                  <span>0 unknown</span>
                  <span>Avg: —</span>
                  <span className="ml-auto text-[var(--color-text-muted)]">{pct}%</span>
                </div>

                {/* Error message */}
                {result?.error && (
                  <p className="text-[12px] text-[var(--color-danger)] mt-1">{result.error}</p>
                )}
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
}
