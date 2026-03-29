import { useEffect, useState, useCallback } from "react";
import { Loader2, Plus, Trash2, Save, ChevronLeft, ChevronDown, ChevronRight, GripVertical } from "lucide-react";
import { useParams, useNavigate, Link } from "react-router-dom";
import {
  getProfile,
  fetchProfileNodePools,
  fetchProfileRuleSets,
  fetchProfileStrategies,
  fetchProfileRoutingRules,
  createProfileNodePool,
  createProfileRuleSet,
  createProfileStrategy,
  createProfileRoutingRule,
  deleteProfileNodePool,
  deleteProfileRuleSet,
  deleteProfileStrategy,
  deleteProfileRoutingRule,
  fetchEndpoints,
  apiFetch,
} from "@/lib/api";
import { useAuth } from "@/context/AuthContext";
import type {
  Profile,
  ProfileNodePool,
  ProfileRuleSet,
  ProfileStrategy,
  ProfileRoutingRule,
  Endpoint,
} from "@/types/api";
import { Button } from "@/components/ui/button";
import { CopyButton } from "@/components/ui/copy-button";

// ── Tab definitions ───────────────────────────────────────────────────────────

type TabId = "node-pools" | "rule-sets" | "strategies" | "routing-rules" | "settings";

const TABS: { id: TabId; label: string }[] = [
  { id: "node-pools", label: "Node Pools" },
  { id: "rule-sets", label: "Rule Sets" },
  { id: "strategies", label: "Strategies" },
  { id: "routing-rules", label: "Routing Rules" },
  { id: "settings", label: "Settings" },
];

// ── Shared inline styles ──────────────────────────────────────────────────────

const inputStyle: React.CSSProperties = {
  backgroundColor: "var(--color-bg-accent)",
  border: "1px solid var(--color-border)",
  color: "var(--color-text-primary)",
};

const selectStyle: React.CSSProperties = {
  ...inputStyle,
  appearance: "none" as const,
  backgroundImage: `url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='12' height='12' viewBox='0 0 24 24' fill='none' stroke='%2378716C' stroke-width='2'%3E%3Cpolyline points='6 9 12 15 18 9'%3E%3C/polyline%3E%3C/svg%3E")`,
  backgroundRepeat: "no-repeat",
  backgroundPosition: "right 8px center",
  paddingRight: "28px",
};

// ── Shared card skeleton ──────────────────────────────────────────────────────

function CardSkeleton() {
  return (
    <div className="flex flex-col gap-3">
      {[0, 1, 2].map((i) => (
        <div
          key={i}
          className="h-16 rounded-xl animate-pulse"
          style={{ backgroundColor: "var(--color-bg-accent)" }}
        />
      ))}
    </div>
  );
}

function CardError({ message }: { message: string }) {
  return (
    <div
      className="rounded-xl border px-4 py-3 text-[13px]"
      style={{
        backgroundColor: "var(--color-danger-bg)",
        color: "var(--color-danger)",
        borderColor: "var(--color-danger-bg)",
      }}
    >
      {message}
    </div>
  );
}

function CardEmpty({ message }: { message: string }) {
  return (
    <div
      className="rounded-xl border px-6 py-12 text-center text-[13px]"
      style={{ color: "var(--color-text-muted)", borderColor: "var(--color-border)" }}
    >
      {message}
    </div>
  );
}

// ── Settings tab ──────────────────────────────────────────────────────────────

interface KVRow {
  key: string;
  value: string;
}

function settingsToRows(settings: Record<string, unknown> | null | undefined): KVRow[] {
  if (!settings) return [];
  return Object.entries(settings).map(([k, v]) => ({
    key: k,
    value: typeof v === "string" ? v : String(v),
  }));
}

function rowsToSettings(rows: KVRow[]): Record<string, unknown> {
  const out: Record<string, unknown> = {};
  for (const { key, value } of rows) {
    if (!key.trim()) continue;
    // Try to coerce to number/bool for well-known numeric keys
    const n = Number(value);
    if (value !== "" && !isNaN(n) && String(n) === value) {
      out[key] = n;
    } else if (value === "true") {
      out[key] = true;
    } else if (value === "false") {
      out[key] = false;
    } else {
      out[key] = value;
    }
  }
  return out;
}

interface SettingsTabProps {
  profile: Profile;
  token?: string;
  onSaved: (updated: Profile) => void;
}

function SettingsTab({ profile, token, onSaved }: SettingsTabProps) {
  const [rows, setRows] = useState<KVRow[]>(() => settingsToRows(profile.settings));
  const [saving, setSaving] = useState(false);
  const [saveError, setSaveError] = useState<string | null>(null);
  const [saveSuccess, setSaveSuccess] = useState(false);

  // Reset rows when profile changes (e.g. navigated to different profile)
  useEffect(() => {
    setRows(settingsToRows(profile.settings));
  }, [profile.id, profile.settings]);

  function addRow() {
    setRows((prev) => [...prev, { key: "", value: "" }]);
  }

  function updateRow(index: number, field: "key" | "value", val: string) {
    setRows((prev) => prev.map((r, i) => (i === index ? { ...r, [field]: val } : r)));
  }

  function removeRow(index: number) {
    setRows((prev) => prev.filter((_, i) => i !== index));
  }

  async function handleSave() {
    setSaving(true);
    setSaveError(null);
    setSaveSuccess(false);
    try {
      const updated = await apiFetch<Profile>(`/api/profiles/${profile.id}`, {
        token,
        method: "PUT",
        body: { settings: rowsToSettings(rows) },
      });
      setSaveSuccess(true);
      onSaved(updated);
      setTimeout(() => setSaveSuccess(false), 2000);
    } catch (err: unknown) {
      setSaveError(err instanceof Error ? err.message : "Failed to save settings");
    } finally {
      setSaving(false);
    }
  }

  return (
    <div className="flex flex-col gap-4">
      <div className="flex items-center justify-between">
        <p className="text-[13px]" style={{ color: "var(--color-text-muted)" }}>
          Top-level Mihomo settings (mixed-port, mode, log-level, etc.)
        </p>
        <div className="flex items-center gap-2">
          {saveSuccess && (
            <span className="text-xs" style={{ color: "var(--color-success)" }}>
              Saved ✓
            </span>
          )}
          {saveError && (
            <span className="text-xs" style={{ color: "var(--color-danger)" }}>
              {saveError}
            </span>
          )}
          <Button size="sm" onClick={handleSave} disabled={saving}>
            {saving ? (
              <>
                <Loader2 className="h-3.5 w-3.5 animate-spin mr-1" />
                Saving…
              </>
            ) : (
              <>
                <Save className="h-3.5 w-3.5 mr-1" />
                Save
              </>
            )}
          </Button>
        </div>
      </div>

      {/* KV table */}
      <div
        className="rounded-xl border overflow-hidden"
        style={{ borderColor: "var(--color-border)", backgroundColor: "var(--color-bg-card)" }}
      >
        {rows.length === 0 ? (
          <div
            className="px-4 py-8 text-center text-[13px]"
            style={{ color: "var(--color-text-muted)" }}
          >
            No settings yet.{" "}
            <button
              className="underline"
              style={{ color: "var(--color-text-secondary)" }}
              onClick={addRow}
            >
              Add one
            </button>{" "}
            to configure Mihomo top-level fields.
          </div>
        ) : (
          <table className="w-full text-[13px]">
            <thead>
              <tr style={{ borderBottom: "1px solid var(--color-border)" }}>
                <th className="px-3 py-2 text-left font-medium text-[11px] uppercase tracking-wide" style={{ color: "var(--color-text-muted)", width: "40%" }}>
                  Key
                </th>
                <th className="px-3 py-2 text-left font-medium text-[11px] uppercase tracking-wide" style={{ color: "var(--color-text-muted)" }}>
                  Value
                </th>
                <th className="px-3 py-2 w-10" />
              </tr>
            </thead>
            <tbody>
              {rows.map((row, i) => (
                <tr
                  key={i}
                  style={{ borderBottom: i < rows.length - 1 ? "1px solid var(--color-border)" : undefined }}
                >
                  <td className="px-3 py-1.5">
                    <input
                      value={row.key}
                      onChange={(e) => updateRow(i, "key", e.target.value)}
                      placeholder="key"
                      className="w-full rounded px-2 py-1 text-xs font-mono outline-none"
                      style={inputStyle}
                    />
                  </td>
                  <td className="px-3 py-1.5">
                    <input
                      value={row.value}
                      onChange={(e) => updateRow(i, "value", e.target.value)}
                      placeholder="value"
                      className="w-full rounded px-2 py-1 text-xs font-mono outline-none"
                      style={inputStyle}
                    />
                  </td>
                  <td className="px-3 py-1.5 text-center">
                    <button
                      onClick={() => removeRow(i)}
                      className="rounded p-1 opacity-80 hover:opacity-100 transition-opacity"
                      style={{ color: "var(--color-danger)" }}
                      title="Remove row"
                    >
                      <Trash2 className="h-3.5 w-3.5" />
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>

      <button
        onClick={addRow}
        className="flex items-center gap-1.5 text-[13px] w-fit rounded-lg px-3 py-1.5 border transition-colors"
        style={{
          borderColor: "var(--color-border)",
          color: "var(--color-text-muted)",
        }}
      >
        <Plus className="h-3.5 w-3.5" />
        Add setting
      </button>
    </div>
  );
}

// ── Node Pools Tab ────────────────────────────────────────────────────────────

interface NodePoolsTabProps {
  profileId: number;
  nodePools: ProfileNodePool[];
  loading: boolean;
  error: string | null;
  token?: string;
  endpoints: Endpoint[];
  onChanged: () => void;
}

function NodePoolsTab({ profileId, nodePools, loading, error, token, endpoints, onChanged }: NodePoolsTabProps) {
  const [showForm, setShowForm] = useState(false);
  const [formName, setFormName] = useState("");
  const [formEndpointId, setFormEndpointId] = useState<string>("");
  const [formHealthCheckUrl, setFormHealthCheckUrl] = useState("http://www.gstatic.com/generate_204");
  const [formHealthCheckInterval, setFormHealthCheckInterval] = useState(300);
  const [submitting, setSubmitting] = useState(false);
  const [formError, setFormError] = useState<string | null>(null);
  const [deleting, setDeleting] = useState<number | null>(null);

  function resetForm() {
    setFormName("");
    setFormEndpointId("");
    setFormHealthCheckUrl("http://www.gstatic.com/generate_204");
    setFormHealthCheckInterval(300);
    setFormError(null);
    setShowForm(false);
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!formName.trim()) { setFormError("Name is required"); return; }
    setSubmitting(true);
    setFormError(null);
    try {
      await createProfileNodePool(profileId, {
        name: formName.trim(),
        endpoint_id: formEndpointId ? Number(formEndpointId) : undefined,
        health_check_url: formHealthCheckUrl,
        health_check_interval: formHealthCheckInterval,
        position: nodePools.length + 1,
      }, { token });
      resetForm();
      onChanged();
    } catch (err: unknown) {
      setFormError(err instanceof Error ? err.message : "Failed to create node pool");
    } finally {
      setSubmitting(false);
    }
  }

  async function handleDelete(id: number) {
    setDeleting(id);
    try {
      await deleteProfileNodePool(profileId, id, { token });
      onChanged();
    } catch {
      // silently ignore
    } finally {
      setDeleting(null);
    }
  }

  return (
    <div className="flex flex-col gap-4">
      {/* Tab content header */}
      <div className="flex items-center justify-between">
        <span />
        <button
          onClick={() => setShowForm(true)}
          className="flex items-center gap-1.5 text-[13px] px-3 py-1.5 rounded-lg border transition-colors"
          style={{
            borderColor: "var(--color-border)",
            color: "var(--color-text-secondary)",
          }}
        >
          <Plus className="h-4 w-4" /> + Add Pool
        </button>
      </div>

      {/* Card list */}
      {loading ? (
        <CardSkeleton />
      ) : error ? (
        <CardError message={error} />
      ) : nodePools.length === 0 ? (
        <CardEmpty message="No node pools yet. Add one above." />
      ) : (
        <div className="flex flex-col gap-3">
          {nodePools.map((pool) => (
            <div
              key={pool.id}
              className="flex items-center gap-4 rounded-xl border px-5 py-4"
              style={{ backgroundColor: "var(--color-bg-card)", borderColor: "var(--color-border)" }}
            >
              {/* Drag handle */}
              <GripVertical
                className="h-5 w-5 flex-shrink-0 cursor-grab"
                style={{ color: "var(--color-text-muted)" }}
              />
              {/* Main content */}
              <div className="flex flex-col gap-1.5 flex-1 min-w-0">
                <span className="text-[15px] font-semibold" style={{ color: "var(--color-text-primary)" }}>
                  {pool.name}
                </span>
                {pool.endpoint_slug && (
                  <div className="flex flex-wrap gap-1.5">
                    <span
                      className="px-2 py-0.5 rounded-full text-[11px] font-medium"
                      style={{ backgroundColor: "var(--color-bg-accent)", color: "var(--color-text-secondary)" }}
                    >
                      {pool.endpoint_slug}
                    </span>
                  </div>
                )}
                <span className="text-[11px]" style={{ color: "var(--color-text-muted)" }}>
                  {pool.health_check_interval}s interval
                </span>
              </div>
              {/* Badge */}
              <span
                className="px-2.5 py-1 rounded-md text-[11px] font-medium flex-shrink-0"
                style={{ backgroundColor: "var(--color-primary-bg)", color: "var(--color-primary-light)" }}
              >
                node-pool
              </span>
              {/* Actions */}
              <div className="flex items-center gap-1 flex-shrink-0">
                <button
                  onClick={() => handleDelete(pool.id)}
                  disabled={deleting === pool.id}
                  className="px-3 py-1.5 text-[13px] rounded-lg border transition-colors"
                  style={{
                    borderColor: "var(--color-danger-bg)",
                    backgroundColor: "var(--color-danger-bg)",
                    color: "var(--color-danger)",
                  }}
                >
                  {deleting === pool.id ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : "Delete"}
                </button>
              </div>
            </div>
          ))}
        </div>
      )}

      {/* Inline create form */}
      {showForm && (
        <form
          onSubmit={handleSubmit}
          className="rounded-xl border p-4 flex flex-col gap-3"
          style={{ borderColor: "var(--color-border)", backgroundColor: "var(--color-bg-card)" }}
        >
          <div className="flex flex-col gap-1.5">
            <label className="text-[11px] font-medium uppercase tracking-wide" style={{ color: "var(--color-text-muted)" }}>Name *</label>
            <input
              value={formName}
              onChange={(e) => setFormName(e.target.value)}
              placeholder="e.g. my-pool"
              className="rounded-lg px-3 py-2 text-[13px] outline-none"
              style={inputStyle}
            />
          </div>
          <div className="flex flex-col gap-1.5">
            <label className="text-[11px] font-medium uppercase tracking-wide" style={{ color: "var(--color-text-muted)" }}>Endpoint</label>
            <select
              value={formEndpointId}
              onChange={(e) => setFormEndpointId(e.target.value)}
              className="rounded-lg px-3 py-2 text-[13px] outline-none"
              style={selectStyle}
            >
              <option value="">-- None --</option>
              {endpoints.map((ep) => (
                <option key={ep.id} value={ep.id}>{ep.name} ({ep.slug})</option>
              ))}
            </select>
          </div>
          <div className="flex gap-3">
            <div className="flex flex-col gap-1.5 flex-1">
              <label className="text-[11px] font-medium uppercase tracking-wide" style={{ color: "var(--color-text-muted)" }}>Health Check URL</label>
              <input
                value={formHealthCheckUrl}
                onChange={(e) => setFormHealthCheckUrl(e.target.value)}
                className="rounded-lg px-3 py-2 text-[13px] outline-none"
                style={inputStyle}
              />
            </div>
            <div className="flex flex-col gap-1.5" style={{ width: "100px" }}>
              <label className="text-[11px] font-medium uppercase tracking-wide" style={{ color: "var(--color-text-muted)" }}>Interval (s)</label>
              <input
                type="number"
                value={formHealthCheckInterval}
                onChange={(e) => setFormHealthCheckInterval(Number(e.target.value))}
                className="rounded-lg px-3 py-2 text-[13px] outline-none"
                style={inputStyle}
              />
            </div>
          </div>
          {formError && (
            <p className="text-xs" style={{ color: "var(--color-danger)" }}>{formError}</p>
          )}
          <div className="flex gap-2 justify-end">
            <Button type="button" variant="outline" size="sm" onClick={resetForm}>Cancel</Button>
            <Button type="submit" size="sm" disabled={submitting}>
              {submitting ? <Loader2 className="h-3.5 w-3.5 animate-spin mr-1" /> : <Plus className="h-3.5 w-3.5 mr-1" />}
              Add
            </Button>
          </div>
        </form>
      )}
    </div>
  );
}

// ── Rule Sets Tab ─────────────────────────────────────────────────────────────

interface RuleSetsTabProps {
  profileId: number;
  ruleSets: ProfileRuleSet[];
  loading: boolean;
  error: string | null;
  token?: string;
  endpoints: Endpoint[];
  onChanged: () => void;
}

function ruleSetBadgeStyle(target: string): { bg: string; color: string } {
  if (target === "REJECT") return { bg: "var(--color-danger-bg)", color: "var(--color-danger)" };
  if (target === "DIRECT") return { bg: "var(--color-success-bg)", color: "var(--color-success)" };
  return { bg: "var(--color-primary-bg)", color: "var(--color-primary-light)" };
}

function RuleSetsTab({ profileId, ruleSets, loading, error, token, endpoints, onChanged }: RuleSetsTabProps) {
  const [showForm, setShowForm] = useState(false);
  const [formName, setFormName] = useState("");
  const [formSourceType, setFormSourceType] = useState<"endpoint" | "url">("endpoint");
  const [formEndpointId, setFormEndpointId] = useState<string>("");
  const [formUrl, setFormUrl] = useState("");
  const [formBehavior, setFormBehavior] = useState("domain");
  const [formInterval, setFormInterval] = useState(86400);
  const [submitting, setSubmitting] = useState(false);
  const [formError, setFormError] = useState<string | null>(null);
  const [deleting, setDeleting] = useState<number | null>(null);

  function resetForm() {
    setFormName("");
    setFormSourceType("endpoint");
    setFormEndpointId("");
    setFormUrl("");
    setFormBehavior("domain");
    setFormInterval(86400);
    setFormError(null);
    setShowForm(false);
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!formName.trim()) { setFormError("Name is required"); return; }
    setSubmitting(true);
    setFormError(null);
    try {
      const selectedEndpoint = formSourceType === "endpoint" && formEndpointId
        ? endpoints.find((ep) => ep.id === Number(formEndpointId))
        : undefined;
      await createProfileRuleSet(profileId, {
        name: formName.trim(),
        endpoint_slug: selectedEndpoint?.slug,
        url: formSourceType === "url" ? formUrl : undefined,
        metadata: { behavior: formBehavior },
      }, { token });
      resetForm();
      onChanged();
    } catch (err: unknown) {
      setFormError(err instanceof Error ? err.message : "Failed to create rule set");
    } finally {
      setSubmitting(false);
    }
  }

  async function handleDelete(id: number) {
    setDeleting(id);
    try {
      await deleteProfileRuleSet(profileId, id, { token });
      onChanged();
    } catch {
      // silently ignore
    } finally {
      setDeleting(null);
    }
  }

  return (
    <div className="flex flex-col gap-4">
      {/* Tab content header */}
      <div className="flex items-center justify-between">
        <span />
        <button
          onClick={() => setShowForm(true)}
          className="flex items-center gap-1.5 text-[13px] px-3 py-1.5 rounded-lg border transition-colors"
          style={{
            borderColor: "var(--color-border)",
            color: "var(--color-text-secondary)",
          }}
        >
          <Plus className="h-4 w-4" /> + Add Rule Set
        </button>
      </div>

      {/* Card list */}
      {loading ? (
        <CardSkeleton />
      ) : error ? (
        <CardError message={error} />
      ) : ruleSets.length === 0 ? (
        <CardEmpty message="No rule sets yet. Add one above." />
      ) : (
        <div className="flex flex-col gap-3">
          {ruleSets.map((ruleSet) => {
            const targetStr = (ruleSet.metadata?.target as string) || "";
            const badgeStyle = ruleSetBadgeStyle(targetStr);
            const source = ruleSet.endpoint_slug || ruleSet.url || "—";
            const behavior = (ruleSet.metadata?.behavior as string) || "classical";
            return (
              <div
                key={ruleSet.id}
                className="flex items-center gap-4 rounded-xl border px-5 py-4"
                style={{ backgroundColor: "var(--color-bg-card)", borderColor: "var(--color-border)" }}
              >
                {/* Drag handle */}
                <GripVertical
                  className="h-5 w-5 flex-shrink-0 cursor-grab"
                  style={{ color: "var(--color-text-muted)" }}
                />
                {/* Main content */}
                <div className="flex flex-col gap-1.5 flex-1 min-w-0">
                  <span className="text-[15px] font-semibold" style={{ color: "var(--color-text-primary)" }}>
                    {ruleSet.name}
                  </span>
                  <span className="text-[11px]" style={{ color: "var(--color-text-muted)" }}>
                    Source: <span style={{ color: "var(--color-text-secondary)" }}>{source}</span>
                  </span>
                  <span className="text-[11px]" style={{ color: "var(--color-text-muted)" }}>
                    {behavior} · {ruleSet.interval}s
                  </span>
                </div>
                {/* Strategy badge */}
                {targetStr && (
                  <span
                    className="px-2.5 py-1 rounded-md text-[11px] font-medium flex-shrink-0 uppercase"
                    style={{ backgroundColor: badgeStyle.bg, color: badgeStyle.color }}
                  >
                    {targetStr}
                  </span>
                )}
                {/* Actions */}
                <div className="flex items-center gap-1 flex-shrink-0">
                  <button
                    onClick={() => handleDelete(ruleSet.id)}
                    disabled={deleting === ruleSet.id}
                    className="px-3 py-1.5 text-[13px] rounded-lg border transition-colors"
                    style={{
                      borderColor: "var(--color-danger-bg)",
                      backgroundColor: "var(--color-danger-bg)",
                      color: "var(--color-danger)",
                    }}
                  >
                    {deleting === ruleSet.id ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : "Delete"}
                  </button>
                </div>
              </div>
            );
          })}
        </div>
      )}

      {/* Inline create form */}
      {showForm && (
        <form
          onSubmit={handleSubmit}
          className="rounded-xl border p-4 flex flex-col gap-3"
          style={{ borderColor: "var(--color-border)", backgroundColor: "var(--color-bg-card)" }}
        >
          <div className="flex flex-col gap-1.5">
            <label className="text-[11px] font-medium uppercase tracking-wide" style={{ color: "var(--color-text-muted)" }}>Name *</label>
            <input
              value={formName}
              onChange={(e) => setFormName(e.target.value)}
              placeholder="e.g. lb_foreign"
              className="rounded-lg px-3 py-2 text-[13px] outline-none"
              style={inputStyle}
            />
          </div>

          <div className="flex flex-col gap-1.5">
            <label className="text-[11px] font-medium uppercase tracking-wide" style={{ color: "var(--color-text-muted)" }}>Source Type</label>
            <div className="flex gap-4">
              <label className="flex items-center gap-1.5 text-[13px] cursor-pointer" style={{ color: "var(--color-text-secondary)" }}>
                <input
                  type="radio"
                  name="sourceType"
                  checked={formSourceType === "endpoint"}
                  onChange={() => setFormSourceType("endpoint")}
                />
                Endpoint
              </label>
              <label className="flex items-center gap-1.5 text-[13px] cursor-pointer" style={{ color: "var(--color-text-secondary)" }}>
                <input
                  type="radio"
                  name="sourceType"
                  checked={formSourceType === "url"}
                  onChange={() => setFormSourceType("url")}
                />
                External URL
              </label>
            </div>
          </div>

          {formSourceType === "endpoint" ? (
            <div className="flex flex-col gap-1.5">
              <label className="text-[11px] font-medium uppercase tracking-wide" style={{ color: "var(--color-text-muted)" }}>Endpoint</label>
              <select
                value={formEndpointId}
                onChange={(e) => setFormEndpointId(e.target.value)}
                className="rounded-lg px-3 py-2 text-[13px] outline-none"
                style={selectStyle}
              >
                <option value="">-- None --</option>
                {endpoints.map((ep) => (
                  <option key={ep.id} value={ep.id}>{ep.name} ({ep.slug})</option>
                ))}
              </select>
            </div>
          ) : (
            <div className="flex flex-col gap-1.5">
              <label className="text-[11px] font-medium uppercase tracking-wide" style={{ color: "var(--color-text-muted)" }}>URL</label>
              <input
                value={formUrl}
                onChange={(e) => setFormUrl(e.target.value)}
                placeholder="https://..."
                className="rounded-lg px-3 py-2 text-[13px] outline-none"
                style={inputStyle}
              />
            </div>
          )}

          <div className="flex gap-3">
            <div className="flex flex-col gap-1.5 flex-1">
              <label className="text-[11px] font-medium uppercase tracking-wide" style={{ color: "var(--color-text-muted)" }}>Behavior</label>
              <select
                value={formBehavior}
                onChange={(e) => setFormBehavior(e.target.value)}
                className="rounded-lg px-3 py-2 text-[13px] outline-none"
                style={selectStyle}
              >
                <option value="domain">domain</option>
                <option value="ipcidr">ipcidr</option>
                <option value="classical">classical</option>
              </select>
            </div>
            <div className="flex flex-col gap-1.5" style={{ width: "120px" }}>
              <label className="text-[11px] font-medium uppercase tracking-wide" style={{ color: "var(--color-text-muted)" }}>Interval (s)</label>
              <input
                type="number"
                value={formInterval}
                onChange={(e) => setFormInterval(Number(e.target.value))}
                className="rounded-lg px-3 py-2 text-[13px] outline-none"
                style={inputStyle}
              />
            </div>
          </div>

          {formError && (
            <p className="text-xs" style={{ color: "var(--color-danger)" }}>{formError}</p>
          )}
          <div className="flex gap-2 justify-end">
            <Button type="button" variant="outline" size="sm" onClick={resetForm}>Cancel</Button>
            <Button type="submit" size="sm" disabled={submitting}>
              {submitting ? <Loader2 className="h-3.5 w-3.5 animate-spin mr-1" /> : <Plus className="h-3.5 w-3.5 mr-1" />}
              Add
            </Button>
          </div>
        </form>
      )}
    </div>
  );
}

// ── Strategies Tab ────────────────────────────────────────────────────────────

interface StrategiesTabProps {
  profileId: number;
  strategies: ProfileStrategy[];
  loading: boolean;
  error: string | null;
  token?: string;
  onChanged: () => void;
}

function StrategiesTab({ profileId, strategies, loading, error, token, onChanged }: StrategiesTabProps) {
  const [showForm, setShowForm] = useState(false);
  const [formName, setFormName] = useState("");
  const [formStrategy, setFormStrategy] = useState("select");
  const [formPools, setFormPools] = useState("");
  const [formProxies, setFormProxies] = useState("");
  const [showAdvanced, setShowAdvanced] = useState(false);
  const [formFilter, setFormFilter] = useState("");
  const [formIncludeAll, setFormIncludeAll] = useState(false);
  const [formTolerance, setFormTolerance] = useState("");
  const [formConfigUrl, setFormConfigUrl] = useState("");
  const [formConfigInterval, setFormConfigInterval] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [formError, setFormError] = useState<string | null>(null);
  const [deleting, setDeleting] = useState<number | null>(null);

  function resetForm() {
    setFormName("");
    setFormStrategy("select");
    setFormPools("");
    setFormProxies("");
    setShowAdvanced(false);
    setFormFilter("");
    setFormIncludeAll(false);
    setFormTolerance("");
    setFormConfigUrl("");
    setFormConfigInterval("");
    setFormError(null);
    setShowForm(false);
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!formName.trim()) { setFormError("Name is required"); return; }
    setSubmitting(true);
    setFormError(null);
    try {
      const pools = formPools.split(",").map((s) => s.trim()).filter(Boolean);
      const proxies = formProxies.split(",").map((s) => s.trim()).filter(Boolean);
      const config: Record<string, unknown> = {};
      if (formFilter.trim()) config.filter = formFilter.trim();
      if (formIncludeAll) config["include-all"] = true;
      if (formTolerance.trim()) config.tolerance = Number(formTolerance);
      if (formConfigUrl.trim()) config.url = formConfigUrl.trim();
      if (formConfigInterval.trim()) config.interval = Number(formConfigInterval);

      await createProfileStrategy(profileId, {
        name: formName.trim(),
        strategy: formStrategy,
        pools: pools.length > 0 ? pools : undefined,
        proxies: proxies.length > 0 ? proxies : undefined,
        config: Object.keys(config).length > 0 ? config : undefined,
        position: strategies.length + 1,
      }, { token });
      resetForm();
      onChanged();
    } catch (err: unknown) {
      setFormError(err instanceof Error ? err.message : "Failed to create strategy");
    } finally {
      setSubmitting(false);
    }
  }

  async function handleDelete(id: number) {
    setDeleting(id);
    try {
      await deleteProfileStrategy(profileId, id, { token });
      onChanged();
    } catch {
      // silently ignore
    } finally {
      setDeleting(null);
    }
  }

  return (
    <div className="flex flex-col gap-4">
      {/* Tab content header */}
      <div className="flex items-center justify-between">
        <span />
        <button
          onClick={() => setShowForm(true)}
          className="flex items-center gap-1.5 text-[13px] px-3 py-1.5 rounded-lg border transition-colors"
          style={{
            borderColor: "var(--color-border)",
            color: "var(--color-text-secondary)",
          }}
        >
          <Plus className="h-4 w-4" /> + Add Strategy
        </button>
      </div>

      {/* Card list */}
      {loading ? (
        <CardSkeleton />
      ) : error ? (
        <CardError message={error} />
      ) : strategies.length === 0 ? (
        <CardEmpty message="No strategies yet. Add one above." />
      ) : (
        <div className="flex flex-col gap-3">
          {strategies.map((strategy) => (
            <div
              key={strategy.id}
              className="flex items-center gap-4 rounded-xl border px-5 py-4"
              style={{ backgroundColor: "var(--color-bg-card)", borderColor: "var(--color-border)" }}
            >
              {/* Drag handle */}
              <GripVertical
                className="h-5 w-5 flex-shrink-0 cursor-grab"
                style={{ color: "var(--color-text-muted)" }}
              />
              {/* Main content */}
              <div className="flex flex-col gap-1.5 flex-1 min-w-0">
                <span className="text-[15px] font-semibold" style={{ color: "var(--color-text-primary)" }}>
                  {strategy.name}
                </span>
                {(strategy.pools ?? []).length > 0 && (
                  <div className="flex flex-wrap gap-1.5">
                    {(strategy.pools ?? []).map((pool) => (
                      <span
                        key={pool}
                        className="px-2 py-0.5 rounded-full text-[11px] font-medium"
                        style={{ backgroundColor: "var(--color-bg-accent)", color: "var(--color-text-secondary)" }}
                      >
                        {pool}
                      </span>
                    ))}
                  </div>
                )}
                {(strategy.proxies ?? []).length > 0 && (
                  <span className="text-[11px]" style={{ color: "var(--color-text-muted)" }}>
                    proxies: {(strategy.proxies ?? []).join(", ")}
                  </span>
                )}
              </div>
              {/* Strategy type badge */}
              <span
                className="px-2.5 py-1 rounded-md text-[11px] font-medium flex-shrink-0"
                style={{ backgroundColor: "var(--color-bg-accent)", color: "var(--color-text-secondary)" }}
              >
                {strategy.strategy}
              </span>
              {/* Actions */}
              <div className="flex items-center gap-1 flex-shrink-0">
                <button
                  onClick={() => handleDelete(strategy.id)}
                  disabled={deleting === strategy.id}
                  className="px-3 py-1.5 text-[13px] rounded-lg border transition-colors"
                  style={{
                    borderColor: "var(--color-danger-bg)",
                    backgroundColor: "var(--color-danger-bg)",
                    color: "var(--color-danger)",
                  }}
                >
                  {deleting === strategy.id ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : "Delete"}
                </button>
              </div>
            </div>
          ))}
        </div>
      )}

      {/* Inline create form */}
      {showForm && (
        <form
          onSubmit={handleSubmit}
          className="rounded-xl border p-4 flex flex-col gap-3"
          style={{ borderColor: "var(--color-border)", backgroundColor: "var(--color-bg-card)" }}
        >
          <div className="flex gap-3">
            <div className="flex flex-col gap-1.5 flex-1">
              <label className="text-[11px] font-medium uppercase tracking-wide" style={{ color: "var(--color-text-muted)" }}>Name *</label>
              <input
                value={formName}
                onChange={(e) => setFormName(e.target.value)}
                placeholder="e.g. Auto Select"
                className="rounded-lg px-3 py-2 text-[13px] outline-none"
                style={inputStyle}
              />
            </div>
            <div className="flex flex-col gap-1.5" style={{ width: "150px" }}>
              <label className="text-[11px] font-medium uppercase tracking-wide" style={{ color: "var(--color-text-muted)" }}>Strategy Type</label>
              <select
                value={formStrategy}
                onChange={(e) => setFormStrategy(e.target.value)}
                className="rounded-lg px-3 py-2 text-[13px] outline-none"
                style={selectStyle}
              >
                <option value="select">select</option>
                <option value="auto">auto</option>
                <option value="fallback">fallback</option>
                <option value="load_balance">load_balance</option>
              </select>
            </div>
          </div>

          <div className="flex flex-col gap-1.5">
            <label className="text-[11px] font-medium uppercase tracking-wide" style={{ color: "var(--color-text-muted)" }}>Pools (comma-separated)</label>
            <input
              value={formPools}
              onChange={(e) => setFormPools(e.target.value)}
              placeholder="e.g. my-pool, another-pool"
              className="rounded-lg px-3 py-2 text-[13px] outline-none"
              style={inputStyle}
            />
          </div>

          <div className="flex flex-col gap-1.5">
            <label className="text-[11px] font-medium uppercase tracking-wide" style={{ color: "var(--color-text-muted)" }}>Proxies (comma-separated)</label>
            <input
              value={formProxies}
              onChange={(e) => setFormProxies(e.target.value)}
              placeholder="e.g. DIRECT, REJECT"
              className="rounded-lg px-3 py-2 text-[13px] outline-none"
              style={inputStyle}
            />
          </div>

          {/* Advanced config */}
          <button
            type="button"
            onClick={() => setShowAdvanced(!showAdvanced)}
            className="flex items-center gap-1 text-[11px] font-medium w-fit"
            style={{ color: "var(--color-text-muted)" }}
          >
            {showAdvanced ? <ChevronDown className="h-3 w-3" /> : <ChevronRight className="h-3 w-3" />}
            Advanced Config
          </button>

          {showAdvanced && (
            <div className="flex flex-col gap-3 pl-2" style={{ borderLeft: "2px solid var(--color-border)" }}>
              <div className="flex flex-col gap-1.5">
                <label className="text-[11px] font-medium uppercase tracking-wide" style={{ color: "var(--color-text-muted)" }}>Filter (regex)</label>
                <input
                  value={formFilter}
                  onChange={(e) => setFormFilter(e.target.value)}
                  placeholder="e.g. (?i)hong|hk"
                  className="rounded-lg px-3 py-2 text-[13px] font-mono outline-none"
                  style={inputStyle}
                />
              </div>
              <label className="flex items-center gap-2 text-[13px] cursor-pointer" style={{ color: "var(--color-text-secondary)" }}>
                <input
                  type="checkbox"
                  checked={formIncludeAll}
                  onChange={(e) => setFormIncludeAll(e.target.checked)}
                />
                include-all
              </label>
              <div className="flex gap-3">
                <div className="flex flex-col gap-1.5 flex-1">
                  <label className="text-[11px] font-medium uppercase tracking-wide" style={{ color: "var(--color-text-muted)" }}>URL (health check)</label>
                  <input
                    value={formConfigUrl}
                    onChange={(e) => setFormConfigUrl(e.target.value)}
                    placeholder="http://www.gstatic.com/generate_204"
                    className="rounded-lg px-3 py-2 text-[13px] outline-none"
                    style={inputStyle}
                  />
                </div>
                <div className="flex flex-col gap-1.5" style={{ width: "80px" }}>
                  <label className="text-[11px] font-medium uppercase tracking-wide" style={{ color: "var(--color-text-muted)" }}>Interval</label>
                  <input
                    type="number"
                    value={formConfigInterval}
                    onChange={(e) => setFormConfigInterval(e.target.value)}
                    placeholder="300"
                    className="rounded-lg px-3 py-2 text-[13px] outline-none"
                    style={inputStyle}
                  />
                </div>
                <div className="flex flex-col gap-1.5" style={{ width: "80px" }}>
                  <label className="text-[11px] font-medium uppercase tracking-wide" style={{ color: "var(--color-text-muted)" }}>Tolerance</label>
                  <input
                    type="number"
                    value={formTolerance}
                    onChange={(e) => setFormTolerance(e.target.value)}
                    placeholder="150"
                    className="rounded-lg px-3 py-2 text-[13px] outline-none"
                    style={inputStyle}
                  />
                </div>
              </div>
            </div>
          )}

          {formError && (
            <p className="text-xs" style={{ color: "var(--color-danger)" }}>{formError}</p>
          )}
          <div className="flex gap-2 justify-end">
            <Button type="button" variant="outline" size="sm" onClick={resetForm}>Cancel</Button>
            <Button type="submit" size="sm" disabled={submitting}>
              {submitting ? <Loader2 className="h-3.5 w-3.5 animate-spin mr-1" /> : <Plus className="h-3.5 w-3.5 mr-1" />}
              Add
            </Button>
          </div>
        </form>
      )}
    </div>
  );
}

// ── Routing Rules Tab ─────────────────────────────────────────────────────────

const MATCH_TYPES = [
  "RULE-SET",
  "DOMAIN",
  "DOMAIN-SUFFIX",
  "DOMAIN-KEYWORD",
  "IP-CIDR",
  "GEOSITE",
  "GEOIP",
  "MATCH",
  "SRC-IP-CIDR",
  "SRC-PORT",
  "DST-PORT",
  "PROCESS-NAME",
];

const NO_RESOLVE_TYPES = new Set(["IP-CIDR", "GEOIP", "SRC-IP-CIDR"]);
const NO_VALUE_TYPES = new Set(["MATCH"]);

interface RoutingRulesTabProps {
  profileId: number;
  routingRules: ProfileRoutingRule[];
  loading: boolean;
  error: string | null;
  token?: string;
  strategies: ProfileStrategy[];
  onChanged: () => void;
}

function routingTargetColor(target: string): string {
  if (target === "DIRECT") return "var(--color-success)";
  if (target === "REJECT") return "var(--color-danger)";
  return "var(--color-text-secondary)";
}

function RoutingRulesTab({ profileId, routingRules, loading, error, token, strategies, onChanged }: RoutingRulesTabProps) {
  const [showForm, setShowForm] = useState(false);
  const [formMatch, setFormMatch] = useState("RULE-SET");
  const [formValue, setFormValue] = useState("");
  const [formTarget, setFormTarget] = useState("");
  const [formNoResolve, setFormNoResolve] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [formError, setFormError] = useState<string | null>(null);
  const [deleting, setDeleting] = useState<number | null>(null);

  function resetForm() {
    setFormMatch("RULE-SET");
    setFormValue("");
    setFormTarget("");
    setFormNoResolve(false);
    setFormError(null);
    setShowForm(false);
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!formTarget.trim()) { setFormError("Target is required"); return; }
    if (!NO_VALUE_TYPES.has(formMatch) && !formValue.trim()) { setFormError("Value is required"); return; }
    setSubmitting(true);
    setFormError(null);
    try {
      await createProfileRoutingRule(profileId, {
        match: formMatch.toLowerCase(),
        value: NO_VALUE_TYPES.has(formMatch) ? undefined : formValue.trim(),
        target: formTarget.trim(),
        no_resolve: NO_RESOLVE_TYPES.has(formMatch) ? formNoResolve : undefined,
        position: routingRules.length + 1,
      }, { token });
      resetForm();
      onChanged();
    } catch (err: unknown) {
      setFormError(err instanceof Error ? err.message : "Failed to create routing rule");
    } finally {
      setSubmitting(false);
    }
  }

  async function handleDelete(id: number) {
    setDeleting(id);
    try {
      await deleteProfileRoutingRule(profileId, id, { token });
      onChanged();
    } catch {
      // silently ignore
    } finally {
      setDeleting(null);
    }
  }

  const sortedRules = [...routingRules].sort((a, b) => a.position - b.position);
  const strategyNames = strategies.map((s) => s.name);

  return (
    <div className="flex flex-col gap-4">
      {/* Tab content header */}
      <div className="flex items-center justify-between">
        <span />
        <button
          onClick={() => setShowForm(true)}
          className="flex items-center gap-1.5 text-[13px] px-3 py-1.5 rounded-lg border transition-colors"
          style={{
            borderColor: "var(--color-border)",
            color: "var(--color-text-secondary)",
          }}
        >
          <Plus className="h-4 w-4" /> + Add Routing Rule
        </button>
      </div>

      {/* Card list */}
      {loading ? (
        <CardSkeleton />
      ) : error ? (
        <CardError message={error} />
      ) : sortedRules.length === 0 ? (
        <CardEmpty message="No routing rules yet. Add one above." />
      ) : (
        <div className="flex flex-col gap-3">
          {sortedRules.map((rule) => (
            <div
              key={rule.id}
              className="flex items-center gap-4 rounded-xl border px-5 py-4"
              style={{ backgroundColor: "var(--color-bg-card)", borderColor: "var(--color-border)" }}
            >
              {/* Drag handle */}
              <GripVertical
                className="h-5 w-5 flex-shrink-0 cursor-grab"
                style={{ color: "var(--color-text-muted)" }}
              />
              {/* Rule position */}
              <span
                className="text-[11px] font-mono flex-shrink-0 w-6 text-center"
                style={{ color: "var(--color-text-muted)" }}
              >
                {rule.position}
              </span>
              {/* Main content */}
              <div className="flex items-center gap-3 flex-1 min-w-0 flex-wrap">
                {/* Type badge */}
                <span
                  className="px-2 py-0.5 rounded-md text-[11px] font-medium font-mono flex-shrink-0"
                  style={{ backgroundColor: "var(--color-primary-bg)", color: "var(--color-primary-light)" }}
                >
                  {rule.type}
                </span>
                {/* Payload */}
                {rule.payload && (
                  <span
                    className="text-[13px] font-mono truncate"
                    style={{ color: "var(--color-text-secondary)" }}
                  >
                    {rule.payload}
                  </span>
                )}
                {/* Arrow */}
                <span style={{ color: "var(--color-text-muted)" }}>→</span>
                {/* Target */}
                <span
                  className="text-[13px] font-medium flex-shrink-0"
                  style={{ color: routingTargetColor(rule.target) }}
                >
                  {rule.target}
                </span>
                {rule.no_resolve && (
                  <span
                    className="text-[10px] px-1.5 py-0.5 rounded"
                    style={{ backgroundColor: "var(--color-bg-accent)", color: "var(--color-text-muted)" }}
                  >
                    no-resolve
                  </span>
                )}
              </div>
              {/* Actions */}
              <div className="flex items-center gap-1 flex-shrink-0">
                <button
                  onClick={() => handleDelete(rule.id)}
                  disabled={deleting === rule.id}
                  className="px-3 py-1.5 text-[13px] rounded-lg border transition-colors"
                  style={{
                    borderColor: "var(--color-danger-bg)",
                    backgroundColor: "var(--color-danger-bg)",
                    color: "var(--color-danger)",
                  }}
                >
                  {deleting === rule.id ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : "Delete"}
                </button>
              </div>
            </div>
          ))}
        </div>
      )}

      {/* Inline create form */}
      {showForm && (
        <form
          onSubmit={handleSubmit}
          className="rounded-xl border p-4 flex flex-col gap-3"
          style={{ borderColor: "var(--color-border)", backgroundColor: "var(--color-bg-card)" }}
        >
          <div className="flex gap-3">
            <div className="flex flex-col gap-1.5" style={{ width: "180px" }}>
              <label className="text-[11px] font-medium uppercase tracking-wide" style={{ color: "var(--color-text-muted)" }}>Match Type</label>
              <select
                value={formMatch}
                onChange={(e) => { setFormMatch(e.target.value); setFormNoResolve(false); }}
                className="rounded-lg px-3 py-2 text-[13px] outline-none"
                style={selectStyle}
              >
                {MATCH_TYPES.map((t) => (
                  <option key={t} value={t}>{t}</option>
                ))}
              </select>
            </div>
            {!NO_VALUE_TYPES.has(formMatch) && (
              <div className="flex flex-col gap-1.5 flex-1">
                <label className="text-[11px] font-medium uppercase tracking-wide" style={{ color: "var(--color-text-muted)" }}>Value</label>
                <input
                  value={formValue}
                  onChange={(e) => setFormValue(e.target.value)}
                  placeholder="e.g. lb_foreign, google.com, 192.168.0.0/16"
                  className="rounded-lg px-3 py-2 text-[13px] font-mono outline-none"
                  style={inputStyle}
                />
              </div>
            )}
          </div>

          <div className="flex gap-3 items-end">
            <div className="flex flex-col gap-1.5 flex-1">
              <label className="text-[11px] font-medium uppercase tracking-wide" style={{ color: "var(--color-text-muted)" }}>Target *</label>
              {strategyNames.length > 0 ? (
                <div className="flex gap-2">
                  <select
                    value={formTarget}
                    onChange={(e) => setFormTarget(e.target.value)}
                    className="rounded-lg px-3 py-2 text-[13px] outline-none flex-1"
                    style={selectStyle}
                  >
                    <option value="">-- Select or type below --</option>
                    <option value="DIRECT">DIRECT</option>
                    <option value="REJECT">REJECT</option>
                    {strategyNames.map((n) => (
                      <option key={n} value={n}>{n}</option>
                    ))}
                  </select>
                  <input
                    value={formTarget}
                    onChange={(e) => setFormTarget(e.target.value)}
                    placeholder="or type custom"
                    className="rounded-lg px-3 py-2 text-[13px] outline-none"
                    style={{ ...inputStyle, width: "140px" }}
                  />
                </div>
              ) : (
                <input
                  value={formTarget}
                  onChange={(e) => setFormTarget(e.target.value)}
                  placeholder="e.g. DIRECT, REJECT, or strategy name"
                  className="rounded-lg px-3 py-2 text-[13px] outline-none"
                  style={inputStyle}
                />
              )}
            </div>
          </div>

          {NO_RESOLVE_TYPES.has(formMatch) && (
            <label className="flex items-center gap-2 text-[13px] cursor-pointer" style={{ color: "var(--color-text-secondary)" }}>
              <input
                type="checkbox"
                checked={formNoResolve}
                onChange={(e) => setFormNoResolve(e.target.checked)}
              />
              no-resolve
            </label>
          )}

          {formError && (
            <p className="text-xs" style={{ color: "var(--color-danger)" }}>{formError}</p>
          )}
          <div className="flex gap-2 justify-end">
            <Button type="button" variant="outline" size="sm" onClick={resetForm}>Cancel</Button>
            <Button type="submit" size="sm" disabled={submitting}>
              {submitting ? <Loader2 className="h-3.5 w-3.5 animate-spin mr-1" /> : <Plus className="h-3.5 w-3.5 mr-1" />}
              Add Rule
            </Button>
          </div>
        </form>
      )}
    </div>
  );
}

// ── ProfileEditorPage ─────────────────────────────────────────────────────────

export function ProfileEditorPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const { token } = useAuth();
  const profileId = id ? Number(id) : NaN;

  const [activeTab, setActiveTab] = useState<TabId>("node-pools");

  // Mobile panel switcher (< md)
  const [mobilePanel, setMobilePanel] = useState<"edit" | "preview">("edit");

  // Profile
  const [profile, setProfile] = useState<Profile | null>(null);
  const [profileLoading, setProfileLoading] = useState(true);
  const [profileError, setProfileError] = useState<string | null>(null);

  // Profile meta (name + slug editing)
  const [showMeta, setShowMeta] = useState(false);
  const [metaName, setMetaName] = useState("");
  const [metaSlug, setMetaSlug] = useState("");
  const [metaError, setMetaError] = useState<string | null>(null);
  const [metaSaving, setMetaSaving] = useState(false);

  // Endpoints (shared across tabs)
  const [endpoints, setEndpoints] = useState<Endpoint[]>([]);

  // Sub-resources
  const [nodePools, setNodePools] = useState<ProfileNodePool[]>([]);
  const [nodePoolsLoading, setNodePoolsLoading] = useState(false);
  const [nodePoolsError, setNodePoolsError] = useState<string | null>(null);

  const [ruleSets, setRuleSets] = useState<ProfileRuleSet[]>([]);
  const [ruleSetsLoading, setRuleSetsLoading] = useState(false);
  const [ruleSetsError, setRuleSetsError] = useState<string | null>(null);

  const [strategies, setStrategies] = useState<ProfileStrategy[]>([]);
  const [strategiesLoading, setStrategiesLoading] = useState(false);
  const [strategiesError, setStrategiesError] = useState<string | null>(null);

  const [routingRules, setRoutingRules] = useState<ProfileRoutingRule[]>([]);
  const [routingRulesLoading, setRoutingRulesLoading] = useState(false);
  const [routingRulesError, setRoutingRulesError] = useState<string | null>(null);

  // YAML preview
  const [yaml, setYaml] = useState<string>("");
  const [yamlLoading, setYamlLoading] = useState(false);

  // ── Load profile ──────────────────────────────────────────────────────────

  useEffect(() => {
    if (isNaN(profileId)) {
      setProfileError("Invalid profile ID.");
      setProfileLoading(false);
      return;
    }
    let cancelled = false;
    setProfileLoading(true);
    setProfileError(null);
    getProfile(profileId, { token })
      .then((p) => {
        if (!cancelled) {
          setProfile(p);
          setMetaName(p.name);
          setMetaSlug(p.slug);
          setProfileLoading(false);
        }
      })
      .catch((err: unknown) => {
        if (!cancelled) {
          const msg = err instanceof Error ? err.message : "Failed to load profile";
          setProfileError(msg);
          setProfileLoading(false);
        }
      });
    return () => { cancelled = true; };
  }, [profileId, token]);

  // ── Load endpoints once ─────────────────────────────────────────────────

  useEffect(() => {
    fetchEndpoints({ token }).then(setEndpoints).catch(() => {});
  }, [token]);

  // ── Load sub-resources when tab activates ────────────────────────────────

  const loadNodePools = useCallback(() => {
    if (isNaN(profileId)) return;
    setNodePoolsLoading(true);
    setNodePoolsError(null);
    fetchProfileNodePools(profileId, { token })
      .then((items) => { setNodePools(items); setNodePoolsLoading(false); })
      .catch((err: unknown) => { setNodePoolsError(err instanceof Error ? err.message : "Failed to load node pools"); setNodePoolsLoading(false); });
  }, [profileId, token]);

  const loadRuleSets = useCallback(() => {
    if (isNaN(profileId)) return;
    setRuleSetsLoading(true);
    setRuleSetsError(null);
    fetchProfileRuleSets(profileId, { token })
      .then((items) => { setRuleSets(items); setRuleSetsLoading(false); })
      .catch((err: unknown) => { setRuleSetsError(err instanceof Error ? err.message : "Failed to load rule sets"); setRuleSetsLoading(false); });
  }, [profileId, token]);

  const loadStrategies = useCallback(() => {
    if (isNaN(profileId)) return;
    setStrategiesLoading(true);
    setStrategiesError(null);
    fetchProfileStrategies(profileId, { token })
      .then((items) => { setStrategies(items); setStrategiesLoading(false); })
      .catch((err: unknown) => { setStrategiesError(err instanceof Error ? err.message : "Failed to load strategies"); setStrategiesLoading(false); });
  }, [profileId, token]);

  const loadRoutingRules = useCallback(() => {
    if (isNaN(profileId)) return;
    setRoutingRulesLoading(true);
    setRoutingRulesError(null);
    fetchProfileRoutingRules(profileId, { token })
      .then((items) => { setRoutingRules(items); setRoutingRulesLoading(false); })
      .catch((err: unknown) => { setRoutingRulesError(err instanceof Error ? err.message : "Failed to load routing rules"); setRoutingRulesLoading(false); });
  }, [profileId, token]);

  useEffect(() => {
    if (activeTab === "node-pools") loadNodePools();
  }, [activeTab, loadNodePools]);

  useEffect(() => {
    if (activeTab === "rule-sets") loadRuleSets();
  }, [activeTab, loadRuleSets]);

  useEffect(() => {
    if (activeTab === "strategies") loadStrategies();
  }, [activeTab, loadStrategies]);

  useEffect(() => {
    if (activeTab === "routing-rules") {
      loadRoutingRules();
      // Also load strategies for target dropdown
      loadStrategies();
    }
  }, [activeTab, loadRoutingRules, loadStrategies]);

  // ── YAML preview (on profile load and settings changes) ──────────────────

  const refreshYaml = useCallback(
    (slug: string) => {
      setYamlLoading(true);
      fetch(`/profile/${slug}`, {
        headers: token ? { Authorization: `Bearer ${token}` } : {},
      })
        .then((res) => (res.ok ? res.text() : Promise.reject(new Error(`HTTP ${res.status}`))))
        .then((text) => { setYaml(text); setYamlLoading(false); })
        .catch(() => { setYamlLoading(false); });
    },
    [token]
  );

  useEffect(() => {
    if (profile?.slug) refreshYaml(profile.slug);
  }, [profile?.slug, refreshYaml]);

  const handleSettingsSaved = useCallback(
    (updated: Profile) => {
      setProfile(updated);
      setMetaName(updated.name);
      setMetaSlug(updated.slug);
      if (updated.slug) refreshYaml(updated.slug);
    },
    [refreshYaml]
  );

  async function handleMetaSave() {
    if (!profile) return;
    const trimmedSlug = metaSlug.trim();
    if (!trimmedSlug) { setMetaError("Slug cannot be empty"); return; }
    if (!/^[a-z0-9][a-z0-9-]*$/.test(trimmedSlug)) { setMetaError("Lowercase alphanumeric + hyphens only"); return; }
    if (!metaName.trim()) { setMetaError("Name cannot be empty"); return; }
    setMetaSaving(true);
    setMetaError(null);
    try {
      const updated = await apiFetch<Profile>(`/api/profiles/${profile.id}`, {
        token, method: "PUT",
        body: { name: metaName.trim(), slug: trimmedSlug },
      });
      setProfile(updated);
      setMetaName(updated.name);
      setMetaSlug(updated.slug);
      setShowMeta(false);
      if (updated.slug) refreshYaml(updated.slug);
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : "Failed to save";
      if (msg.includes("slug already in use")) setMetaError("This slug is already taken");
      else setMetaError(msg);
    } finally {
      setMetaSaving(false);
    }
  }

  // Callbacks for sub-resource changes that refresh data + YAML
  const handleNodePoolsChanged = useCallback(() => {
    loadNodePools();
    if (profile?.slug) refreshYaml(profile.slug);
  }, [loadNodePools, profile?.slug, refreshYaml]);

  const handleRuleSetsChanged = useCallback(() => {
    loadRuleSets();
    if (profile?.slug) refreshYaml(profile.slug);
  }, [loadRuleSets, profile?.slug, refreshYaml]);

  const handleStrategiesChanged = useCallback(() => {
    loadStrategies();
    if (profile?.slug) refreshYaml(profile.slug);
  }, [loadStrategies, profile?.slug, refreshYaml]);

  const handleRoutingRulesChanged = useCallback(() => {
    loadRoutingRules();
    if (profile?.slug) refreshYaml(profile.slug);
  }, [loadRoutingRules, profile?.slug, refreshYaml]);

  // ── Loading / error states ────────────────────────────────────────────────

  if (profileLoading) {
    return (
      <div className="flex flex-col gap-4">
        <div className="h-7 w-48 rounded-lg animate-pulse" style={{ backgroundColor: "var(--color-bg-accent)" }} />
        <div className="h-4 w-32 rounded-lg animate-pulse" style={{ backgroundColor: "var(--color-bg-accent)" }} />
      </div>
    );
  }

  if (profileError || !profile) {
    return (
      <div className="flex flex-col gap-4">
        <button
          onClick={() => navigate("/profiles")}
          className="flex items-center gap-1 text-[13px] w-fit"
          style={{ color: "var(--color-text-muted)" }}
        >
          <ChevronLeft className="h-4 w-4" />
          Back to Profiles
        </button>
        <div
          className="rounded-xl border px-4 py-3 text-[13px]"
          style={{
            backgroundColor: "var(--color-danger-bg)",
            color: "var(--color-danger)",
            borderColor: "var(--color-danger-bg)",
          }}
        >
          {profileError ?? "Profile not found"}
        </div>
      </div>
    );
  }

  // ── Main render ───────────────────────────────────────────────────────────

  return (
    <div className="flex flex-col gap-5 h-full">
      {/* Page header */}
      <div className="flex items-center justify-between gap-4 flex-wrap">
        <div className="flex flex-col gap-1">
          {/* Breadcrumb */}
          <div className="flex items-center gap-1.5 text-[12px]" style={{ color: "var(--color-text-muted)" }}>
            <Link
              to="/profiles"
              className="hover:underline transition-colors"
              style={{ color: "var(--color-text-muted)" }}
            >
              Profiles
            </Link>
            <span>/</span>
            <span style={{ color: "var(--color-text-secondary)" }}>{profile.name}</span>
          </div>
          {/* Title + Profile Meta */}
          <div className="flex flex-col sm:flex-row sm:items-center gap-2 sm:gap-4">
            <h1 className="text-[26px] font-bold leading-tight" style={{ color: "var(--color-text-primary)" }}>
              {profile.name}
            </h1>
            <button
              onClick={() => setShowMeta((v) => !v)}
              className="text-[12px] px-2 py-1 rounded-md self-start transition-colors"
              style={{
                color: "var(--color-text-muted)",
                backgroundColor: "var(--color-bg-accent)",
                border: "1px solid var(--color-border)",
              }}
            >
              {showMeta ? "Hide" : "Edit"} profile info
            </button>
          </div>
          {/* Collapsible name + slug editor */}
          {showMeta && (
            <div className="flex flex-col gap-3 mt-2 p-4 rounded-xl border" style={{ backgroundColor: "var(--color-bg-card)", borderColor: "var(--color-border)" }}>
              <div className="flex flex-col gap-1.5">
                <label className="text-[12px] font-medium" style={{ color: "var(--color-text-secondary)" }}>Name</label>
                <input
                  value={metaName}
                  onChange={(e) => setMetaName(e.target.value)}
                  className="h-9 rounded-[var(--radius)] px-3 text-[13px] outline-none transition-colors"
                  style={{ backgroundColor: "var(--color-bg)", color: "var(--color-text-primary)", border: "1px solid var(--color-border)" }}
                />
              </div>
              <div className="flex flex-col gap-1.5">
                <label className="text-[12px] font-medium" style={{ color: "var(--color-text-secondary)" }}>Slug (profile URL)</label>
                <div className="flex items-center gap-2">
                  <input
                    value={metaSlug}
                    onChange={(e) => { setMetaSlug(e.target.value.toLowerCase().replace(/[^a-z0-9-]/g, "")); setMetaError(null); }}
                    placeholder="e.g. daily-driver"
                    className="flex-1 h-9 rounded-[var(--radius)] px-3 text-[13px] font-mono outline-none transition-colors"
                    style={{
                      backgroundColor: "var(--color-bg)",
                      color: "var(--color-text-primary)",
                      border: metaError ? "1px solid var(--color-danger)" : "1px solid var(--color-border)",
                    }}
                  />
                  <CopyButton text={`${window.location.origin}/profile/${metaSlug}`} />
                </div>
                {metaError && <p className="text-[12px]" style={{ color: "var(--color-danger)" }}>{metaError}</p>}
                <p className="text-[11px] font-mono" style={{ color: "var(--color-text-muted)" }}>
                  {window.location.origin}/profile/{metaSlug}
                </p>
              </div>
              <div className="flex gap-2 justify-end">
                <Button size="sm" variant="secondary" onClick={() => { setShowMeta(false); setMetaName(profile.name); setMetaSlug(profile.slug); setMetaError(null); }}>Cancel</Button>
                <Button size="sm" disabled={metaSaving} onClick={handleMetaSave}>
                  {metaSaving ? "Saving..." : "Save"}
                </Button>
              </div>
            </div>
          )}
        </div>
      </div>

      {/* Mobile Edit / Preview tab switcher — hidden on md+ */}
      <div
        className="flex md:hidden border-b"
        style={{ borderColor: "var(--color-border)" }}
      >
        {(["edit", "preview"] as const).map((panel) => {
          const isActive = mobilePanel === panel;
          return (
            <button
              key={panel}
              onClick={() => setMobilePanel(panel)}
              className="px-5 py-2.5 text-[13px] font-medium transition-colors relative capitalize"
              style={{
                color: isActive ? "var(--color-text-primary)" : "var(--color-text-muted)",
                borderBottom: isActive
                  ? "2px solid var(--color-primary)"
                  : "2px solid transparent",
                marginBottom: "-1px",
              }}
            >
              {panel === "edit" ? "Edit" : "Preview"}
            </button>
          );
        })}
      </div>

      {/* Two-panel layout */}
      <div className="flex gap-6 flex-1 min-h-0" style={{ minHeight: "500px" }}>

        {/* Left: tabs + content — full-width on mobile when Edit tab active */}
        <div
          className={`flex-col gap-0 flex-1 min-w-0 ${mobilePanel === "edit" ? "flex" : "hidden md:flex"}`}
          style={{ maxWidth: "560px" }}
        >

          {/* Desktop tab bar (underline style) */}
          <div className="hidden md:flex border-b gap-0" style={{ borderColor: "var(--color-border)" }}>
            {TABS.map(({ id: tabId, label }) => (
              <button
                key={tabId}
                onClick={() => setActiveTab(tabId)}
                className="px-5 py-3 text-[13px] font-medium border-b-2 -mb-px transition-colors"
                style={{
                  borderBottomColor: activeTab === tabId ? "var(--color-primary)" : "transparent",
                  color: activeTab === tabId ? "var(--color-text-primary)" : "var(--color-text-muted)",
                }}
              >
                {label}
              </button>
            ))}
          </div>

          {/* Mobile tab pills */}
          <div className="md:hidden flex gap-2 overflow-x-auto pb-2 pt-1 -mx-4 px-4" style={{ scrollbarWidth: "none" }}>
            {TABS.map(({ id: tabId, label }) => (
              <button
                key={tabId}
                onClick={() => setActiveTab(tabId)}
                className="flex-shrink-0 px-4 py-2 rounded-full text-[13px] font-medium transition-colors"
                style={{
                  backgroundColor: activeTab === tabId ? "var(--color-primary)" : "var(--color-bg-accent)",
                  color: activeTab === tabId ? "var(--color-bg-sidebar)" : "var(--color-text-secondary)",
                }}
              >
                {label}
              </button>
            ))}
          </div>

          {/* Tab content */}
          <div className="flex-1 overflow-y-auto pt-4">
            {activeTab === "settings" && (
              <SettingsTab
                profile={profile}
                token={token}
                onSaved={handleSettingsSaved}
              />
            )}

            {activeTab === "node-pools" && (
              <NodePoolsTab
                profileId={profileId}
                nodePools={nodePools}
                loading={nodePoolsLoading}
                error={nodePoolsError}
                token={token}
                endpoints={endpoints}
                onChanged={handleNodePoolsChanged}
              />
            )}

            {activeTab === "rule-sets" && (
              <RuleSetsTab
                profileId={profileId}
                ruleSets={ruleSets}
                loading={ruleSetsLoading}
                error={ruleSetsError}
                token={token}
                endpoints={endpoints}
                onChanged={handleRuleSetsChanged}
              />
            )}

            {activeTab === "strategies" && (
              <StrategiesTab
                profileId={profileId}
                strategies={strategies}
                loading={strategiesLoading}
                error={strategiesError}
                token={token}
                onChanged={handleStrategiesChanged}
              />
            )}

            {activeTab === "routing-rules" && (
              <RoutingRulesTab
                profileId={profileId}
                routingRules={routingRules}
                loading={routingRulesLoading}
                error={routingRulesError}
                token={token}
                strategies={strategies}
                onChanged={handleRoutingRulesChanged}
              />
            )}
          </div>
        </div>

        {/* Right: Config preview — full-width on mobile when Preview tab active */}
        <div
          className={`flex-col gap-3 flex-1 min-w-0 overflow-hidden ${mobilePanel === "preview" ? "flex" : "hidden md:flex"}`}
        >
          <div className="flex items-center justify-between">
            <h2 className="text-[15px] font-semibold" style={{ color: "var(--color-text-primary)" }}>Config Preview</h2>
            <div className="flex items-center gap-2">
              {yamlLoading && (
                <Loader2
                  className="h-3.5 w-3.5 animate-spin"
                  style={{ color: "var(--color-text-muted)" }}
                />
              )}
              <span
                className="font-mono text-[11px] px-2 py-0.5 rounded-md"
                style={{
                  color: "var(--color-text-muted)",
                  backgroundColor: "var(--color-bg-accent)",
                  border: "1px solid var(--color-border)",
                }}
              >
                /profile/{profile.slug}
              </span>
            </div>
          </div>

          <div
            className="flex-1 rounded-xl border overflow-auto"
            style={{
              backgroundColor: "var(--color-bg-accent)",
              borderColor: "var(--color-border)",
              minHeight: "300px",
            }}
          >
            <pre
              className="p-4 text-xs font-mono leading-relaxed whitespace-pre"
              style={{ color: "var(--color-text-secondary)", minHeight: "100%" }}
            >
              {yaml || (yamlLoading ? "Loading\u2026" : "# Empty profile \u2014 add settings or node pools to see YAML.")}
            </pre>
          </div>
        </div>
      </div>
    </div>
  );
}
