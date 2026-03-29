import { useEffect, useState } from "react";
import { fetchDashboardStats, fetchSubscriptions, fetchEndpoints } from "@/lib/api";
import { useAuth } from "@/context/AuthContext";
import { Badge } from "@/components/ui/badge";
import {
  Table,
  TableHeader,
  TableBody,
  TableRow,
  TableHead,
  TableCell,
} from "@/components/ui/table";
import type { DashboardStats, Subscription, Endpoint } from "@/types/api";

// ──── Helpers ────────────────────────────────────────────────────────────────

function statusVariant(status: string): "success" | "danger" | "warning" {
  if (status === "active") return "success";
  if (status === "error") return "danger";
  return "warning";
}

function formatLastFetched(date: string | null): string {
  if (!date) return "Never";
  const d = new Date(date);
  const now = Date.now();
  const diff = now - d.getTime();
  const minutes = Math.floor(diff / 60_000);
  if (minutes < 1) return "Just now";
  if (minutes < 60) return `${minutes}m ago`;
  const hours = Math.floor(minutes / 60);
  if (hours < 24) return `${hours}h ago`;
  const days = Math.floor(hours / 24);
  return `${days}d ago`;
}

// ──── StatCard ───────────────────────────────────────────────────────────────

interface StatCardProps {
  label: string;
  value: number | string;
  subtitle: string;
  accentColor: string;
  loading?: boolean;
}

function StatCard({ label, value, subtitle, accentColor, loading }: StatCardProps) {
  return (
    <div
      className="relative overflow-hidden flex flex-col gap-2 p-5 rounded-[var(--radius-lg)] border border-[var(--color-border)]/50 shadow-[var(--shadow-card)]"
      style={{ backgroundColor: "var(--color-bg-card)" }}
    >
      {/* Accent bar */}
      <div
        className="absolute top-0 left-0 right-0 h-[3px] rounded-t-[var(--radius-lg)]"
        style={{ backgroundColor: accentColor }}
      />
      {/* Label */}
      <span
        className="text-[11px] font-medium uppercase tracking-wide"
        style={{ color: "var(--color-text-muted)" }}
      >
        {label}
      </span>
      {/* Value */}
      {loading ? (
        <div
          className="h-8 w-20 rounded animate-pulse"
          style={{ backgroundColor: "var(--color-bg-accent)" }}
        />
      ) : (
        <span
          className="text-[28px] font-bold leading-none mt-1"
          style={{ color: "var(--color-text-primary)" }}
        >
          {value}
        </span>
      )}
      {/* Subtitle */}
      <span
        className="text-[12px]"
        style={{ color: "var(--color-text-secondary)" }}
      >
        {loading ? (
          <span
            className="inline-block h-3 w-16 rounded animate-pulse"
            style={{ backgroundColor: "var(--color-bg-accent)" }}
          />
        ) : (
          subtitle
        )}
      </span>
    </div>
  );
}

// ──── Skeleton StatCard ──────────────────────────────────────────────────────

function StatCardSkeleton({ accentColor }: { accentColor: string }) {
  return (
    <div
      className="relative overflow-hidden flex flex-col gap-2 p-5 rounded-[var(--radius-lg)] border border-[var(--color-border)]/50 shadow-[var(--shadow-card)]"
      style={{ backgroundColor: "var(--color-bg-card)" }}
    >
      <div
        className="absolute top-0 left-0 right-0 h-[3px] rounded-t-[var(--radius-lg)]"
        style={{ backgroundColor: accentColor }}
      />
      <div
        className="h-3 w-24 rounded animate-pulse"
        style={{ backgroundColor: "var(--color-bg-accent)" }}
      />
      <div
        className="h-8 w-20 rounded animate-pulse mt-1"
        style={{ backgroundColor: "var(--color-bg-accent)" }}
      />
      <div
        className="h-3 w-16 rounded animate-pulse"
        style={{ backgroundColor: "var(--color-bg-accent)" }}
      />
    </div>
  );
}

// ──── Node Health Donut ──────────────────────────────────────────────────────

interface NodeHealthProps {
  totalNodes: number;
  aliveNodes: number;
}

function NodeHealth({ totalNodes, aliveNodes }: NodeHealthProps) {
  const deadNodes = Math.max(0, totalNodes - aliveNodes);
  const unknownNodes = 0;
  const aliveRate = totalNodes > 0 ? aliveNodes / totalNodes : 0;
  const alivePercent = Math.round(aliveRate * 100);

  // SVG donut ring: r=40, cx=cy=52, circumference = 2πr ≈ 251.2
  const r = 40;
  const cx = 52;
  const cy = 52;
  const circumference = 2 * Math.PI * r;
  const strokeDashoffset = circumference * (1 - aliveRate);

  return (
    <div
      className="rounded-[var(--radius-lg)] border border-[var(--color-border)]/50 shadow-[var(--shadow-card)] p-5 flex flex-col gap-4"
      style={{ backgroundColor: "var(--color-bg-card)" }}
    >
      <h2
        className="text-[15px] font-semibold"
        style={{ color: "var(--color-text-primary)" }}
      >
        Node Health
      </h2>

      {/* Desktop/tablet: donut ring */}
      <div className="hidden sm:flex flex-col items-center gap-4">
        <svg width="104" height="104" viewBox="0 0 104 104">
          {/* Background circle */}
          <circle
            cx={cx}
            cy={cy}
            r={r}
            fill="none"
            stroke="var(--color-bg-accent)"
            strokeWidth="12"
          />
          {/* Foreground arc */}
          <circle
            cx={cx}
            cy={cy}
            r={r}
            fill="none"
            stroke="var(--color-success)"
            strokeWidth="12"
            strokeLinecap="round"
            strokeDasharray={circumference}
            strokeDashoffset={strokeDashoffset}
            transform={`rotate(-90 ${cx} ${cy})`}
            style={{ transition: "stroke-dashoffset 0.6s ease" }}
          />
          {/* Center text */}
          <text
            x={cx}
            y={cy + 1}
            textAnchor="middle"
            dominantBaseline="middle"
            fontSize="16"
            fontWeight="700"
            fill="var(--color-text-primary)"
          >
            {alivePercent}%
          </text>
        </svg>

        {/* Legend */}
        <div className="w-full flex flex-col gap-2">
          <LegendRow color="var(--color-success)" label="Alive Nodes" count={aliveNodes} />
          <LegendRow color="var(--color-danger)" label="Dead Nodes" count={deadNodes} />
          <LegendRow color="var(--color-text-muted)" label="Unknown" count={unknownNodes} />
        </div>
      </div>

      {/* Mobile: progress bar */}
      <div className="flex sm:hidden flex-col gap-2">
        <div
          className="h-2 rounded-full w-full"
          style={{ backgroundColor: "var(--color-bg-accent)" }}
        >
          <div
            className="h-2 rounded-full transition-all duration-500"
            style={{
              backgroundColor: "var(--color-success)",
              width: `${alivePercent}%`,
            }}
          />
        </div>
        <span
          className="text-[12px]"
          style={{ color: "var(--color-text-secondary)" }}
        >
          {alivePercent}% alive &mdash; {aliveNodes}/{totalNodes} nodes
        </span>
      </div>
    </div>
  );
}

function LegendRow({
  color,
  label,
  count,
}: {
  color: string;
  label: string;
  count: number;
}) {
  return (
    <div className="flex items-center gap-2">
      <span
        className="inline-block w-2 h-2 rounded-full flex-shrink-0"
        style={{ backgroundColor: color }}
      />
      <span
        className="text-[12px] flex-1"
        style={{ color: "var(--color-text-secondary)" }}
      >
        {label}
      </span>
      <span
        className="text-[12px] font-semibold tabular-nums"
        style={{ color: "var(--color-text-primary)" }}
      >
        {count}
      </span>
    </div>
  );
}

// ──── Recent Subscriptions ───────────────────────────────────────────────────

interface RecentSubscriptionsProps {
  subscriptions: Subscription[];
}

function RecentSubscriptions({ subscriptions }: RecentSubscriptionsProps) {
  const recent = subscriptions.slice(0, 8);

  return (
    <div className="flex flex-col gap-3">
      <h2
        className="text-[15px] font-semibold"
        style={{ color: "var(--color-text-primary)" }}
      >
        Recent Subscriptions
      </h2>

      {/* Desktop table */}
      <div className="hidden lg:block">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Name</TableHead>
              <TableHead>Type</TableHead>
              <TableHead>Nodes</TableHead>
              <TableHead>Status</TableHead>
              <TableHead>Last Fetched</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {recent.length === 0 ? (
              <TableRow>
                <TableCell
                  colSpan={5}
                  className="text-center py-8"
                  style={{ color: "var(--color-text-muted)" }}
                >
                  No subscriptions yet
                </TableCell>
              </TableRow>
            ) : (
              recent.map((sub) => (
                <TableRow key={sub.id}>
                  <TableCell
                    className="font-medium"
                    style={{ color: "var(--color-text-primary)" }}
                  >
                    {sub.name}
                  </TableCell>
                  <TableCell>
                    <Badge variant="type">{sub.type}</Badge>
                  </TableCell>
                  <TableCell style={{ color: "var(--color-text-secondary)" }}>
                    {sub.node_count}
                  </TableCell>
                  <TableCell>
                    <Badge variant={statusVariant(sub.status)}>{sub.status}</Badge>
                  </TableCell>
                  <TableCell
                    className="text-[11px]"
                    style={{ color: "var(--color-text-muted)" }}
                  >
                    {formatLastFetched(sub.last_fetch_at)}
                  </TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </div>

      {/* Mobile stacked list */}
      <div className="flex lg:hidden flex-col gap-2">
        {recent.length === 0 ? (
          <p
            className="text-[13px] py-4 text-center"
            style={{ color: "var(--color-text-muted)" }}
          >
            No subscriptions yet
          </p>
        ) : (
          recent.map((sub) => (
            <div
              key={sub.id}
              className="flex flex-col gap-1.5 p-3 rounded-[var(--radius)] border border-[var(--color-border)]/50"
              style={{ backgroundColor: "var(--color-bg-card)" }}
            >
              <div className="flex items-center justify-between gap-2">
                <span
                  className="text-[13px] font-medium truncate"
                  style={{ color: "var(--color-text-primary)" }}
                >
                  {sub.name}
                </span>
                <Badge variant={statusVariant(sub.status)}>{sub.status}</Badge>
              </div>
              <div className="flex items-center gap-2">
                <Badge variant="type">{sub.type}</Badge>
                <span
                  className="text-[11px]"
                  style={{ color: "var(--color-text-muted)" }}
                >
                  {sub.node_count} nodes &bull; {formatLastFetched(sub.last_fetch_at)}
                </span>
              </div>
            </div>
          ))
        )}
      </div>
    </div>
  );
}

// ──── DashboardPage ──────────────────────────────────────────────────────────

export function DashboardPage() {
  const { token } = useAuth();
  const [stats, setStats] = useState<DashboardStats | null>(null);
  const [subscriptions, setSubscriptions] = useState<Subscription[]>([]);
  const [endpointCount, setEndpointCount] = useState<number>(0);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;

    Promise.all([
      fetchDashboardStats({ token }),
      fetchSubscriptions({ token }),
      fetchEndpoints({ token }),
    ])
      .then(([statsData, subsData, endpointsData]: [DashboardStats, Subscription[], Endpoint[]]) => {
        if (!cancelled) {
          setStats(statsData);
          setSubscriptions(subsData);
          setEndpointCount(endpointsData.length);
          setLoading(false);
        }
      })
      .catch((err: unknown) => {
        if (!cancelled) {
          setError(err instanceof Error ? err.message : "Failed to load stats");
          setLoading(false);
        }
      });

    return () => {
      cancelled = true;
    };
  }, [token]);

  const aliveRate =
    stats && stats.node_count > 0
      ? stats.alive_node_count / stats.node_count
      : 0;

  const accentColors = [
    "var(--color-primary)",
    "var(--color-success)",
    "var(--color-warning)",
    "var(--color-danger)",
  ];

  return (
    <div className="flex flex-col gap-6">
      {/* Page header */}
      <div>
        <h1
          className="text-[26px] font-bold"
          style={{ color: "var(--color-text-primary)" }}
        >
          Dashboard
        </h1>
        <p
          className="text-[13px] mt-1"
          style={{ color: "var(--color-text-secondary)" }}
        >
          Monitor your proxy infrastructure
        </p>
      </div>

      {/* Error state */}
      {error && (
        <div
          className="rounded-[var(--radius)] border px-4 py-3 text-[13px]"
          style={{
            backgroundColor: "var(--color-danger-bg)",
            color: "var(--color-danger)",
            borderColor: "var(--color-danger)",
          }}
        >
          <strong>Error loading stats:</strong> {error}
        </div>
      )}

      {/* Stat cards — 2 columns mobile, 4 desktop */}
      <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
        {loading ? (
          accentColors.map((color, i) => (
            <StatCardSkeleton key={i} accentColor={color} />
          ))
        ) : (
          <>
            <StatCard
              label="Subscriptions"
              value={stats?.subscription_count ?? 0}
              subtitle="Active sources"
              accentColor="var(--color-primary)"
            />
            <StatCard
              label="Active Nodes"
              value={stats?.alive_node_count ?? 0}
              subtitle={`${Math.round(aliveRate * 100)}% alive`}
              accentColor="var(--color-success)"
            />
            <StatCard
              label="Endpoints"
              value={endpointCount}
              subtitle="Formats"
              accentColor="var(--color-warning)"
            />
            <StatCard
              label="Total Nodes"
              value={stats?.node_count ?? 0}
              subtitle={`${stats?.active_subscription_count ?? 0} active subs`}
              accentColor="var(--color-danger)"
            />
          </>
        )}
      </div>

      {/* Bottom section: table + node health */}
      <div className="grid grid-cols-1 lg:grid-cols-[1fr_280px] gap-6">
        {/* Recent Subscriptions */}
        {loading ? (
          <div
            className="rounded-[var(--radius-lg)] border border-[var(--color-border)]/50 p-5 flex flex-col gap-3"
            style={{ backgroundColor: "var(--color-bg-card)" }}
          >
            <div
              className="h-4 w-40 rounded animate-pulse"
              style={{ backgroundColor: "var(--color-bg-accent)" }}
            />
            {[...Array(4)].map((_, i) => (
              <div
                key={i}
                className="h-8 rounded animate-pulse"
                style={{ backgroundColor: "var(--color-bg-accent)" }}
              />
            ))}
          </div>
        ) : (
          <div
            className="rounded-[var(--radius-lg)] border border-[var(--color-border)]/50 p-5"
            style={{ backgroundColor: "var(--color-bg-card)" }}
          >
            <RecentSubscriptions subscriptions={subscriptions} />
          </div>
        )}

        {/* Node Health */}
        {loading ? (
          <div
            className="rounded-[var(--radius-lg)] border border-[var(--color-border)]/50 p-5 flex flex-col gap-4"
            style={{ backgroundColor: "var(--color-bg-card)" }}
          >
            <div
              className="h-4 w-28 rounded animate-pulse"
              style={{ backgroundColor: "var(--color-bg-accent)" }}
            />
            <div className="flex justify-center">
              <div
                className="w-24 h-24 rounded-full animate-pulse"
                style={{ backgroundColor: "var(--color-bg-accent)" }}
              />
            </div>
          </div>
        ) : (
          <NodeHealth
            totalNodes={stats?.node_count ?? 0}
            aliveNodes={stats?.alive_node_count ?? 0}
          />
        )}
      </div>
    </div>
  );
}
