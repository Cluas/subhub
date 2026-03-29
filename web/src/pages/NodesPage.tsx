import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { Loader2, Pencil, Trash2 } from "lucide-react";
import { Pagination } from "@/components/ui/pagination";
import {
  fetchSubscriptions,
  fetchProxies,
  createProxy,
  updateProxy,
  deleteProxy,
  triggerHealthCheck,
} from "@/lib/api";
import { useAuth } from "@/context/AuthContext";
import { useToast } from "@/components/ui/toast";
import type { Subscription, Proxy, ProxyFilter, CreateProxyInput } from "@/types/api";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import {
  Table,
  TableHeader,
  TableBody,
  TableRow,
  TableHead,
  TableCell,
} from "@/components/ui/table";
import { ConfirmDialog } from "@/components/ui/confirm-dialog";
import { AddNodeDialog } from "@/components/AddNodeDialog";

// ── Helpers ───────────────────────────────────────────────────────────────────

const PROXY_TYPES = ["ss", "vmess", "vless", "trojan", "hysteria2", "socks5"];

function latencyColor(ms: number | null): string {
  if (ms === null) return "var(--color-text-muted)";
  if (ms < 100) return "var(--color-success)";
  if (ms < 200) return "var(--color-warning)";
  return "var(--color-danger)";
}

// ── Shared select style ───────────────────────────────────────────────────────

const selectClass =
  "h-9 rounded-[var(--radius)] bg-[var(--color-bg-card)] border border-[var(--color-border)] px-3 text-[13px] text-[var(--color-text-secondary)] outline-none focus:border-[var(--color-primary)] appearance-none pr-8";

// ── NodesPage ─────────────────────────────────────────────────────────────────

export function NodesPage() {
  const { token } = useAuth();
  const { showSuccess, showError } = useToast();
  const [proxies, setProxies] = useState<Proxy[]>([]);
  const [subscriptions, setSubscriptions] = useState<Subscription[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [healthChecking, setHealthChecking] = useState(false);
  const [filters, setFilters] = useState<ProxyFilter & { aliveStr?: string }>({});
  const [search, setSearch] = useState("");

  // Pagination
  const [page, setPage] = useState(1);
  const PAGE_SIZE = 50;

  // Dialog state
  const [dialogOpen, setDialogOpen] = useState(false);
  const [editTarget, setEditTarget] = useState<Proxy | null>(null);

  // Confirm-delete state
  const [confirmDelete, setConfirmDelete] = useState<{ id: number; name: string } | null>(null);
  const [rowLoading, setRowLoading] = useState<Record<number, string>>({});

  // ── Data loading ────────────────────────────────────────────────────────────

  useEffect(() => {
    let cancelled = false;
    fetchSubscriptions({ token })
      .then((data) => {
        if (!cancelled) setSubscriptions(data);
      })
      .catch(() => {
        // Non-fatal — filter dropdown stays empty
      });
    return () => {
      cancelled = true;
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  function loadProxies() {
    let cancelled = false;
    setLoading(true);
    setError(null);

    const filter: ProxyFilter = {};
    if (filters.subscription_id !== undefined) filter.subscription_id = filters.subscription_id;
    if (filters.type) filter.type = filters.type;
    if (filters.alive !== undefined) filter.alive = filters.alive;

    fetchProxies(filter, { token })
      .then((data) => {
        if (!cancelled) {
          setProxies(data);
          setLoading(false);
        }
      })
      .catch((err: unknown) => {
        if (!cancelled) {
          setError(err instanceof Error ? err.message : "Failed to load nodes");
          setLoading(false);
        }
      });
    return () => {
      cancelled = true;
    };
  }

  useEffect(() => {
    return loadProxies();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [filters.subscription_id, filters.type, filters.alive]);

  // ── Actions ─────────────────────────────────────────────────────────────────

  async function handleHealthCheck() {
    if (!filters.subscription_id) return;
    setHealthChecking(true);
    setError(null);
    try {
      await triggerHealthCheck(filters.subscription_id, { token });
      const filter: ProxyFilter = {};
      if (filters.subscription_id !== undefined) filter.subscription_id = filters.subscription_id;
      if (filters.type) filter.type = filters.type;
      if (filters.alive !== undefined) filter.alive = filters.alive;
      const data = await fetchProxies(filter, { token });
      setProxies(data);
      showSuccess("Health check complete");
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : "Health check failed";
      setError(msg);
      showError(msg);
    } finally {
      setHealthChecking(false);
    }
  }

  function handleAliveFilter(val: string) {
    if (val === "") {
      setFilters((f) => {
        const next = { ...f };
        delete next.alive;
        return { ...next, aliveStr: "" };
      });
    } else {
      setFilters((f) => ({ ...f, alive: val === "true", aliveStr: val }));
    }
  }

  async function handleDialogSubmit(data: CreateProxyInput) {
    if (editTarget) {
      await updateProxy(editTarget.id, data, { token });
      showSuccess("Node updated successfully");
    } else {
      await createProxy(data, { token });
      showSuccess("Node added successfully");
    }
    setDialogOpen(false);
    setEditTarget(null);
    loadProxies();
  }

  function openAdd() {
    setEditTarget(null);
    setDialogOpen(true);
  }

  function openEdit(proxy: Proxy) {
    setEditTarget(proxy);
    setDialogOpen(true);
  }

  function requestDelete(proxy: Proxy) {
    setConfirmDelete({ id: proxy.id, name: proxy.name });
  }

  async function onConfirmDelete() {
    if (!confirmDelete) return;
    const { id } = confirmDelete;
    setConfirmDelete(null);
    setRowLoading((prev) => ({ ...prev, [id]: "delete" }));
    try {
      await deleteProxy(id, { token });
      setProxies((prev) => prev.filter((p) => p.id !== id));
      showSuccess("Node deleted successfully");
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : "Failed to delete node";
      setError(msg);
      showError(msg);
    } finally {
      setRowLoading((prev) => {
        const next = { ...prev };
        delete next[id];
        return next;
      });
    }
  }

  // ── Derived / pagination ─────────────────────────────────────────────────────

  const filtered = proxies.filter((p) => {
    if (!search) return true;
    const q = search.toLowerCase();
    return (
      p.name.toLowerCase().includes(q) ||
      p.server.toLowerCase().includes(q) ||
      p.type.toLowerCase().includes(q)
    );
  });

  const totalPages = Math.max(1, Math.ceil(filtered.length / PAGE_SIZE));
  const paginated = filtered.slice((page - 1) * PAGE_SIZE, page * PAGE_SIZE);

  // ── Render ───────────────────────────────────────────────────────────────────

  return (
    <div className="flex flex-col gap-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-[26px] font-bold text-[var(--color-text-primary)]">Nodes</h1>
          <p className="text-[13px] text-[var(--color-text-secondary)]">Browse and manage subscription nodes</p>
        </div>
        <Button onClick={() => handleHealthCheck()} disabled={!filters.subscription_id || healthChecking}>
          {healthChecking ? <><Loader2 className="h-4 w-4 animate-spin" />Checking…</> : "+ Health Check"}
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

      {/* Filter bar */}
      <div className="mb-4 flex flex-wrap gap-3">
        {/* Search */}
        <input
          type="text"
          placeholder="Search nodes…"
          value={search}
          onChange={(e) => { setSearch(e.target.value); setPage(1); }}
          className="h-9 flex-1 min-w-[200px] rounded-[var(--radius)] bg-[var(--color-bg-card)] border border-[var(--color-border)] px-3 text-[13px] text-[var(--color-text-secondary)] outline-none focus:border-[var(--color-primary)] transition-colors"
        />

        {/* Subscription select */}
        <select
          className={selectClass}
          value={filters.subscription_id ?? ""}
          onChange={(e) => {
            const val = e.target.value;
            setFilters((f) => {
              const next = { ...f };
              if (val === "") delete next.subscription_id;
              else next.subscription_id = Number(val);
              return next;
            });
            setPage(1);
          }}
        >
          <option value="">All Subscriptions</option>
          {subscriptions.map((s) => (
            <option key={s.id} value={s.id}>{s.name}</option>
          ))}
        </select>

        {/* Type select */}
        <select
          className={selectClass}
          value={filters.type ?? ""}
          onChange={(e) => {
            const val = e.target.value;
            setFilters((f) => {
              const next = { ...f };
              if (val === "") delete next.type;
              else next.type = val;
              return next;
            });
            setPage(1);
          }}
        >
          <option value="">All Types</option>
          {PROXY_TYPES.map((t) => (
            <option key={t} value={t}>{t}</option>
          ))}
        </select>

        {/* Status select */}
        <select
          className={selectClass}
          value={filters.aliveStr ?? ""}
          onChange={(e) => { handleAliveFilter(e.target.value); setPage(1); }}
        >
          <option value="">All Status</option>
          <option value="true">Alive</option>
          <option value="false">Dead</option>
        </select>

        {/* Add node button */}
        <Button variant="secondary" size="sm" onClick={openAdd}>+ Add Node</Button>
      </div>

      {/* Table */}
      {loading ? (
        <div className="flex flex-col gap-2">
          {[...Array(5)].map((_, i) => (
            <div
              key={i}
              className="h-10 rounded-md animate-pulse"
              style={{ backgroundColor: "var(--color-bg-card)" }}
            />
          ))}
        </div>
      ) : filtered.length === 0 ? (
        <div
          className="rounded-xl border px-6 py-12 text-center text-[13px]"
          style={{
            color: "var(--color-text-muted)",
            borderColor: "var(--color-border)",
            backgroundColor: "var(--color-bg-card)",
          }}
        >
          No nodes found.{" "}
          <button
            onClick={openAdd}
            className="underline hover:no-underline"
            style={{ color: "var(--color-primary)" }}
          >
            Add a self-managed node
          </button>{" "}
          or fetch a subscription to import nodes.
        </div>
      ) : (
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>NAME</TableHead>
              <TableHead>TYPE</TableHead>
              <TableHead>SERVER</TableHead>
              <TableHead>LATENCY</TableHead>
              <TableHead>STATUS</TableHead>
              <TableHead>SUBSCRIPTION</TableHead>
              <TableHead>ACTIONS</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {paginated.map((proxy) => {
              const sub = subscriptions.find((s) => s.id === proxy.subscription_id);
              const busy = rowLoading[proxy.id];
              return (
                <TableRow key={proxy.id}>
                  <TableCell className="font-medium max-w-[160px] truncate" title={proxy.name}>
                    <Link
                      to={`/nodes/${proxy.id}`}
                      className="hover:underline transition-colors"
                      style={{ color: "var(--color-text-primary)" }}
                    >
                      {proxy.name}
                    </Link>
                  </TableCell>
                  <TableCell>
                    <Badge variant="type">{proxy.type}</Badge>
                  </TableCell>
                  <TableCell className="font-mono text-xs text-[var(--color-text-secondary)]">
                    {proxy.server}:{proxy.port}
                  </TableCell>
                  <TableCell style={{ color: latencyColor(proxy.latency), fontWeight: 600 }}>
                    {proxy.latency !== null ? `${proxy.latency}ms` : "—"}
                  </TableCell>
                  <TableCell>
                    <Badge variant={proxy.alive ? "success" : proxy.alive === null ? "default" : "danger"}>
                      {proxy.alive ? "alive" : proxy.alive === null ? "untested" : "dead"}
                    </Badge>
                  </TableCell>
                  <TableCell className="text-[13px] text-[var(--color-text-secondary)]">
                    {proxy.subscription_id === null ? (
                      <Badge>Self-managed</Badge>
                    ) : (
                      sub?.name ?? `#${proxy.subscription_id}`
                    )}
                  </TableCell>
                  <TableCell>
                    <div className="flex gap-1.5">
                      <button
                        title="Edit node"
                        disabled={!!busy}
                        onClick={() => openEdit(proxy)}
                        className="rounded-md p-1.5 hover:bg-[var(--color-bg-accent)] transition-colors disabled:opacity-40"
                        aria-label={`Edit ${proxy.name}`}
                      >
                        <Pencil className="h-3.5 w-3.5 text-[var(--color-text-muted)]" />
                      </button>
                      <button
                        title="Delete node"
                        disabled={!!busy}
                        onClick={() => requestDelete(proxy)}
                        className="rounded-md p-1.5 hover:bg-[var(--color-danger-bg)] transition-colors disabled:opacity-40"
                        aria-label={`Delete ${proxy.name}`}
                      >
                        {busy === "delete" ? (
                          <Loader2 className="h-3.5 w-3.5 animate-spin text-[var(--color-text-muted)]" />
                        ) : (
                          <Trash2 className="h-3.5 w-3.5 text-[var(--color-danger)]" />
                        )}
                      </button>
                    </div>
                  </TableCell>
                </TableRow>
              );
            })}
          </TableBody>
        </Table>
      )}

      {/* Pagination */}
      {!loading && (
        <Pagination
          currentPage={page}
          totalPages={totalPages}
          totalItems={filtered.length}
          onPageChange={setPage}
        />
      )}

      {/* Add / Edit dialog */}
      <AddNodeDialog
        open={dialogOpen}
        editProxy={editTarget}
        onSubmit={handleDialogSubmit}
        onCancel={() => {
          setDialogOpen(false);
          setEditTarget(null);
        }}
      />

      {/* Delete confirmation dialog */}
      <ConfirmDialog
        open={!!confirmDelete}
        title="Delete Node"
        message={
          confirmDelete ? (
            <>
              Delete node <strong>{confirmDelete.name}</strong>? This cannot be undone.
            </>
          ) : null
        }
        onConfirm={onConfirmDelete}
        onCancel={() => setConfirmDelete(null)}
      />
    </div>
  );
}
