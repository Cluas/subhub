import { useEffect, useState } from "react";
import { Loader2, Plus } from "lucide-react";
import { Pagination } from "@/components/ui/pagination";
import { Link, useNavigate } from "react-router-dom";
import { fetchProfiles, createProfile, deleteProfile } from "@/lib/api";
import { useAuth } from "@/context/AuthContext";
import type { Profile, CreateProfileInput } from "@/types/api";
import { Button } from "@/components/ui/button";
import { CopyButton } from "@/components/ui/copy-button";
import { ConfirmDialog } from "@/components/ui/confirm-dialog";
import { useToast } from "@/components/ui/toast";

function profileUrl(slug: string) {
  return `${window.location.origin}/profile/${slug}`;
}

// ── ProfilesPage ──────────────────────────────────────────────────────────────

export function ProfilesPage() {
  const navigate = useNavigate();
  const { token } = useAuth();
  const { showSuccess, showError } = useToast();
  const [profiles, setProfiles] = useState<Profile[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [deleting, setDeleting] = useState<Record<number, boolean>>({});
  const [confirmDelete, setConfirmDelete] = useState<{ id: number; name: string } | null>(null);
  const [creating, setCreating] = useState(false);
  const [formName, setFormName] = useState("");
  const [formSettings, setFormSettings] = useState('{\n  "port": 7890,\n  "socks-port": 7891,\n  "allow-lan": true,\n  "mode": "rule",\n  "log-level": "info",\n  "external-controller": "127.0.0.1:9090"\n}');
  const [submitting, setSubmitting] = useState(false);
  const [formError, setFormError] = useState<string | null>(null);
  const [page, setPage] = useState(1);
  const PAGE_SIZE = 20;

  const totalPages = Math.max(1, Math.ceil(profiles.length / PAGE_SIZE));
  const paged = profiles.slice((page - 1) * PAGE_SIZE, page * PAGE_SIZE);

  function load() {
    let cancelled = false;
    setLoading(true);
    setError(null);
    fetchProfiles({ token })
      .then((ps) => {
        if (!cancelled) {
          setProfiles(ps);
          setLoading(false);
        }
      })
      .catch((err: unknown) => {
        if (!cancelled) {
          setError(err instanceof Error ? err.message : "Failed to load profiles");
          setLoading(false);
        }
      });
    return () => {
      cancelled = true;
    };
  }

  useEffect(() => {
    return load();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  async function handleCreate(e: React.FormEvent) {
    e.preventDefault();
    if (!formName.trim()) return;
    setSubmitting(true);
    setFormError(null);
    try {
      let settings = {};
      if (formSettings.trim()) {
        try { settings = JSON.parse(formSettings); } catch { setFormError("Invalid JSON in settings"); setSubmitting(false); return; }
      }
      const created = await createProfile({ name: formName.trim(), settings } as CreateProfileInput, { token });
      showSuccess("Profile created");
      setCreating(false);
      setFormName("");
      navigate(`/profiles/${created.id}`);
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : "Failed to create profile";
      setFormError(msg);
      showError(msg);
    } finally {
      setSubmitting(false);
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
      await deleteProfile(id, { token });
      setProfiles((prev) => prev.filter((p) => p.id !== id));
      showSuccess("Profile deleted successfully");
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : `Failed to delete profile ${id}`;
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

  return (
    <div className="flex flex-col gap-6">
      {/* Header */}
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-[26px] font-bold text-[var(--color-text-primary)]">Profiles</h1>
          <p className="text-[13px] text-[var(--color-text-secondary)]">Manage your subscription profiles</p>
        </div>
        <Button className="hidden md:flex" onClick={() => { setCreating(v => !v); setFormName(""); setFormError(null); }}>
          {creating ? "Cancel" : "+ New Profile"}
        </Button>
      </div>

      {/* Error banner */}
      {error && (
        <div
          className="rounded-md border px-4 py-3 text-sm mb-4"
          style={{
            backgroundColor: "var(--color-danger-bg)",
            color: "var(--color-danger)",
            borderColor: "var(--color-danger)",
          }}
        >
          <strong>Error:</strong> {error}
        </div>
      )}

      {/* Create form */}
      {creating && (
        <form onSubmit={handleCreate} className="rounded-[var(--radius-lg)] border border-[var(--color-border)]/50 bg-[var(--color-bg-card)] p-6 flex flex-col gap-4 shadow-[var(--shadow-card)]">
          <h2 className="font-semibold text-base text-[var(--color-text-primary)]">New Profile</h2>
          {formError && <p className="text-sm text-[var(--color-danger)]">{formError}</p>}
          <div className="flex flex-col gap-1.5">
            <label className="text-sm font-medium text-[var(--color-text-secondary)]">Name <span className="text-[var(--color-danger)]">*</span></label>
            <input
              type="text" required value={formName}
              onChange={e => setFormName(e.target.value)}
              placeholder="e.g. Daily Driver"
              className="h-10 rounded-[var(--radius)] bg-[var(--color-bg)] border border-[var(--color-border)] px-3 text-[13px] text-[var(--color-text-primary)] placeholder:text-[var(--color-text-muted)] outline-none focus:border-[var(--color-primary)] transition-colors"
            />
          </div>
          <div className="flex flex-col gap-1.5">
            <label className="text-sm font-medium text-[var(--color-text-secondary)]">Settings (JSON)</label>
            <textarea
              rows={8} value={formSettings}
              onChange={e => setFormSettings(e.target.value)}
              className="rounded-[var(--radius)] bg-[var(--color-bg)] border border-[var(--color-border)] px-3 py-2 text-[12px] font-mono text-[var(--color-text-primary)] placeholder:text-[var(--color-text-muted)] outline-none focus:border-[var(--color-primary)] transition-colors resize-y"
            />
            <p className="text-[11px] text-[var(--color-text-muted)]">Clash top-level settings: port, socks-port, allow-lan, mode, log-level, external-controller</p>
          </div>
          <div className="flex gap-2 justify-end">
            <Button type="button" variant="secondary" onClick={() => setCreating(false)}>Cancel</Button>
            <Button type="submit" disabled={submitting}>{submitting ? "Creating..." : "Create Profile"}</Button>
          </div>
        </form>
      )}

      {/* Loading skeleton */}
      {loading ? (
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          {[...Array(4)].map((_, i) => (
            <div
              key={i}
              className="h-40 rounded-xl animate-pulse"
              style={{ backgroundColor: "var(--color-bg-card)" }}
            />
          ))}
        </div>
      ) : profiles.length === 0 ? (
        <div
          className="rounded-xl border px-6 py-12 text-center text-[13px]"
          style={{
            color: "var(--color-text-muted)",
            borderColor: "var(--color-border)",
            backgroundColor: "var(--color-bg-card)",
          }}
        >
          No profiles yet. Create one to get started.
        </div>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          {paged.map((p) => (
            <div
              key={p.id}
              className="rounded-xl border p-5"
              style={{
                backgroundColor: "var(--color-bg-card)",
                borderColor: "var(--color-border)",
              }}
            >
              {/* Name + description */}
              <div className="text-[15px] font-semibold text-[var(--color-text-primary)]">{p.name}</div>
              <div className="text-[13px] text-[var(--color-text-secondary)] mt-1">
                {new Date(p.created_at).toLocaleDateString()}
              </div>

              {/* Profile serve URL */}
              <div className="flex items-center gap-2 mt-3 bg-[var(--color-bg)] rounded-[var(--radius)] border border-[var(--color-border)] px-3 py-2">
                <span className="flex-1 text-[12px] font-mono text-[var(--color-text-muted)] truncate">
                  {profileUrl(p.slug)}
                </span>
                <CopyButton text={profileUrl(p.slug)} />
              </div>

              {/* 4-stat counter row */}
              <div
                className="mt-4 grid grid-cols-4 gap-3 border-t pt-4"
                style={{ borderColor: "var(--color-border)" }}
              >
                {[
                  { label: "Node Pools", value: p.node_pool_count ?? 0 },
                  { label: "Rule Sets", value: p.rule_set_count ?? 0 },
                  { label: "Strategies", value: p.strategy_count ?? 0 },
                  { label: "Routing", value: p.routing_rule_count ?? 0 },
                ].map((stat) => (
                  <div key={stat.label} className="text-center">
                    <div className="text-[22px] font-bold text-[var(--color-primary-light)]">{stat.value}</div>
                    <div className="text-[11px] text-[var(--color-text-muted)] mt-0.5">{stat.label}</div>
                  </div>
                ))}
              </div>

              {/* Action footer */}
              <div className="mt-4 flex gap-2">
                <Link to={`/profiles/${p.id}`}>
                  <Button variant="secondary" size="sm">Edit</Button>
                </Link>
                <Button
                  variant="destructive"
                  size="sm"
                  disabled={!!deleting[p.id]}
                  onClick={() => handleDelete(p.id, p.name)}
                >
                  {deleting[p.id] ? <><Loader2 className="h-3 w-3 animate-spin" />Deleting…</> : "Delete"}
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
          totalItems={profiles.length}
          onPageChange={setPage}
        />
      )}

      {/* Mobile FAB */}
      <button
        className="md:hidden fixed bottom-6 right-6 w-12 h-12 rounded-full flex items-center justify-center shadow-lg z-50"
        style={{ backgroundColor: "var(--color-primary)" }}
        onClick={() => { setCreating(v => !v); setFormName(""); setFormError(null); }}
        aria-label="New Profile"
      >
        <Plus className="h-5 w-5 text-black" />
      </button>

      <ConfirmDialog
        open={!!confirmDelete}
        title="Delete Profile"
        message={
          confirmDelete ? (
            <>
              Delete profile <strong>{confirmDelete.name}</strong>? This cannot be undone.
            </>
          ) : null
        }
        onConfirm={onConfirmDelete}
        onCancel={() => setConfirmDelete(null)}
      />
    </div>
  );
}
