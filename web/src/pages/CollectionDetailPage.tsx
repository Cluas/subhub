import { useCallback, useEffect, useState } from "react";
import { useParams, Link } from "react-router-dom";
import { ArrowLeft, Loader2, Plus, Trash2, Pencil } from "lucide-react";
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
import { useToast } from "@/components/ui/toast";
import { useAuth } from "@/context/AuthContext";
import {
  fetchCollections,
  fetchCollectionProxies,
  fetchCollectionRules,
  createProxy,
  updateProxy,
  deleteProxy,
  createRule,
  deleteRule,
} from "@/lib/api";
import type {
  Collection,
  CreateProxyInput,
  CreateRuleInput,
  Proxy,
  Rule,
} from "@/types/api";
import { AddNodeDialog } from "@/components/AddNodeDialog";
import { AddRuleDialog } from "@/components/AddRuleDialog";

export function CollectionDetailPage() {
  const { id } = useParams<{ id: string }>();
  const collectionId = Number(id);
  const { token } = useAuth();
  const { showSuccess, showError } = useToast();

  const [collection, setCollection] = useState<Collection | null>(null);
  const [proxies, setProxies] = useState<Proxy[]>([]);
  const [rules, setRules] = useState<Rule[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Node dialog
  const [nodeDialogOpen, setNodeDialogOpen] = useState(false);
  const [editProxy, setEditProxy] = useState<Proxy | null>(null);
  const [confirmDeleteProxy, setConfirmDeleteProxy] = useState<{ id: number; name: string } | null>(null);

  // Rule dialog
  const [ruleDialogOpen, setRuleDialogOpen] = useState(false);
  const [editRule, setEditRule] = useState<Rule | null>(null);
  const [confirmDeleteRule, setConfirmDeleteRule] = useState<{ id: number; name: string } | null>(null);

  const loadData = useCallback(() => {
    let cancelled = false;
    setLoading(true);
    setError(null);

    fetchCollections({ token })
      .then((cols) => {
        const col = cols.find((c) => c.id === collectionId);
        if (!cancelled) setCollection(col ?? null);
        if (col?.content_type === "proxy") {
          return fetchCollectionProxies(collectionId, { token }).then((data) => {
            if (!cancelled) { setProxies(data); setLoading(false); }
          });
        } else if (col?.content_type === "rule") {
          return fetchCollectionRules(collectionId, { token }).then((data) => {
            if (!cancelled) { setRules(data); setLoading(false); }
          });
        } else {
          if (!cancelled) setLoading(false);
        }
      })
      .catch((err: unknown) => {
        if (!cancelled) {
          setError(err instanceof Error ? err.message : "Failed to load collection");
          setLoading(false);
        }
      });

    return () => { cancelled = true; };
  }, [token, collectionId]);

  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect
    loadData();
  }, [loadData]);

  // ── Proxy handlers ───────────────────────────────────────────────────────────

  async function handleProxySubmit(data: CreateProxyInput) {
    if (editProxy) {
      await updateProxy(editProxy.id, data, { token });
      showSuccess("Node updated");
    } else {
      await createProxy({ ...data, collection_id: collectionId }, { token });
      showSuccess("Node added");
    }
    setNodeDialogOpen(false);
    setEditProxy(null);
    loadData();
  }

  async function onConfirmDeleteProxy() {
    if (!confirmDeleteProxy) return;
    const { id: pid } = confirmDeleteProxy;
    setConfirmDeleteProxy(null);
    try {
      await deleteProxy(pid, { token });
      setProxies((prev) => prev.filter((p) => p.id !== pid));
      showSuccess("Node deleted");
    } catch (err) {
      showError(err instanceof Error ? err.message : "Failed to delete node");
    }
  }

  // ── Rule handlers ────────────────────────────────────────────────────────────

  async function handleRuleSubmit(data: CreateRuleInput) {
    await createRule({ ...data, collection_id: collectionId }, { token });
    showSuccess("Rule added");
    setRuleDialogOpen(false);
    setEditRule(null);
    loadData();
  }

  async function onConfirmDeleteRule() {
    if (!confirmDeleteRule) return;
    const { id: rid } = confirmDeleteRule;
    setConfirmDeleteRule(null);
    try {
      await deleteRule(rid, { token });
      setRules((prev) => prev.filter((r) => r.id !== rid));
      showSuccess("Rule deleted");
    } catch (err) {
      showError(err instanceof Error ? err.message : "Failed to delete rule");
    }
  }

  if (loading) {
    return (
      <div
        className="flex items-center gap-2 py-8 text-[13px]"
        style={{ color: "var(--color-text-muted)" }}
      >
        <Loader2 className="h-5 w-5 animate-spin" />
        Loading…
      </div>
    );
  }

  if (error) {
    return (
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
    );
  }

  if (!collection) {
    return (
      <p className="text-[13px]" style={{ color: "var(--color-text-muted)" }}>
        Collection not found.
      </p>
    );
  }

  return (
    <div className="flex flex-col gap-6">
      {/* Breadcrumb */}
      <div className="flex items-center gap-2 text-[13px]">
        <Link
          to="/collections"
          className="flex items-center gap-1 font-medium transition-colors hover:underline"
          style={{ color: "var(--color-primary)" }}
        >
          <ArrowLeft className="h-3.5 w-3.5" />
          Collections
        </Link>
      </div>

      {/* Header */}
      <div className="flex flex-col gap-1">
        <div className="flex flex-wrap items-center gap-3">
          <h1
            className="text-[26px] font-bold leading-tight"
            style={{ color: "var(--color-text-primary)" }}
          >
            {collection.name}
          </h1>
          <Badge variant="default">
            {collection.content_type === "proxy" ? "Proxy Collection" : "Rule Collection"}
          </Badge>
        </div>
        {collection.description && (
          <p className="text-[13px]" style={{ color: "var(--color-text-muted)" }}>
            {collection.description}
          </p>
        )}
      </div>

      {/* Proxy collection content */}
      {collection.content_type === "proxy" && (
        <div className="flex flex-col gap-4">
          <div className="flex items-center justify-between">
            <h2
              className="text-[15px] font-semibold"
              style={{ color: "var(--color-text-primary)" }}
            >
              Nodes{" "}
              <span
                className="text-[13px] font-normal"
                style={{ color: "var(--color-text-muted)" }}
              >
                ({proxies.length})
              </span>
            </h2>
            <Button size="sm" onClick={() => { setEditProxy(null); setNodeDialogOpen(true); }}>
              <Plus className="h-4 w-4" />
              Add Node
            </Button>
          </div>

          {proxies.length === 0 ? (
            <div
              className="rounded-[var(--radius-lg)] border px-6 py-12 text-center text-[13px]"
              style={{ color: "var(--color-text-muted)", borderColor: "var(--color-border)" }}
            >
              No nodes yet. Add one to get started.
            </div>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Name</TableHead>
                  <TableHead>Type</TableHead>
                  <TableHead>Server</TableHead>
                  <TableHead>Port</TableHead>
                  <TableHead />
                </TableRow>
              </TableHeader>
              <TableBody>
                {proxies.map((p) => (
                  <TableRow key={p.id}>
                    <TableCell
                      className="font-medium"
                      style={{ color: "var(--color-text-primary)" }}
                    >
                      {p.name}
                    </TableCell>
                    <TableCell>
                      <Badge>{p.type.toUpperCase()}</Badge>
                    </TableCell>
                    <TableCell
                      className="font-mono text-[12px]"
                      style={{ color: "var(--color-text-secondary)" }}
                    >
                      {p.server}
                    </TableCell>
                    <TableCell
                      className="tabular-nums"
                      style={{ color: "var(--color-text-secondary)" }}
                    >
                      {p.port}
                    </TableCell>
                    <TableCell>
                      <div className="flex items-center gap-1 justify-end">
                        <Button
                          variant="ghost"
                          size="sm"
                          onClick={() => { setEditProxy(p); setNodeDialogOpen(true); }}
                        >
                          <Pencil className="h-3.5 w-3.5" />
                        </Button>
                        <Button
                          variant="ghost"
                          size="sm"
                          onClick={() => setConfirmDeleteProxy({ id: p.id, name: p.name })}
                        >
                          <Trash2
                            className="h-3.5 w-3.5"
                            style={{ color: "var(--color-danger)" }}
                          />
                        </Button>
                      </div>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}

          <AddNodeDialog
            open={nodeDialogOpen}
            editProxy={editProxy}
            onSubmit={handleProxySubmit}
            onCancel={() => { setNodeDialogOpen(false); setEditProxy(null); }}
          />

          <ConfirmDialog
            open={!!confirmDeleteProxy}
            title="Delete Node"
            message={`Delete "${confirmDeleteProxy?.name}"?`}
            confirmLabel="Delete"
            onConfirm={onConfirmDeleteProxy}
            onCancel={() => setConfirmDeleteProxy(null)}
          />
        </div>
      )}

      {/* Rule collection content */}
      {collection.content_type === "rule" && (
        <div className="flex flex-col gap-4">
          <div className="flex items-center justify-between">
            <h2
              className="text-[15px] font-semibold"
              style={{ color: "var(--color-text-primary)" }}
            >
              Rules{" "}
              <span
                className="text-[13px] font-normal"
                style={{ color: "var(--color-text-muted)" }}
              >
                ({rules.length})
              </span>
            </h2>
            <Button size="sm" onClick={() => { setEditRule(null); setRuleDialogOpen(true); }}>
              <Plus className="h-4 w-4" />
              Add Rule
            </Button>
          </div>

          {rules.length === 0 ? (
            <div
              className="rounded-[var(--radius-lg)] border px-6 py-12 text-center text-[13px]"
              style={{ color: "var(--color-text-muted)", borderColor: "var(--color-border)" }}
            >
              No rules yet. Add one to get started.
            </div>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Type</TableHead>
                  <TableHead>Payload</TableHead>
                  <TableHead>Target</TableHead>
                  <TableHead />
                </TableRow>
              </TableHeader>
              <TableBody>
                {rules.map((r) => (
                  <TableRow key={r.id}>
                    <TableCell>
                      <Badge>{r.type}</Badge>
                    </TableCell>
                    <TableCell
                      className="font-mono text-[12px]"
                      style={{ color: "var(--color-text-secondary)" }}
                    >
                      {r.payload || "—"}
                    </TableCell>
                    <TableCell
                      className="text-[13px]"
                      style={{ color: "var(--color-text-primary)" }}
                    >
                      {r.target}
                    </TableCell>
                    <TableCell>
                      <div className="flex justify-end">
                        <Button
                          variant="ghost"
                          size="sm"
                          onClick={() =>
                            setConfirmDeleteRule({ id: r.id, name: `${r.type} ${r.payload}` })
                          }
                        >
                          <Trash2
                            className="h-3.5 w-3.5"
                            style={{ color: "var(--color-danger)" }}
                          />
                        </Button>
                      </div>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}

          <AddRuleDialog
            open={ruleDialogOpen}
            editRule={editRule}
            onSubmit={handleRuleSubmit}
            onCancel={() => { setRuleDialogOpen(false); setEditRule(null); }}
          />

          <ConfirmDialog
            open={!!confirmDeleteRule}
            title="Delete Rule"
            message={`Delete rule "${confirmDeleteRule?.name}"?`}
            confirmLabel="Delete"
            onConfirm={onConfirmDeleteRule}
            onCancel={() => setConfirmDeleteRule(null)}
          />
        </div>
      )}
    </div>
  );
}
