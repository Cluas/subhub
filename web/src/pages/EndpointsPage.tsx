import { useEffect, useRef, useState } from "react";
import { Loader2, RefreshCw } from "lucide-react";
import { Pagination } from "@/components/ui/pagination";
import { CopyButton } from "@/components/ui/copy-button";
import {
  fetchEndpoints,
  createEndpoint,
  updateEndpoint,
  deleteEndpoint,
  fetchSubscriptions,
  fetchCollections,
} from "@/lib/api";
import { useAuth } from "@/context/AuthContext";
import type {
  Endpoint,
  CreateEndpointInput,
  UpdateEndpointInput,
  Subscription,
  Collection,
} from "@/types/api";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { ConfirmDialog } from "@/components/ui/confirm-dialog";
import { useToast } from "@/components/ui/toast";

// ── constants ────────────────────────────────────────────────────────────────

const OUTPUT_TYPES = ["proxy", "rule"] as const;
const FORMATS = ["clash", "surge", "quantumult-x"] as const;

const EMPTY_FORM: CreateEndpointInput = {
  name: "",
  subscription_id: null,
  collection_id: null,
  output_type: "proxy",
  format: "clash",
  filters: {},
};

// ── helpers ──────────────────────────────────────────────────────────────────

function providerUrl(slug: string) {
  return `${window.location.origin}/p/${slug}`;
}

function sourceName(
  subs: Subscription[],
  cols: Collection[],
  ep: Endpoint
): string {
  if (ep.collection_id) {
    const c = cols.find((c) => c.id === ep.collection_id);
    return c ? `${c.name}` : `Collection ${ep.collection_id}`;
  }
  if (ep.subscription_id) {
    const s = subs.find((s) => s.id === ep.subscription_id);
    return s ? s.name : String(ep.subscription_id);
  }
  return "All Sources";
}

function toSlug(name: string): string {
  return name
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/^-|-$/g, "") || "preview-url";
}

// ── EndpointForm ─────────────────────────────────────────────────────────────

interface EndpointFormProps {
  subscriptions: Subscription[];
  collections: Collection[];
  initial?: CreateEndpointInput;
  endpointId?: number;
  token?: string;
  onSubmit: (data: CreateEndpointInput) => Promise<Endpoint | void>;
  onCancel: () => void;
  submitLabel: string;
}

function EndpointForm({
  subscriptions,
  collections,
  initial = EMPTY_FORM,
  endpointId,
  token,
  onSubmit,
  onCancel,
  submitLabel,
}: EndpointFormProps) {
  const [form, setForm] = useState<CreateEndpointInput>(initial);
  const [formError, setFormError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);
  const [previewText, setPreviewText] = useState<string | null>(null);
  const [previewLoading, setPreviewLoading] = useState(false);
  const [previewError, setPreviewError] = useState<string | null>(null);
  const savedEndpointId = useRef<number | undefined>(endpointId);

  const [sourceType, setSourceType] = useState<"none" | "subscription" | "collection">(
    initial.collection_id ? "collection" : initial.subscription_id ? "subscription" : "none"
  );

  // Auto-fetch preview when editing an existing endpoint
  useEffect(() => {
    if (endpointId) {
      fetchPreview(endpointId);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  async function fetchPreview(id: number) {
    setPreviewLoading(true);
    setPreviewError(null);
    try {
      // Preview returns YAML/text, not JSON — use raw fetch
      const headers: Record<string, string> = {};
      if (token) headers["Authorization"] = `Bearer ${token}`;
      const res = await fetch(`/api/endpoints/${id}/preview`, { headers });
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      setPreviewText(await res.text());
    } catch (err) {
      setPreviewError(err instanceof Error ? err.message : "Preview unavailable");
      setPreviewText(null);
    } finally {
      setPreviewLoading(false);
    }
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!form.name.trim()) {
      setFormError("Name is required.");
      return;
    }
    if (slug && !/^[a-z0-9][a-z0-9-]*$/.test(slug)) {
      setSlugError("Lowercase alphanumeric + hyphens only");
      return;
    }
    setSubmitting(true);
    setFormError(null);
    try {
      const submitData = { ...form, slug: slug || undefined };
      const result = await onSubmit(submitData);
      // If we got back an Endpoint with an id, fetch preview
      if (result && typeof (result as Endpoint).id === "number") {
        const ep = result as Endpoint;
        savedEndpointId.current = ep.id;
        fetchPreview(ep.id);
      } else if (savedEndpointId.current) {
        fetchPreview(savedEndpointId.current);
      }
    } catch (err: unknown) {
      setFormError(err instanceof Error ? err.message : "Operation failed");
    } finally {
      setSubmitting(false);
    }
  }

  const inputClass =
    "h-10 rounded-lg bg-[var(--color-bg)] border border-[var(--color-border)] px-3 text-[13px] text-[var(--color-text-secondary)] placeholder:text-[var(--color-text-muted)] outline-none focus:border-[var(--color-primary)] transition-colors w-full";

  const [slug, setSlug] = useState(initial.slug || toSlug(initial.name));
  const [slugError, setSlugError] = useState<string | null>(null);

  return (
    <div className="grid grid-cols-1 xl:grid-cols-[1fr_320px] gap-6">
      {/* Left: main form card */}
      <form
        onSubmit={handleSubmit}
        className="bg-[var(--color-bg-card)] rounded-xl border border-[var(--color-border)]/50 p-6 flex flex-col gap-5 shadow-[var(--shadow-card)]"
      >
        <h2 className="text-[20px] font-semibold text-[var(--color-text-primary)]">
          {submitLabel}
        </h2>

        {formError && (
          <p className="text-[13px] text-[var(--color-danger)]">{formError}</p>
        )}

        {/* Name */}
        <div className="flex flex-col gap-1.5">
          <label className="text-[12px] font-medium text-[var(--color-text-muted)] uppercase tracking-wide">
            Name
          </label>
          <input
            type="text"
            required
            value={form.name}
            onChange={(e) => setForm((f) => ({ ...f, name: e.target.value }))}
            placeholder="e.g. Main Clash Config"
            className={inputClass}
          />
        </div>

        {/* Slug */}
        <div className="flex flex-col gap-1.5">
          <label className="text-[12px] font-medium text-[var(--color-text-muted)] uppercase tracking-wide">
            Slug
          </label>
          <input
            type="text"
            value={slug}
            onChange={(e) => { setSlug(e.target.value.toLowerCase().replace(/[^a-z0-9-]/g, "")); setSlugError(null); }}
            placeholder="e.g. lb-foreign"
            className={inputClass}
            style={slugError ? { borderColor: "var(--color-danger)" } : undefined}
          />
          {slugError && <p className="text-[12px] text-[var(--color-danger)]">{slugError}</p>}
          <p className="text-[11px] font-mono text-[var(--color-text-muted)]">{window.location.origin}/p/{slug || "..."}</p>
        </div>

        {/* Output Type + Format row */}
        <div className="grid grid-cols-2 gap-4">
          <div className="flex flex-col gap-1.5">
            <label className="text-[12px] font-medium text-[var(--color-text-muted)] uppercase tracking-wide">
              Output Type
            </label>
            <select
              value={form.output_type}
              onChange={(e) => setForm((f) => ({ ...f, output_type: e.target.value }))}
              className={inputClass}
            >
              {OUTPUT_TYPES.map((t) => (
                <option key={t} value={t}>{t === "proxy" ? "Proxy" : "Rule"}</option>
              ))}
            </select>
          </div>
          <div className="flex flex-col gap-1.5">
            <label className="text-[12px] font-medium text-[var(--color-text-muted)] uppercase tracking-wide">
              Format
            </label>
            <select
              value={form.format}
              onChange={(e) => setForm((f) => ({ ...f, format: e.target.value }))}
              className={inputClass}
            >
              {FORMATS.map((fmt) => (
                <option key={fmt} value={fmt}>{fmt}</option>
              ))}
            </select>
          </div>
        </div>

        {/* Source select with optgroups */}
        <div className="flex flex-col gap-1.5">
          <label className="text-[12px] font-medium text-[var(--color-text-muted)] uppercase tracking-wide">
            Source
          </label>
          <select
            value={
              sourceType === "none"
                ? ""
                : sourceType === "subscription"
                ? `sub:${form.subscription_id ?? ""}`
                : `col:${form.collection_id ?? ""}`
            }
            onChange={(e) => {
              const val = e.target.value;
              if (!val) {
                setSourceType("none");
                setForm((f) => ({ ...f, subscription_id: null, collection_id: null }));
              } else if (val.startsWith("sub:")) {
                const id = Number(val.slice(4)) || null;
                setSourceType("subscription");
                setForm((f) => ({ ...f, subscription_id: id, collection_id: null }));
              } else if (val.startsWith("col:")) {
                const id = Number(val.slice(4)) || null;
                setSourceType("collection");
                setForm((f) => ({ ...f, subscription_id: null, collection_id: id }));
              }
            }}
            className={inputClass}
          >
            <option value="">(All Sources)</option>
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
                  <option key={`col:${c.id}`} value={`col:${c.id}`}>{c.name} ({c.content_type})</option>
                ))}
              </optgroup>
            )}
          </select>
        </div>

        {/* Filters section */}
        <div className="flex flex-col gap-3">
          <label className="text-[12px] font-medium text-[var(--color-text-muted)] uppercase tracking-wide">
            Filters
          </label>
          <div className="grid grid-cols-2 gap-4">
            <div className="flex flex-col gap-1.5">
              <label className="text-[11px] text-[var(--color-text-muted)]">Type</label>
              <input
                type="text"
                value={form.filters?.types?.join(", ") ?? ""}
                onChange={(e) => {
                  const val = e.target.value;
                  setForm((f) => ({
                    ...f,
                    filters: {
                      ...f.filters,
                      types: val ? val.split(",").map((v) => v.trim()).filter(Boolean) : undefined,
                    },
                  }));
                }}
                placeholder="ss, vmess, trojan"
                className={inputClass}
              />
            </div>
            <div className="flex flex-col gap-1.5">
              <label className="text-[11px] text-[var(--color-text-muted)]">Region</label>
              <input
                type="text"
                value={form.filters?.regions?.join(", ") ?? ""}
                onChange={(e) => {
                  const val = e.target.value;
                  setForm((f) => ({
                    ...f,
                    filters: {
                      ...f.filters,
                      regions: val ? val.split(",").map((v) => v.trim()).filter(Boolean) : undefined,
                    },
                  }));
                }}
                placeholder="HK, JP, SG"
                className={inputClass}
              />
            </div>
          </div>
          <div className="flex flex-col gap-1.5">
            <label className="text-[11px] text-[var(--color-text-muted)]">Groups</label>
            <input
              type="text"
              value={form.filters?.groups?.join(", ") ?? ""}
              onChange={(e) => {
                const val = e.target.value;
                setForm((f) => ({
                  ...f,
                  filters: {
                    ...f.filters,
                    groups: val ? val.split(",").map((v) => v.trim()).filter(Boolean) : undefined,
                  },
                }));
              }}
              placeholder="访问国外线路组, 内网流量"
              className={inputClass}
            />
            <p className="text-[11px] text-[var(--color-text-muted)]">Filter by proxy-group name from subscription</p>
          </div>
          <div className="grid grid-cols-2 gap-4">
            <div className="flex flex-col gap-1.5">
              <label className="text-[11px] text-[var(--color-text-muted)]">Name Contains</label>
              <input
                type="text"
                value={form.filters?.name_contains ?? ""}
                onChange={(e) =>
                  setForm((f) => ({
                    ...f,
                    filters: { ...f.filters, name_contains: e.target.value || undefined },
                  }))
                }
                placeholder="Filter by name..."
                className={inputClass}
              />
            </div>
            <div className="flex flex-col gap-1.5">
              <label className="text-[11px] text-[var(--color-text-muted)]">Target</label>
              <input
                type="text"
                value={form.filters?.target ?? ""}
                onChange={(e) =>
                  setForm((f) => ({
                    ...f,
                    filters: { ...f.filters, target: e.target.value || undefined },
                  }))
                }
                placeholder="PROXY, DIRECT..."
                className={inputClass}
              />
            </div>
          </div>
        </div>

        {/* Action row */}
        <div className="flex gap-3 items-center pt-1">
          <Button type="submit" disabled={submitting}>
            {submitting ? (
              <>
                <Loader2 className="h-4 w-4 animate-spin" />
                Saving…
              </>
            ) : (
              submitLabel
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

      {/* Right: Live Preview panel */}
      <div className="bg-[var(--color-bg-card)] rounded-xl border border-[var(--color-border)]/50 p-5 flex flex-col gap-3 shadow-[var(--shadow-card)]">
        <div className="flex items-center justify-between">
          <h3 className="text-[13px] font-semibold text-[var(--color-text-primary)]">
            Live Preview
          </h3>
          {savedEndpointId.current && (
            <button
              type="button"
              onClick={() => savedEndpointId.current && fetchPreview(savedEndpointId.current)}
              disabled={previewLoading}
              className="flex items-center gap-1.5 text-[12px] text-[var(--color-text-muted)] hover:text-[var(--color-text-secondary)] transition-colors disabled:opacity-50"
            >
              <RefreshCw className={`h-3 w-3 ${previewLoading ? "animate-spin" : ""}`} />
              Refresh
            </button>
          )}
        </div>

        <div
          className="rounded-lg p-3 overflow-auto"
          style={{
            backgroundColor: "var(--color-bg)",
            maxHeight: "400px",
          }}
        >
          {previewLoading ? (
            <div className="flex items-center justify-center py-8">
              <Loader2 className="h-4 w-4 animate-spin text-[var(--color-text-muted)]" />
            </div>
          ) : previewError ? (
            <p className="text-[11px] text-[var(--color-danger)] font-mono">{previewError}</p>
          ) : previewText ? (
            <pre className="text-[11px] font-mono text-[var(--color-text-muted)] whitespace-pre-wrap break-all">
              {previewText}
            </pre>
          ) : (
            <p className="text-[11px] font-mono text-[var(--color-text-muted)] italic py-4 text-center">
              Save endpoint to see live preview
            </p>
          )}
        </div>
      </div>
    </div>
  );
}

// ── EndpointsPage ─────────────────────────────────────────────────────────────

export function EndpointsPage() {
  const { token } = useAuth();
  const { showSuccess, showError } = useToast();
  const [endpoints, setEndpoints] = useState<Endpoint[]>([]);
  const [subscriptions, setSubscriptions] = useState<Subscription[]>([]);
  const [collections, setCollections] = useState<Collection[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [creating, setCreating] = useState(false);
  const [editingId, setEditingId] = useState<number | null>(null);
  const [deleting, setDeleting] = useState<Record<number, boolean>>({});
  const [confirmDelete, setConfirmDelete] = useState<{ id: number; name: string } | null>(null);
  const [page, setPage] = useState(1);
  const PAGE_SIZE = 20;

  const totalPages = Math.max(1, Math.ceil(endpoints.length / PAGE_SIZE));
  const paged = endpoints.slice((page - 1) * PAGE_SIZE, page * PAGE_SIZE);

  function load() {
    let cancelled = false;
    setLoading(true);
    setError(null);
    Promise.all([fetchEndpoints({ token }), fetchSubscriptions({ token }), fetchCollections({ token })])
      .then(([eps, subs, cols]) => {
        if (!cancelled) {
          setEndpoints(eps);
          setSubscriptions(subs);
          setCollections(cols);
          setLoading(false);
        }
      })
      .catch((err: unknown) => {
        if (!cancelled) {
          setError(err instanceof Error ? err.message : "Failed to load endpoints");
          setLoading(false);
        }
      });
    return () => {
      cancelled = true;
    };
  }

  useEffect(() => {
    load();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  async function handleCreate(data: CreateEndpointInput): Promise<Endpoint> {
    try {
      const ep = await createEndpoint(data, { token });
      setCreating(false);
      load();
      showSuccess("Endpoint created successfully");
      return ep;
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : "Failed to create endpoint";
      setError(msg);
      showError(msg);
      throw err;
    }
  }

  async function handleUpdate(id: number, data: UpdateEndpointInput): Promise<Endpoint> {
    try {
      const updated = await updateEndpoint(id, data, { token });
      setEndpoints((prev) => prev.map((ep) => (ep.id === id ? updated : ep)));
      setEditingId(null);
      showSuccess("Endpoint updated successfully");
      return updated;
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : "Failed to update endpoint";
      setError(msg);
      showError(msg);
      throw err;
    }
  }

  async function handleDelete(id: number, name: string) {
    setConfirmDelete({ id, name });
  }

  async function onConfirmDelete() {
    if (!confirmDelete) return;
    const { id } = confirmDelete;
    setConfirmDelete(null);
    setDeleting((prev) => ({ ...prev, [id]: true }));
    try {
      await deleteEndpoint(id, { token });
      setEndpoints((prev) => prev.filter((ep) => ep.id !== id));
      showSuccess("Endpoint deleted successfully");
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : `Failed to delete endpoint ${id}`;
      setError(msg);
      showError(msg);
    } finally {
      setDeleting((prev) => {
        const next = { ...prev };
        delete next[id];
        return next;
      });
    }
  }

  const editingEndpoint = editingId !== null ? endpoints.find((ep) => ep.id === editingId) : null;

  return (
    <div className="flex flex-col gap-6">
      {/* Page header */}
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-[26px] font-bold text-[var(--color-text-primary)]">Endpoints</h1>
          <p className="text-[13px] text-[var(--color-text-secondary)]">Subscription configuration URLs</p>
        </div>
        <Button
          className="hidden md:flex"
          onClick={() => {
            setCreating((v) => !v);
            setEditingId(null);
          }}
        >
          {creating ? "Cancel" : "New Endpoint"}
        </Button>
      </div>

      {/* Error banner */}
      {error && (
        <div className="rounded-[var(--radius)] border border-[var(--color-danger)]/30 bg-[var(--color-danger-bg)] px-4 py-3 text-sm text-[var(--color-danger)]">
          <strong>Error:</strong> {error}
        </div>
      )}

      {/* Create form */}
      {creating && (
        <EndpointForm
          subscriptions={subscriptions}
          collections={collections}
          token={token ?? undefined}
          onSubmit={handleCreate}
          onCancel={() => setCreating(false)}
          submitLabel="Create Endpoint"
        />
      )}

      {/* Edit form */}
      {editingEndpoint && (
        <EndpointForm
          subscriptions={subscriptions}
          collections={collections}
          endpointId={editingEndpoint.id}
          token={token ?? undefined}
          initial={{
            name: editingEndpoint.name,
            slug: editingEndpoint.slug,
            subscription_id: editingEndpoint.subscription_id,
            collection_id: editingEndpoint.collection_id,
            output_type: editingEndpoint.output_type,
            format: editingEndpoint.format,
            filters: editingEndpoint.filters,
          }}
          onSubmit={(data) => handleUpdate(editingEndpoint.id, data)}
          onCancel={() => setEditingId(null)}
          submitLabel="Save Changes"
        />
      )}

      {/* Loading skeleton */}
      {loading ? (
        <div className="flex flex-col gap-4">
          {[...Array(3)].map((_, i) => (
            <div
              key={i}
              className="h-36 rounded-[var(--radius-lg)] animate-pulse bg-[var(--color-bg-card)]"
            />
          ))}
        </div>
      ) : endpoints.length === 0 ? (
        <div className="rounded-[var(--radius-lg)] border border-[var(--color-border)]/50 px-6 py-12 text-center text-sm text-[var(--color-text-muted)]">
          No endpoints yet. Create one to get a shareable provider URL.
        </div>
      ) : (
        <div className="flex flex-col gap-4">
          {paged.map((ep) => (
            <div
              key={ep.id}
              className="bg-[var(--color-bg-card)] rounded-[var(--radius-lg)] border border-[var(--color-border)]/50 shadow-[var(--shadow-card)] p-5 flex flex-col"
            >
              {/* Name + badges */}
              <div className="flex items-start justify-between gap-2">
                <span className="text-[15px] font-semibold text-[var(--color-text-primary)] leading-tight">
                  {ep.name}
                </span>
                <div className="flex gap-1.5 flex-shrink-0">
                  <Badge variant="type">{ep.format}</Badge>
                  <Badge variant="default">{ep.output_type}</Badge>
                </div>
              </div>

              {/* Source info */}
              <p className="text-[12px] text-[var(--color-text-muted)] mt-1">
                Source: {sourceName(subscriptions, collections, ep)}
              </p>

              {/* URL bar */}
              <div className="flex items-center gap-2 mt-3 bg-[var(--color-bg)] rounded-[var(--radius)] border border-[var(--color-border)] px-3 py-2">
                <span className="flex-1 text-[12px] font-mono text-[var(--color-text-muted)] truncate">
                  {providerUrl(ep.slug)}
                </span>
                <CopyButton text={providerUrl(ep.slug)} />
              </div>

              {/* Action row */}
              <div className="mt-3 flex gap-2 justify-end">
                <Button
                  variant="secondary"
                  size="sm"
                  onClick={() => setEditingId(ep.id === editingId ? null : ep.id)}
                >
                  Edit
                </Button>
                <Button
                  variant="destructive"
                  size="sm"
                  disabled={!!deleting[ep.id]}
                  onClick={() => handleDelete(ep.id, ep.name)}
                >
                  {deleting[ep.id] ? <><Loader2 className="h-3 w-3 animate-spin" />Deleting…</> : "Delete"}
                </Button>
              </div>
            </div>
          ))}
        </div>
      )}

      {/* Pagination */}
      {!loading && (
        <Pagination
          currentPage={page}
          totalPages={totalPages}
          totalItems={endpoints.length}
          onPageChange={setPage}
        />
      )}

      {/* Mobile FAB */}
      <button
        className="fixed bottom-6 right-6 z-40 md:hidden w-12 h-12 rounded-full bg-[var(--color-primary)] text-[var(--color-bg)] flex items-center justify-center shadow-lg text-xl font-bold"
        onClick={() => {
          setCreating((v) => !v);
          setEditingId(null);
        }}
        aria-label="New endpoint"
      >
        +
      </button>

      <ConfirmDialog
        open={!!confirmDelete}
        title="Delete Endpoint"
        message={
          confirmDelete ? (
            <>
              Delete endpoint <strong>{confirmDelete.name}</strong>? This cannot be undone.
            </>
          ) : null
        }
        onConfirm={onConfirmDelete}
        onCancel={() => setConfirmDelete(null)}
      />
    </div>
  );
}
