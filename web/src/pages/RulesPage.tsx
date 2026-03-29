import { useEffect, useMemo, useState } from "react";
import { Loader2, Pencil, Trash2 } from "lucide-react";
import { Pagination } from "@/components/ui/pagination";
import {
  fetchRules,
  fetchSubscriptions,
  fetchCollections,
  createRule,
  updateRule,
  deleteRule,
} from "@/lib/api";
import { useAuth } from "@/context/AuthContext";
import { useToast } from "@/components/ui/toast";
import type { Rule, CreateRuleInput, Subscription, Collection } from "@/types/api";
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
import { ConfirmDialog } from "@/components/ui/confirm-dialog";

// ── constants ─────────────────────────────────────────────────────────────────

const RULE_TYPES_FILTER = ["All", "DOMAIN-SUFFIX", "DOMAIN-KEYWORD", "IP-CIDR", "IN-NAME", "GEOIP"];

const RULE_TYPE_OPTIONS = [
  "DOMAIN-SUFFIX",
  "DOMAIN-KEYWORD",
  "IP-CIDR",
  "IN-NAME",
  "GEOIP",
];

const RULE_TYPE_REFERENCE = [
  { type: "DOMAIN-SUFFIX", desc: "Domain suffix match" },
  { type: "DOMAIN-KEYWORD", desc: "Domain keyword match" },
  { type: "IP-CIDR", desc: "IP range match (CIDR)" },
  { type: "IN-NAME", desc: "Proxy name match" },
  { type: "GEOIP", desc: "Geographic IP match" },
];

const EMPTY_RULE: CreateRuleInput = {
  type: "DOMAIN-SUFFIX",
  payload: "",
  target: "PROXY",
  provider_name: "",
  subscription_id: null,
  collection_id: null,
};

const inputClass =
  "h-10 w-full rounded-lg bg-[var(--color-bg)] border border-[var(--color-border)] px-3 text-[13px] text-[var(--color-text-secondary)] placeholder:text-[var(--color-text-muted)] outline-none focus:border-[var(--color-primary)] transition-colors";

// ── Rule Type Reference Panel ─────────────────────────────────────────────────

function RuleTypeReferencePanel() {
  return (
    <div className="bg-[var(--color-bg-card)] rounded-xl border border-[var(--color-border)]/50 p-5 flex flex-col gap-3 shadow-[var(--shadow-card)]">
      <h3 className="text-[13px] font-semibold text-[var(--color-text-primary)] mb-1">
        Rule Type Reference
      </h3>
      <div className="flex flex-col gap-2.5">
        {RULE_TYPE_REFERENCE.map((item) => (
          <div key={item.type} className="flex items-start gap-2.5">
            <span
              className="flex-shrink-0 rounded px-2 py-0.5 text-[11px] font-mono leading-snug"
              style={{
                backgroundColor: "var(--color-primary-bg)",
                color: "var(--color-primary-light)",
              }}
            >
              {item.type}
            </span>
            <span className="text-[11px] text-[var(--color-text-muted)] leading-snug pt-0.5">
              {item.desc}
            </span>
          </div>
        ))}
      </div>
    </div>
  );
}

// ── Inline Rule Form ──────────────────────────────────────────────────────────

interface RuleFormProps {
  editTarget: Rule | null;
  subscriptions: Subscription[];
  collections: Collection[];
  onSubmit: (data: CreateRuleInput) => Promise<void>;
  onCancel: () => void;
}

function RuleForm({ editTarget, subscriptions, collections, onSubmit, onCancel }: RuleFormProps) {
  const [form, setForm] = useState<CreateRuleInput>(
    editTarget
      ? {
          type: editTarget.type,
          payload: editTarget.payload,
          target: editTarget.target,
          provider_name: editTarget.provider_name ?? "",
          subscription_id: editTarget.subscription_id,
          collection_id: editTarget.collection_id,
        }
      : EMPTY_RULE
  );
  const [formError, setFormError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!form.payload.trim()) {
      setFormError("Payload is required.");
      return;
    }
    setSubmitting(true);
    setFormError(null);
    try {
      await onSubmit(form);
    } catch (err) {
      setFormError(err instanceof Error ? err.message : "Operation failed");
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <div className="grid grid-cols-1 lg:grid-cols-[1fr_280px] gap-6">
      {/* Left: main form */}
      <form
        onSubmit={handleSubmit}
        className="bg-[var(--color-bg-card)] rounded-xl border border-[var(--color-border)]/50 p-6 flex flex-col gap-5 shadow-[var(--shadow-card)]"
      >
        <h2 className="text-[20px] font-semibold text-[var(--color-text-primary)]">
          {editTarget ? "Edit Rule" : "Add Rule"}
        </h2>

        {formError && (
          <p className="text-[13px] text-[var(--color-danger)]">{formError}</p>
        )}

        {/* Rule Type + Target row */}
        <div className="grid grid-cols-2 gap-4">
          <div className="flex flex-col gap-1.5">
            <label className="text-[12px] font-medium text-[var(--color-text-muted)] uppercase tracking-wide">
              Rule Type
            </label>
            <select
              value={form.type}
              onChange={(e) => setForm((f) => ({ ...f, type: e.target.value }))}
              className={inputClass}
            >
              {RULE_TYPE_OPTIONS.map((t) => (
                <option key={t} value={t}>{t}</option>
              ))}
            </select>
          </div>
          <div className="flex flex-col gap-1.5">
            <label className="text-[12px] font-medium text-[var(--color-text-muted)] uppercase tracking-wide">
              Target
            </label>
            <input
              type="text"
              value={form.target}
              onChange={(e) => setForm((f) => ({ ...f, target: e.target.value }))}
              placeholder="PROXY, DIRECT, REJECT..."
              className={inputClass}
            />
          </div>
        </div>

        {/* Payload */}
        <div className="flex flex-col gap-1.5">
          <label className="text-[12px] font-medium text-[var(--color-text-muted)] uppercase tracking-wide">
            Payload
          </label>
          <input
            type="text"
            value={form.payload}
            onChange={(e) => setForm((f) => ({ ...f, payload: e.target.value }))}
            placeholder={
              form.type === "GEOIP"
                ? "e.g. CN"
                : form.type === "IP-CIDR"
                ? "e.g. 8.8.8.0/24"
                : "e.g. google.com"
            }
            className={inputClass}
          />
        </div>

        {/* Provider / Source select with optgroups */}
        <div className="flex flex-col gap-1.5">
          <label className="text-[12px] font-medium text-[var(--color-text-muted)] uppercase tracking-wide">
            Provider / Source (optional)
          </label>
          <select
            value={
              form.subscription_id ? `sub:${form.subscription_id}` :
              form.collection_id ? `col:${form.collection_id}` : ""
            }
            onChange={(e) => {
              const val = e.target.value;
              if (!val) {
                setForm((f) => ({ ...f, subscription_id: null, collection_id: null }));
              } else if (val.startsWith("sub:")) {
                const id = Number(val.slice(4)) || null;
                setForm((f) => ({ ...f, subscription_id: id, collection_id: null }));
              } else if (val.startsWith("col:")) {
                const id = Number(val.slice(4)) || null;
                setForm((f) => ({ ...f, subscription_id: null, collection_id: id }));
              }
            }}
            className={inputClass}
          >
            <option value="">(None)</option>
            {subscriptions.length > 0 && (
              <optgroup label="Subscriptions">
                {subscriptions.map((s) => (
                  <option key={`sub:${s.id}`} value={`sub:${s.id}`}>{s.name}</option>
                ))}
              </optgroup>
            )}
            {collections.length > 0 && (
              <optgroup label="Collections">
                {collections.map((c) => (
                  <option key={`col:${c.id}`} value={`col:${c.id}`}>{c.name}</option>
                ))}
              </optgroup>
            )}
          </select>
        </div>

        {/* Provider Name */}
        <div className="flex flex-col gap-1.5">
          <label className="text-[12px] font-medium text-[var(--color-text-muted)] uppercase tracking-wide">
            Provider Name (optional)
          </label>
          <input
            type="text"
            value={form.provider_name ?? ""}
            onChange={(e) => setForm((f) => ({ ...f, provider_name: e.target.value }))}
            placeholder="e.g. provider1"
            className={inputClass}
          />
        </div>

        {/* Action row */}
        <div className="flex gap-3 items-center pt-1">
          <Button type="submit" disabled={submitting}>
            {submitting ? (
              <>
                <Loader2 className="h-4 w-4 animate-spin" />
                {editTarget ? "Saving…" : "Adding…"}
              </>
            ) : editTarget ? (
              "Save"
            ) : (
              "Add Rule"
            )}
          </Button>
          <button
            type="button"
            onClick={onCancel}
            className="text-[13px] text-[var(--color-text-muted)] hover:text-[var(--color-text-secondary)] transition-colors"
          >
            Cancel
          </button>
        </div>
      </form>

      {/* Right: Rule Type Reference panel */}
      <RuleTypeReferencePanel />
    </div>
  );
}

// ── RulesPage ─────────────────────────────────────────────────────────────────

export function RulesPage() {
  const { token } = useAuth();
  const { showSuccess, showError } = useToast();
  const [rules, setRules] = useState<Rule[]>([]);
  const [subscriptions, setSubscriptions] = useState<Subscription[]>([]);
  const [collections, setCollections] = useState<Collection[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Filter state
  const [activeType, setActiveType] = useState("All");
  const [keywordInput, setKeywordInput] = useState("");
  const [filterKeyword, setFilterKeyword] = useState("");

  // Pagination state
  const [page, setPage] = useState(1);
  const PAGE_SIZE = 50;

  // Inline form state
  const [showForm, setShowForm] = useState(false);
  const [editTarget, setEditTarget] = useState<Rule | null>(null);

  // Confirm-delete state
  const [confirmDelete, setConfirmDelete] = useState<{ id: number; label: string } | null>(null);
  const [rowLoading, setRowLoading] = useState<Record<number, string>>({});

  // ── Data loading ────────────────────────────────────────────────────────────

  function loadAll() {
    let cancelled = false;
    setLoading(true);
    setError(null);
    Promise.all([
      fetchRules(undefined, { token }),
      fetchSubscriptions({ token }),
      fetchCollections({ token }),
    ])
      .then(([ruleData, subData, colData]) => {
        if (!cancelled) {
          setRules(ruleData);
          setSubscriptions(subData);
          setCollections(colData);
          setLoading(false);
        }
      })
      .catch((err: unknown) => {
        if (!cancelled) {
          setError(err instanceof Error ? err.message : "Failed to load rules");
          setLoading(false);
        }
      });
    return () => {
      cancelled = true;
    };
  }

  useEffect(() => {
    return loadAll();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // ── Derived filter & pagination ─────────────────────────────────────────────

  const filtered = useMemo(() => {
    return rules.filter((r) => {
      if (activeType !== "All" && r.type !== activeType) return false;
      if (filterKeyword) {
        const kw = filterKeyword.toLowerCase();
        return (
          r.payload.toLowerCase().includes(kw) ||
          r.type.toLowerCase().includes(kw) ||
          r.target.toLowerCase().includes(kw) ||
          r.provider_name.toLowerCase().includes(kw)
        );
      }
      return true;
    });
  }, [rules, activeType, filterKeyword]);

  const totalPages = Math.max(1, Math.ceil(filtered.length / PAGE_SIZE));
  const paginated = filtered.slice((page - 1) * PAGE_SIZE, page * PAGE_SIZE);

  // Reset to page 1 whenever filters change
  useEffect(() => setPage(1), [activeType, filterKeyword]);

  // ── Actions ─────────────────────────────────────────────────────────────────

  function handleKeywordKeyDown(e: React.KeyboardEvent<HTMLInputElement>) {
    if (e.key === "Enter") setFilterKeyword(keywordInput);
    if (e.key === "Escape") {
      setKeywordInput("");
      setFilterKeyword("");
    }
  }

  function openAdd() {
    setEditTarget(null);
    setShowForm(true);
  }

  function openEdit(rule: Rule) {
    setEditTarget(rule);
    setShowForm(true);
  }

  function closeForm() {
    setShowForm(false);
    setEditTarget(null);
  }

  async function handleFormSubmit(data: CreateRuleInput) {
    if (editTarget) {
      await updateRule(editTarget.id, data, { token });
      showSuccess("Rule updated successfully");
    } else {
      await createRule(data, { token });
      showSuccess("Rule added successfully");
    }
    closeForm();
    loadAll();
  }

  function requestDelete(rule: Rule) {
    setConfirmDelete({ id: rule.id, label: `${rule.type} — ${rule.payload}` });
  }

  async function onConfirmDelete() {
    if (!confirmDelete) return;
    const { id } = confirmDelete;
    setConfirmDelete(null);
    setRowLoading((prev) => ({ ...prev, [id]: "delete" }));
    try {
      await deleteRule(id, { token });
      setRules((prev) => prev.filter((r) => r.id !== id));
      showSuccess("Rule deleted successfully");
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : "Failed to delete rule";
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

  // ── Render ───────────────────────────────────────────────────────────────────

  return (
    <div className="flex flex-col gap-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-[26px] font-bold text-[var(--color-text-primary)]">Rules</h1>
          <p className="text-[13px] text-[var(--color-text-secondary)]">Routing rules for managing routing logic</p>
        </div>
        <Button onClick={showForm ? closeForm : openAdd}>
          {showForm ? "Cancel" : "+ Add Rule"}
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

      {/* Inline form */}
      {showForm && (
        <RuleForm
          editTarget={editTarget}
          subscriptions={subscriptions}
          collections={collections}
          onSubmit={handleFormSubmit}
          onCancel={closeForm}
        />
      )}

      {/* Filter bar — search + type chips */}
      <div className="flex flex-wrap items-center gap-3">
        {/* Search */}
        <input
          type="text"
          placeholder="Search rules…"
          value={keywordInput}
          onChange={(e) => setKeywordInput(e.target.value)}
          onKeyDown={handleKeywordKeyDown}
          onBlur={() => setFilterKeyword(keywordInput)}
          className="h-9 flex-1 min-w-[200px] rounded-[var(--radius)] bg-[var(--color-bg-card)] border border-[var(--color-border)] px-3 text-[13px] text-[var(--color-text-secondary)] outline-none focus:border-[var(--color-primary)] transition-colors"
          aria-label="Keyword search"
        />

        {/* Type filter chips */}
        <div className="flex flex-wrap gap-1.5">
          {RULE_TYPES_FILTER.map((type) => {
            const isActive = activeType === type;
            return (
              <button
                key={type}
                onClick={() => setActiveType(type)}
                className={`rounded-md border px-3 py-1 text-[12px] font-medium transition-colors ${
                  isActive
                    ? "bg-[var(--color-primary-bg)] text-[var(--color-primary-light)] border-[var(--color-primary)]/30"
                    : "bg-[var(--color-bg-card)] text-[var(--color-text-secondary)] border-[var(--color-border)] hover:bg-[var(--color-bg-accent)]"
                }`}
              >
                {type}
              </button>
            );
          })}
        </div>
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
      ) : rules.length === 0 ? (
        <div
          className="rounded-xl border px-6 py-12 text-center text-[13px]"
          style={{
            color: "var(--color-text-muted)",
            borderColor: "var(--color-border)",
            backgroundColor: "var(--color-bg-card)",
          }}
        >
          No rules found.{" "}
          <button
            onClick={openAdd}
            className="underline hover:no-underline"
            style={{ color: "var(--color-primary)" }}
          >
            Add a self-managed rule
          </button>{" "}
          or fetch a subscription to populate rules.
        </div>
      ) : filtered.length === 0 ? (
        <div
          className="rounded-xl border px-6 py-8 text-center text-[13px]"
          style={{
            color: "var(--color-text-muted)",
            borderColor: "var(--color-border)",
            backgroundColor: "var(--color-bg-card)",
          }}
        >
          No rules match the current filters.
        </div>
      ) : (
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>TYPE</TableHead>
              <TableHead>KEYWORD</TableHead>
              <TableHead>TARGET</TableHead>
              <TableHead>ACTION</TableHead>
              <TableHead>SOURCE</TableHead>
              <TableHead>ACTIONS</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {paginated.map((rule) => {
              const busy = rowLoading[rule.id];
              const isSelfManaged = rule.subscription_id === null;
              return (
                <TableRow key={rule.id}>
                  <TableCell>
                    <Badge variant="type">{rule.type}</Badge>
                  </TableCell>
                  <TableCell
                    className="font-mono text-xs max-w-[240px] truncate"
                    title={rule.payload}
                  >
                    {rule.payload}
                  </TableCell>
                  <TableCell>
                    <span className="text-[var(--color-primary-light)] font-medium">{rule.target}</span>
                  </TableCell>
                  <TableCell>
                    <Badge variant="default">{rule.provider_name || "—"}</Badge>
                  </TableCell>
                  <TableCell>
                    {isSelfManaged ? (
                      <Badge variant="default">Self-managed</Badge>
                    ) : (
                      <Badge variant="default">#{rule.subscription_id}</Badge>
                    )}
                  </TableCell>
                  <TableCell>
                    <div className="flex gap-1.5">
                      <button
                        title="Edit rule"
                        disabled={!!busy}
                        onClick={() => openEdit(rule)}
                        className="rounded-md p-1.5 hover:bg-[var(--color-bg-accent)] transition-colors disabled:opacity-40"
                        aria-label={`Edit ${rule.type} ${rule.payload}`}
                      >
                        <Pencil className="h-3.5 w-3.5 text-[var(--color-text-muted)]" />
                      </button>
                      <button
                        title="Delete rule"
                        disabled={!!busy}
                        onClick={() => requestDelete(rule)}
                        className="rounded-md p-1.5 hover:bg-[var(--color-danger-bg)] transition-colors disabled:opacity-40"
                        aria-label={`Delete ${rule.type} ${rule.payload}`}
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

      {/* Delete confirmation dialog */}
      <ConfirmDialog
        open={!!confirmDelete}
        title="Delete Rule"
        message={
          confirmDelete ? (
            <>
              Delete rule <strong>{confirmDelete.label}</strong>? This cannot be undone.
            </>
          ) : null
        }
        onConfirm={onConfirmDelete}
        onCancel={() => setConfirmDelete(null)}
      />
    </div>
  );
}
