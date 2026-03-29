import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { Loader2 } from "lucide-react";
import { Pagination } from "@/components/ui/pagination";
import {
  fetchSubscriptions,
  createSubscription,
  updateSubscription,
  deleteSubscription,
  triggerFetch,
} from "@/lib/api";
import { useAuth } from "@/context/AuthContext";
import type { Subscription, CreateSubscriptionInput } from "@/types/api";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { ConfirmDialog } from "@/components/ui/confirm-dialog";
import { useToast } from "@/components/ui/toast";

const EMPTY_FORM: CreateSubscriptionInput = {
  name: "",
  url: "",
  type: "clash",
  auto_refresh: false,
  refresh_cron: "",
};

function statusVariant(status: string): "success" | "warning" | "danger" {
  if (status === "active") return "success";
  if (status === "error") return "danger";
  return "warning";
}

function formatRefresh(sub: Subscription): string {
  if (!sub.auto_refresh) return "Manual only";
  if (sub.refresh_cron) return sub.refresh_cron;
  return "Auto refresh";
}

const FILTER_TYPES = ["All", "Clash", "V2Ray", "SIP002"] as const;

// ── Input class ───────────────────────────────────────────────────────────────

const inputClass =
  "h-10 w-full rounded-lg bg-[var(--color-bg)] border border-[var(--color-border)] px-3 text-[13px] text-[var(--color-text-secondary)] outline-none focus:border-[var(--color-primary)] placeholder:text-[var(--color-text-muted)] transition-colors";

// ── Supported Formats side panel ──────────────────────────────────────────────

function SupportedFormatsPanel() {
  const formats = [
    { name: "Clash", subtitle: "YAML-based proxy configuration" },
    { name: "V2Ray", subtitle: "Base64-encoded vmess:// links" },
    { name: "SIP002", subtitle: "Shadowsocks ss:// format" },
  ];
  return (
    <div className="bg-[var(--color-bg-card)] rounded-xl border border-[var(--color-border)]/50 p-5 flex flex-col gap-3 shadow-[var(--shadow-card)]">
      <h3 className="text-[13px] font-semibold text-[var(--color-text-primary)] mb-1">
        Supported Formats
      </h3>
      <div className="flex flex-col gap-3">
        {formats.map((fmt) => (
          <div key={fmt.name} className="flex items-start gap-3">
            <div
              className="mt-1 flex-shrink-0 rounded-sm"
              style={{
                width: "3px",
                height: "20px",
                backgroundColor: "var(--color-primary)",
              }}
            />
            <div className="flex flex-col gap-0.5">
              <span className="text-[13px] font-medium text-[var(--color-text-primary)]">
                {fmt.name}
              </span>
              <span className="text-[11px] text-[var(--color-text-muted)]">
                {fmt.subtitle}
              </span>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}

// ── Toggle component ──────────────────────────────────────────────────────────

interface ToggleProps {
  checked: boolean;
  onChange: (val: boolean) => void;
}

function Toggle({ checked, onChange }: ToggleProps) {
  return (
    <div
      role="switch"
      aria-checked={checked}
      onClick={() => onChange(!checked)}
      className="relative cursor-pointer flex-shrink-0"
      style={{
        width: "44px",
        height: "24px",
        borderRadius: "9999px",
        backgroundColor: checked ? "var(--color-primary)" : "var(--color-bg-accent)",
        transition: "background-color 0.2s",
      }}
    >
      <div
        style={{
          position: "absolute",
          top: "2px",
          left: checked ? "22px" : "2px",
          width: "20px",
          height: "20px",
          borderRadius: "9999px",
          backgroundColor: "white",
          transition: "left 0.2s",
        }}
      />
    </div>
  );
}

// ── SubscriptionForm ──────────────────────────────────────────────────────────

interface SubscriptionFormProps {
  editTarget: Subscription | null;
  form: CreateSubscriptionInput;
  setForm: React.Dispatch<React.SetStateAction<CreateSubscriptionInput>>;
  formError: string | null;
  submitting: boolean;
  onSubmit: (e: React.FormEvent) => void;
  onCancel: () => void;
}

function SubscriptionForm({
  editTarget,
  form,
  setForm,
  formError,
  submitting,
  onSubmit,
  onCancel,
}: SubscriptionFormProps) {
  return (
    <div className="grid grid-cols-1 lg:grid-cols-[1fr_280px] gap-6">
      {/* Left: main form card */}
      <form
        onSubmit={onSubmit}
        className="bg-[var(--color-bg-card)] rounded-xl border border-[var(--color-border)]/50 p-6 flex flex-col gap-5 shadow-[var(--shadow-card)]"
      >
        <h2 className="text-[20px] font-semibold text-[var(--color-text-primary)]">
          {editTarget ? "Edit Subscription" : "Add Subscription"}
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
            placeholder="e.g. Airport Premium"
            className={inputClass}
          />
        </div>

        {/* Subscription URL */}
        <div className="flex flex-col gap-1.5">
          <label className="text-[12px] font-medium text-[var(--color-text-muted)] uppercase tracking-wide">
            Subscription URL
          </label>
          <input
            type="url"
            required
            value={form.url}
            onChange={(e) => setForm((f) => ({ ...f, url: e.target.value }))}
            placeholder="https://example.com/subscribe?token=xxx"
            className={inputClass}
          />
        </div>

        {/* Type */}
        <div className="flex flex-col gap-1.5">
          <label className="text-[12px] font-medium text-[var(--color-text-muted)] uppercase tracking-wide">
            Type
          </label>
          <select
            value={form.type}
            onChange={(e) => setForm((f) => ({ ...f, type: e.target.value }))}
            className={inputClass}
          >
            <option value="clash">Clash</option>
            <option value="v2ray">V2Ray</option>
            <option value="sip002">SIP002</option>
          </select>
        </div>

        {/* Auto Refresh toggle */}
        <div className="flex flex-col gap-1.5">
          <div className="flex items-center justify-between">
            <label className="text-[12px] font-medium text-[var(--color-text-muted)] uppercase tracking-wide">
              Auto Refresh
            </label>
            <Toggle
              checked={form.auto_refresh}
              onChange={(val) => setForm((f) => ({ ...f, auto_refresh: val }))}
            />
          </div>
        </div>

        {/* Refresh Cron Expression — only when auto_refresh is on */}
        {form.auto_refresh && (
          <div className="flex flex-col gap-1.5">
            <label className="text-[12px] font-medium text-[var(--color-text-muted)] uppercase tracking-wide">
              Refresh Cron Expression
            </label>
            <input
              type="text"
              value={form.refresh_cron}
              onChange={(e) => setForm((f) => ({ ...f, refresh_cron: e.target.value }))}
              placeholder="0 */6 * * *"
              className={inputClass}
            />
            <p className="text-[11px] text-[var(--color-text-muted)]">
              Runs every 6 hours
            </p>
          </div>
        )}

        {/* Custom User-Agent */}
        <div className="flex flex-col gap-1.5">
          <label className="text-[12px] font-medium text-[var(--color-text-muted)] uppercase tracking-wide">
            Custom User-Agent (optional)
          </label>
          <input
            type="text"
            placeholder="ClashForAndroid/2.5.12"
            className={inputClass}
          />
        </div>

        {/* Action row */}
        <div className="flex gap-3 items-center pt-1">
          <Button type="submit" disabled={submitting}>
            {submitting ? (
              <>
                <Loader2 className="h-4 w-4 animate-spin" />
                {editTarget ? "Saving…" : "Creating…"}
              </>
            ) : editTarget ? (
              "Save Changes"
            ) : (
              "Create & Fetch"
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

      {/* Right: Supported Formats side panel */}
      <SupportedFormatsPanel />
    </div>
  );
}

// ── SubscriptionsPage ─────────────────────────────────────────────────────────

export function SubscriptionsPage() {
  const { token } = useAuth();
  const { showSuccess, showError } = useToast();
  const [subscriptions, setSubscriptions] = useState<Subscription[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [creating, setCreating] = useState(false);
  const [editTarget, setEditTarget] = useState<Subscription | null>(null);
  const [form, setForm] = useState<CreateSubscriptionInput>(EMPTY_FORM);
  const [formError, setFormError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);
  const [actionLoading, setActionLoading] = useState<Record<number, string>>({});
  const [confirmDelete, setConfirmDelete] = useState<{ id: number; name: string } | null>(null);
  const [filterType, setFilterType] = useState<string>("All");
  const [search, setSearch] = useState("");
  const [page, setPage] = useState(1);
  const PAGE_SIZE = 20;

  function loadSubscriptions() {
    let cancelled = false;
    setLoading(true);
    setError(null);
    fetchSubscriptions({ token })
      .then((data) => {
        if (!cancelled) {
          setSubscriptions(data);
          setLoading(false);
        }
      })
      .catch((err: unknown) => {
        if (!cancelled) {
          setError(
            err instanceof Error ? err.message : "Failed to load subscriptions"
          );
          setLoading(false);
        }
      });
    return () => {
      cancelled = true;
    };
  }

  useEffect(() => {
    loadSubscriptions();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  function openCreate() {
    setEditTarget(null);
    setForm(EMPTY_FORM);
    setFormError(null);
    setCreating(true);
  }

  function openEdit(sub: Subscription) {
    setEditTarget(sub);
    setForm({
      name: sub.name,
      url: sub.url,
      type: sub.type,
      auto_refresh: sub.auto_refresh,
      refresh_cron: sub.refresh_cron,
    });
    setFormError(null);
    setCreating(true);
  }

  function closeForm() {
    setCreating(false);
    setEditTarget(null);
    setForm(EMPTY_FORM);
    setFormError(null);
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!form.name.trim() || !form.url.trim()) {
      setFormError("Name and URL are required.");
      return;
    }
    setSubmitting(true);
    setFormError(null);
    try {
      if (editTarget) {
        const updated = await updateSubscription(editTarget.id, form, { token });
        setSubscriptions((prev) =>
          prev.map((s) => (s.id === updated.id ? updated : s))
        );
        showSuccess("Subscription updated successfully");
      } else {
        await createSubscription(form, { token });
        showSuccess("Subscription created successfully");
        loadSubscriptions();
      }
      closeForm();
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : "Failed to save subscription";
      setFormError(msg);
      showError(msg);
    } finally {
      setSubmitting(false);
    }
  }

  async function handleFetch(id: number) {
    setActionLoading((prev) => ({ ...prev, [id]: "fetch" }));
    try {
      const updated = await triggerFetch(id, { token });
      setSubscriptions((prev) =>
        prev.map((s) => (s.id === updated.id ? updated : s))
      );
      showSuccess("Subscription fetched successfully");
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : `Failed to fetch subscription ${id}`;
      setError(msg);
      showError(msg);
    } finally {
      setActionLoading((prev) => {
        const next = { ...prev };
        delete next[id];
        return next;
      });
    }
  }

  async function handleDelete(id: number, name: string) {
    setConfirmDelete({ id, name });
  }

  async function onConfirmDelete() {
    if (!confirmDelete) return;
    const { id } = confirmDelete;
    setConfirmDelete(null);
    setActionLoading((prev) => ({ ...prev, [id]: "delete" }));
    try {
      await deleteSubscription(id, { token });
      setSubscriptions((prev) => prev.filter((s) => s.id !== id));
      showSuccess("Subscription deleted successfully");
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : `Failed to delete subscription ${id}`;
      setError(msg);
      showError(msg);
    } finally {
      setActionLoading((prev) => {
        const next = { ...prev };
        delete next[id];
        return next;
      });
    }
  }

  const filtered = subscriptions.filter((s) =>
    (filterType === "All" || s.type.toLowerCase() === filterType.toLowerCase()) &&
    (search === "" || s.name.toLowerCase().includes(search.toLowerCase()))
  );

  const totalPages = Math.max(1, Math.ceil(filtered.length / PAGE_SIZE));
  const paged = filtered.slice((page - 1) * PAGE_SIZE, page * PAGE_SIZE);

  // Reset page when filters change
  useEffect(() => setPage(1), [filterType, search]);

  return (
    <div className="flex flex-col gap-6">
      {/* Page header */}
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-[26px] font-bold text-[var(--color-text-primary)]">Subscriptions</h1>
          <p className="text-[13px] text-[var(--color-text-secondary)]">Manage your proxy subscription sources</p>
        </div>
        <Button
          className="hidden md:flex"
          onClick={creating ? closeForm : openCreate}
        >
          {creating ? "Cancel" : "+ Add Subscription"}
        </Button>
      </div>

      {/* Error banner */}
      {error && (
        <div className="rounded-[var(--radius)] border border-[var(--color-danger)]/30 bg-[var(--color-danger-bg)] px-4 py-3 text-sm text-[var(--color-danger)]">
          <strong>Error:</strong> {error}
        </div>
      )}

      {/* Inline create/edit form */}
      {creating && (
        <SubscriptionForm
          editTarget={editTarget}
          form={form}
          setForm={setForm}
          formError={formError}
          submitting={submitting}
          onSubmit={handleSubmit}
          onCancel={closeForm}
        />
      )}

      {/* Search + filter chips bar */}
      <div className="flex flex-col sm:flex-row gap-3 mb-6">
        <input
          type="text"
          placeholder="Search subscriptions..."
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          className="flex-1 h-9 rounded-[var(--radius)] bg-[var(--color-bg-card)] border border-[var(--color-border)] px-3 text-[13px] text-[var(--color-text-primary)] placeholder:text-[var(--color-text-muted)] outline-none focus:border-[var(--color-primary)] transition-colors"
        />
        <div className="flex gap-2">
          {FILTER_TYPES.map((t) => (
            <button
              key={t}
              onClick={() => setFilterType(t)}
              className={[
                "px-3 h-9 rounded-[var(--radius)] text-[13px] font-medium transition-colors",
                filterType === t
                  ? "bg-[var(--color-primary-bg)] text-[var(--color-primary-light)] border border-[var(--color-primary)]/30"
                  : "bg-[var(--color-bg-card)] text-[var(--color-text-secondary)] border border-[var(--color-border)] hover:bg-[var(--color-bg-accent)]",
              ].join(" ")}
            >
              {t}
            </button>
          ))}
        </div>
      </div>

      {/* Loading skeleton */}
      {loading ? (
        <div className="flex flex-col gap-3">
          {[...Array(3)].map((_, i) => (
            <div
              key={i}
              className="h-32 rounded-[var(--radius-lg)] animate-pulse bg-[var(--color-bg-card)]"
            />
          ))}
        </div>
      ) : filtered.length === 0 ? (
        <div className="rounded-[var(--radius-lg)] border border-[var(--color-border)]/50 px-6 py-12 text-center text-sm text-[var(--color-text-muted)]">
          {subscriptions.length === 0
            ? "No subscriptions yet. Create one to get started."
            : "No subscriptions match your search."}
        </div>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          {paged.map((sub) => {
            const busy = actionLoading[sub.id];
            return (
              <div
                key={sub.id}
                className="bg-[var(--color-bg-card)] rounded-[var(--radius-lg)] border border-[var(--color-border)]/50 shadow-[var(--shadow-card)] p-5 flex flex-col gap-3"
              >
                {/* Name + status */}
                <div className="flex items-start justify-between gap-2">
                  <Link
                    to={`/subscriptions/${sub.id}`}
                    className="text-[15px] font-semibold text-[var(--color-text-primary)] hover:text-[var(--color-primary-light)] transition-colors leading-tight"
                  >
                    {sub.name}
                  </Link>
                  <Badge variant={statusVariant(sub.status)}>{sub.status}</Badge>
                </div>

                {/* Type badge */}
                <div>
                  <Badge variant="type">{sub.type}</Badge>
                </div>

                {/* Stats row */}
                <div className="flex items-center gap-4 text-[13px] text-[var(--color-text-secondary)]">
                  <span>{sub.node_count} nodes</span>
                  <span className="text-[12px] text-[var(--color-text-muted)]">{formatRefresh(sub)}</span>
                </div>

                {/* Footer actions */}
                <div className="border-t border-[var(--color-border)]/30 pt-3 mt-1 flex gap-2 justify-end">
                  <Button
                    variant="secondary"
                    size="sm"
                    disabled={!!busy}
                    onClick={() => openEdit(sub)}
                  >
                    Edit
                  </Button>
                  <Button
                    variant="secondary"
                    size="sm"
                    disabled={!!busy}
                    onClick={() => handleFetch(sub.id)}
                  >
                    {busy === "fetch" ? <><Loader2 className="h-3 w-3 animate-spin" />Fetching…</> : "Fetch"}
                  </Button>
                  <Button
                    variant="destructive"
                    size="sm"
                    disabled={!!busy}
                    onClick={() => handleDelete(sub.id, sub.name)}
                  >
                    {busy === "delete" ? <><Loader2 className="h-3 w-3 animate-spin" />Deleting…</> : "Delete"}
                  </Button>
                </div>
              </div>
            );
          })}
        </div>
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

      {/* Mobile FAB */}
      <button
        className="fixed bottom-6 right-6 z-40 md:hidden w-12 h-12 rounded-full bg-[var(--color-primary)] text-[var(--color-bg)] flex items-center justify-center shadow-lg text-xl font-bold"
        onClick={openCreate}
        aria-label="Add subscription"
      >
        +
      </button>

      <ConfirmDialog
        open={!!confirmDelete}
        title="Delete Subscription"
        message={
          confirmDelete ? (
            <>
              Delete subscription <strong>{confirmDelete.name}</strong>? This cannot be undone.
            </>
          ) : null
        }
        onConfirm={onConfirmDelete}
        onCancel={() => setConfirmDelete(null)}
      />
    </div>
  );
}
