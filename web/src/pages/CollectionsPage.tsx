import { useCallback, useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";
import { Loader2 } from "lucide-react";
import { Pagination } from "@/components/ui/pagination";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { ConfirmDialog } from "@/components/ui/confirm-dialog";
import { useToast } from "@/components/ui/toast";
import { useAuth } from "@/context/AuthContext";
import {
  fetchCollections,
  createCollection,
  updateCollection,
  deleteCollection,
} from "@/lib/api";
import type { Collection, CreateCollectionInput } from "@/types/api";

const EMPTY_FORM: CreateCollectionInput = {
  name: "",
  content_type: "proxy",
  description: "",
};

const inputClass =
  "h-10 w-full rounded-lg bg-[var(--color-bg)] border border-[var(--color-border)] px-3 text-[13px] text-[var(--color-text-secondary)] outline-none focus:border-[var(--color-primary)] placeholder:text-[var(--color-text-muted)] transition-colors";

// ── Side panel: Add Items After Creation ──────────────────────────────────────

function AddItemsGuidePanel() {
  const steps = [
    "Browse subscriptions",
    "Filter by region/type",
    "Select and add items",
    "Manage & reorder",
  ];
  return (
    <div className="bg-[var(--color-bg-card)] rounded-xl border border-[var(--color-border)]/50 p-5 flex flex-col gap-4 shadow-[var(--shadow-card)]">
      <h3 className="text-[13px] font-semibold text-[var(--color-text-primary)]">
        Add Items After Creation
      </h3>
      <p className="text-[12px] text-[var(--color-text-muted)] -mt-2">
        Once created, you can add proxies or rules from your subscriptions to this collection.
      </p>
      <div className="flex flex-col gap-3">
        {steps.map((step, i) => (
          <div key={i} className="flex items-start gap-3">
            <div
              className="flex-shrink-0 flex items-center justify-center rounded-full text-[12px] font-semibold"
              style={{
                width: "20px",
                height: "20px",
                backgroundColor: "var(--color-primary-bg)",
                color: "var(--color-primary-light)",
              }}
            >
              {i + 1}
            </div>
            <span className="text-[13px] text-[var(--color-text-secondary)] leading-snug pt-px">
              {step}
            </span>
          </div>
        ))}
      </div>
    </div>
  );
}

// ── CollectionsPage ───────────────────────────────────────────────────────────

export function CollectionsPage() {
  const { token } = useAuth();
  const navigate = useNavigate();
  const { showSuccess, showError } = useToast();

  const [collections, setCollections] = useState<Collection[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [showForm, setShowForm] = useState(false);
  const [editTarget, setEditTarget] = useState<Collection | null>(null);
  const [form, setForm] = useState<CreateCollectionInput>(EMPTY_FORM);
  const [submitting, setSubmitting] = useState(false);
  const [formError, setFormError] = useState<string | null>(null);
  const [confirmDelete, setConfirmDelete] = useState<{ id: number; name: string } | null>(null);
  const [page, setPage] = useState(1);
  const PAGE_SIZE = 20;

  const totalPages = Math.max(1, Math.ceil(collections.length / PAGE_SIZE));
  const paged = collections.slice((page - 1) * PAGE_SIZE, page * PAGE_SIZE);

  const loadCollections = useCallback(() => {
    let cancelled = false;
    setLoading(true);
    setError(null);
    fetchCollections({ token })
      .then((data) => {
        if (!cancelled) {
          setCollections(data);
          setLoading(false);
        }
      })
      .catch((err: unknown) => {
        if (!cancelled) {
          setError(err instanceof Error ? err.message : "Failed to load collections");
          setLoading(false);
        }
      });
    return () => { cancelled = true; };
  }, [token]);

  useEffect(() => {
    return loadCollections();
  }, [loadCollections]);

  function openCreate() {
    setEditTarget(null);
    setForm(EMPTY_FORM);
    setFormError(null);
    setShowForm(true);
  }

  function openEdit(col: Collection) {
    setEditTarget(col);
    setForm({
      name: col.name,
      content_type: col.content_type,
      description: col.description ?? "",
    });
    setFormError(null);
    setShowForm(true);
  }

  function closeForm() {
    setShowForm(false);
    setEditTarget(null);
    setForm(EMPTY_FORM);
    setFormError(null);
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setFormError(null);
    if (!form.name.trim()) {
      setFormError("Name is required");
      return;
    }
    setSubmitting(true);
    try {
      if (editTarget) {
        await updateCollection(
          editTarget.id,
          { name: form.name.trim(), description: form.description },
          { token }
        );
        showSuccess("Collection updated");
      } else {
        await createCollection({ ...form, name: form.name.trim() }, { token });
        showSuccess("Collection created");
      }
      closeForm();
      loadCollections();
    } catch (err) {
      showError(err instanceof Error ? err.message : "Failed to save collection");
    } finally {
      setSubmitting(false);
    }
  }

  async function onConfirmDelete() {
    if (!confirmDelete) return;
    const { id } = confirmDelete;
    setConfirmDelete(null);
    try {
      await deleteCollection(id, { token });
      showSuccess("Collection deleted");
      loadCollections();
    } catch (err) {
      showError(err instanceof Error ? err.message : "Failed to delete collection");
    }
  }

  return (
    <div className="flex flex-col gap-6">
      {/* Page header */}
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-[26px] font-bold text-[var(--color-text-primary)]">Collections</h1>
          <p className="text-[13px] text-[var(--color-text-secondary)]">Organize nodes into reusable collection groups</p>
        </div>
        <Button
          className="hidden md:flex"
          onClick={showForm ? closeForm : openCreate}
        >
          {showForm ? "Cancel" : "New Collection"}
        </Button>
      </div>

      {/* Error banner */}
      {error && (
        <div className="rounded-[var(--radius)] border border-[var(--color-danger)]/30 bg-[var(--color-danger-bg)] px-4 py-3 text-sm text-[var(--color-danger)]">
          <strong>Error:</strong> {error}
        </div>
      )}

      {/* Inline create/edit form — two-column */}
      {showForm && (
        <div className="grid grid-cols-1 lg:grid-cols-[1fr_280px] gap-6">
          {/* Left: main form */}
          <form
            onSubmit={handleSubmit}
            className="bg-[var(--color-bg-card)] rounded-xl border border-[var(--color-border)]/50 p-6 flex flex-col gap-5 shadow-[var(--shadow-card)]"
          >
            <h2 className="text-[20px] font-semibold text-[var(--color-text-primary)]">
              {editTarget ? "Edit Collection" : "Create Collection"}
            </h2>

            {formError && (
              <p className="text-[13px] text-[var(--color-danger)]">{formError}</p>
            )}

            {/* Collection Name */}
            <div className="flex flex-col gap-1.5">
              <label className="text-[12px] font-medium text-[var(--color-text-muted)] uppercase tracking-wide">
                Collection Name
              </label>
              <input
                className={inputClass}
                value={form.name}
                onChange={(e) => setForm((f) => ({ ...f, name: e.target.value }))}
                placeholder="e.g. Asia Premium Nodes"
                autoFocus
              />
            </div>

            {/* Type */}
            <div className="flex flex-col gap-1.5">
              <label className="text-[12px] font-medium text-[var(--color-text-muted)] uppercase tracking-wide">
                Type
              </label>
              <select
                className={inputClass}
                value={form.content_type}
                onChange={(e) =>
                  setForm((f) => ({ ...f, content_type: e.target.value as "proxy" | "rule" }))
                }
                disabled={!!editTarget}
              >
                <option value="proxy">Proxy</option>
                <option value="rule">Rule</option>
              </select>
            </div>

            {/* Description */}
            <div className="flex flex-col gap-1.5">
              <label className="text-[12px] font-medium text-[var(--color-text-muted)] uppercase tracking-wide">
                Description (optional)
              </label>
              <textarea
                className="w-full rounded-lg bg-[var(--color-bg)] border border-[var(--color-border)] px-3 py-2 text-[13px] text-[var(--color-text-secondary)] outline-none focus:border-[var(--color-primary)] placeholder:text-[var(--color-text-muted)] transition-colors resize-none"
                rows={3}
                value={form.description}
                onChange={(e) => setForm((f) => ({ ...f, description: e.target.value }))}
                placeholder="Describe what this collection is for..."
              />
            </div>

            {/* Action row */}
            <div className="flex gap-3 items-center pt-1">
              <Button type="submit" disabled={submitting}>
                {submitting && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
                {editTarget ? "Save" : "Create Collection"}
              </Button>
              <button
                type="button"
                onClick={closeForm}
                className="text-[13px] text-[var(--color-text-muted)] hover:text-[var(--color-text-secondary)] transition-colors"
              >
                Cancel
              </button>
            </div>
          </form>

          {/* Right: guide panel */}
          <AddItemsGuidePanel />
        </div>
      )}

      {/* Collections grid */}
      {loading ? (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
          {[...Array(3)].map((_, i) => (
            <div key={i} className="h-40 rounded-[var(--radius-lg)] animate-pulse bg-[var(--color-bg-card)]" />
          ))}
        </div>
      ) : collections.length === 0 ? (
        <div className="rounded-[var(--radius-lg)] border border-[var(--color-border)]/50 px-6 py-12 text-center text-sm text-[var(--color-text-muted)]">
          No collections yet. Create one to get started.
        </div>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
          {paged.map((col) => (
            <div
              key={col.id}
              className="bg-[var(--color-bg-card)] rounded-[var(--radius-lg)] border border-[var(--color-border)]/50 shadow-[var(--shadow-card)] p-5 flex flex-col gap-2"
            >
              {/* Name */}
              <div className="flex items-start justify-between gap-2">
                <span className="text-[15px] font-semibold text-[var(--color-text-primary)] leading-tight">
                  {col.name}
                </span>
              </div>

              {/* Type badge */}
              <div>
                <Badge variant="type">{col.content_type}</Badge>
              </div>

              {/* Description */}
              <p className="text-[13px] text-[var(--color-text-secondary)] mt-2 line-clamp-2 flex-1">
                {col.description || <span className="text-[var(--color-text-muted)]">No description</span>}
              </p>

              {/* Footer */}
              <div className="border-t border-[var(--color-border)]/30 pt-3 mt-2 flex gap-2">
                <Button
                  variant="secondary"
                  size="sm"
                  onClick={() => navigate(`/collections/${col.id}`)}
                >
                  View Details
                </Button>
                <Button
                  variant="secondary"
                  size="sm"
                  onClick={() => openEdit(col)}
                >
                  Edit
                </Button>
                <Button
                  variant="destructive"
                  size="sm"
                  onClick={() => setConfirmDelete({ id: col.id, name: col.name })}
                >
                  Delete
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
          totalItems={collections.length}
          onPageChange={setPage}
        />
      )}

      {/* Mobile FAB */}
      <button
        className="fixed bottom-6 right-6 z-40 md:hidden w-12 h-12 rounded-full bg-[var(--color-primary)] text-[var(--color-bg)] flex items-center justify-center shadow-lg text-xl font-bold"
        onClick={openCreate}
        aria-label="New collection"
      >
        +
      </button>

      <ConfirmDialog
        open={!!confirmDelete}
        title="Delete Collection"
        message={`Delete "${confirmDelete?.name}"? All nodes/rules inside will be deleted too.`}
        confirmLabel="Delete"
        onConfirm={onConfirmDelete}
        onCancel={() => setConfirmDelete(null)}
      />
    </div>
  );
}
