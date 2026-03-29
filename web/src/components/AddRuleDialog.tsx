import { useCallback, useEffect, useRef, useState } from "react";
import { Loader2, X } from "lucide-react";
import { Button } from "@/components/ui/button";
import type { CreateRuleInput, Rule } from "@/types/api";

// ── Types ─────────────────────────────────────────────────────────────────────

export interface AddRuleDialogProps {
  open: boolean;
  editRule?: Rule | null;
  onSubmit: (data: CreateRuleInput) => Promise<void>;
  onCancel: () => void;
}

const RULE_TYPES = [
  "DOMAIN",
  "DOMAIN-SUFFIX",
  "DOMAIN-KEYWORD",
  "DOMAIN-REGEX",
  "IP-CIDR",
  "IP-CIDR6",
  "SRC-IP-CIDR",
  "SRC-PORT",
  "DST-PORT",
  "GEOSITE",
  "GEOIP",
  "RULE-SET",
  "PROCESS-NAME",
  "MATCH",
];

const TYPES_WITHOUT_PAYLOAD = new Set(["MATCH"]);
const TYPES_WITH_NO_RESOLVE = new Set(["IP-CIDR", "IP-CIDR6", "SRC-IP-CIDR", "GEOIP"]);

function getPayloadMeta(type: string): { label: string; placeholder: string } {
  switch (type) {
    case "DOMAIN":
      return { label: "Domain", placeholder: "example.com" };
    case "DOMAIN-SUFFIX":
      return { label: "Domain Suffix", placeholder: "google.com" };
    case "DOMAIN-KEYWORD":
      return { label: "Keyword", placeholder: "google" };
    case "DOMAIN-REGEX":
      return { label: "Regex", placeholder: ".*\\.google\\.com$" };
    case "IP-CIDR":
    case "IP-CIDR6":
    case "SRC-IP-CIDR":
      return { label: "CIDR", placeholder: "10.0.0.0/8" };
    case "DST-PORT":
    case "SRC-PORT":
      return { label: "Port", placeholder: "443" };
    case "GEOSITE":
      return { label: "Tag", placeholder: "cn, google, telegram, github..." };
    case "GEOIP":
      return { label: "Country Code", placeholder: "CN, US, JP..." };
    case "RULE-SET":
      return { label: "Provider Name", placeholder: "my-rule-provider" };
    case "PROCESS-NAME":
      return { label: "Process", placeholder: "chrome.exe" };
    default:
      return { label: "Payload", placeholder: "..." };
  }
}

const EMPTY_FORM: CreateRuleInput = {
  type: "DOMAIN-SUFFIX",
  payload: "",
  target: "",
  provider_name: "",
};

// ── Input helper ──────────────────────────────────────────────────────────────

const inputStyle: React.CSSProperties = {
  borderColor: "var(--color-input)",
  backgroundColor: "var(--color-background)",
  color: "var(--color-foreground)",
};

function Field({
  label,
  required,
  children,
}: {
  label: string;
  required?: boolean;
  children: React.ReactNode;
}) {
  return (
    <div className="flex flex-col gap-1.5">
      <label className="text-sm font-medium">
        {label}{" "}
        {required && (
          <span style={{ color: "var(--color-destructive)" }}>*</span>
        )}
      </label>
      {children}
    </div>
  );
}

// ── Dialog ────────────────────────────────────────────────────────────────────

export function AddRuleDialog({
  open,
  editRule,
  onSubmit,
  onCancel,
}: AddRuleDialogProps) {
  const isEdit = !!editRule;
  const [form, setForm] = useState<CreateRuleInput>(EMPTY_FORM);
  const [noResolve, setNoResolve] = useState(false);
  const [formError, setFormError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  // Sync form when dialog opens or switches between add/edit
  useEffect(() => {
    if (!open) return;
    if (editRule) {
      // Strip ",no-resolve" suffix from payload if present
      let payload = editRule.payload;
      let nr = false;
      if (payload.endsWith(",no-resolve")) {
        payload = payload.slice(0, -",no-resolve".length);
        nr = true;
      }
      setForm({
        type: editRule.type,
        payload,
        target: editRule.target,
        provider_name: editRule.provider_name ?? "",
      });
      setNoResolve(nr);
    } else {
      setForm(EMPTY_FORM);
      setNoResolve(false);
    }
    setFormError(null);
  }, [open, editRule]);

  // Escape key closes dialog
  useEffect(() => {
    if (!open) return;
    function handleKey(e: KeyboardEvent) {
      if (e.key === "Escape") onCancel();
    }
    document.addEventListener("keydown", handleKey);
    return () => document.removeEventListener("keydown", handleKey);
  }, [open, onCancel]);

  const payloadRef = useCallback(
    (el: HTMLInputElement | null) => {
      if (el && open) {
        setTimeout(() => el.focus(), 50);
      }
    },
    [open]
  );

  const panelRef = useRef<HTMLDivElement>(null);

  const needsPayload = !TYPES_WITHOUT_PAYLOAD.has(form.type);
  const showNoResolve = TYPES_WITH_NO_RESOLVE.has(form.type);
  const payloadMeta = getPayloadMeta(form.type);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();

    if (needsPayload && !form.payload.trim()) {
      setFormError(`${payloadMeta.label} is required.`);
      return;
    }
    if (!form.target.trim()) {
      setFormError("Target is required.");
      return;
    }

    setFormError(null);
    setSubmitting(true);
    try {
      let payload = needsPayload ? form.payload.trim() : "";
      if (showNoResolve && noResolve && payload) {
        payload += ",no-resolve";
      }
      await onSubmit({
        ...form,
        payload,
        target: form.target.trim(),
        provider_name: form.provider_name?.trim() || undefined,
      });
    } catch (err: unknown) {
      setFormError(err instanceof Error ? err.message : "Failed to save rule.");
    } finally {
      setSubmitting(false);
    }
  }

  if (!open) return null;

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center"
      role="dialog"
      aria-modal="true"
      aria-labelledby="add-rule-dialog-title"
    >
      {/* Backdrop */}
      <div
        className="absolute inset-0 bg-black/50 backdrop-blur-[2px]"
        onClick={onCancel}
      />

      {/* Panel */}
      <div
        ref={panelRef}
        className="relative rounded-xl border shadow-2xl p-6 flex flex-col gap-5 w-full max-w-lg mx-4"
        style={{
          backgroundColor: "var(--color-card)",
          borderColor: "var(--color-border)",
          boxShadow: "0 0 0 1px rgba(0,0,0,0.06), 0 8px 32px rgba(0,0,0,0.18)",
        }}
      >
        {/* Header */}
        <div className="flex items-center justify-between gap-3">
          <h2
            id="add-rule-dialog-title"
            className="font-semibold text-base"
            style={{
              color: "var(--color-foreground)",
              textWrap: "balance",
            } as React.CSSProperties}
          >
            {isEdit ? "Edit Rule" : "Add Rule"}
          </h2>
          <button
            type="button"
            onClick={onCancel}
            className="shrink-0 rounded-md p-1.5 hover:bg-[var(--color-muted)] transition-colors"
            aria-label="Close"
            style={{ minWidth: 32, minHeight: 32 }}
          >
            <X className="h-4 w-4" style={{ color: "var(--color-muted-foreground)" }} />
          </button>
        </div>

        {/* Self-managed notice */}
        {!isEdit && (
          <p
            className="text-xs rounded-md px-3 py-2"
            style={{
              backgroundColor: "var(--color-muted)",
              color: "var(--color-muted-foreground)",
            }}
          >
            This rule will be self-managed (not bound to any subscription).
          </p>
        )}

        {/* Form error */}
        {formError && (
          <p className="text-sm" style={{ color: "var(--color-destructive)" }}>
            {formError}
          </p>
        )}

        {/* Form */}
        <form onSubmit={handleSubmit} className="flex flex-col gap-4">
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
            {/* Type */}
            <Field label="Type" required>
              <select
                value={form.type}
                onChange={(e) => {
                  const newType = e.target.value;
                  setForm((f) => ({ ...f, type: newType }));
                  // Reset no-resolve when switching away from applicable types
                  if (!TYPES_WITH_NO_RESOLVE.has(newType)) {
                    setNoResolve(false);
                  }
                }}
                className="rounded-md border px-3 py-2 text-sm outline-none focus:ring-2"
                style={inputStyle}
              >
                {RULE_TYPES.map((t) => (
                  <option key={t} value={t}>
                    {t}
                  </option>
                ))}
              </select>
            </Field>

            {/* Target */}
            <Field label="Target" required>
              <input
                type="text"
                required
                value={form.target}
                onChange={(e) => setForm((f) => ({ ...f, target: e.target.value }))}
                placeholder="PROXY, DIRECT, 🔰 节点选择..."
                className="rounded-md border px-3 py-2 text-sm outline-none focus:ring-2"
                style={inputStyle}
              />
            </Field>

            {/* Payload — dynamic label, hidden for MATCH */}
            {needsPayload && (
              <div className="sm:col-span-2">
                <Field label={payloadMeta.label} required>
                  <input
                    ref={payloadRef}
                    type="text"
                    required
                    value={form.payload}
                    onChange={(e) =>
                      setForm((f) => ({ ...f, payload: e.target.value }))
                    }
                    placeholder={payloadMeta.placeholder}
                    className="rounded-md border px-3 py-2 text-sm outline-none focus:ring-2 w-full font-mono"
                    style={inputStyle}
                  />
                </Field>
              </div>
            )}

            {/* no-resolve checkbox for IP/GEO types */}
            {showNoResolve && (
              <div className="sm:col-span-2">
                <label className="flex items-center gap-2 text-sm cursor-pointer">
                  <input
                    type="checkbox"
                    checked={noResolve}
                    onChange={(e) => setNoResolve(e.target.checked)}
                  />
                  <span>no-resolve</span>
                  <span
                    className="text-xs"
                    style={{ color: "var(--color-muted-foreground)" }}
                  >
                    (skip DNS resolution for this rule)
                  </span>
                </label>
              </div>
            )}

            {/* Provider name */}
            <div className="sm:col-span-2">
              <Field label="Provider Name">
                <input
                  type="text"
                  value={form.provider_name ?? ""}
                  onChange={(e) =>
                    setForm((f) => ({ ...f, provider_name: e.target.value }))
                  }
                  placeholder="custom (optional)"
                  className="rounded-md border px-3 py-2 text-sm outline-none focus:ring-2"
                  style={inputStyle}
                />
              </Field>
            </div>
          </div>

          {/* Actions */}
          <div className="flex gap-2 justify-end pt-1">
            <Button type="button" variant="outline" onClick={onCancel}>
              Cancel
            </Button>
            <Button type="submit" disabled={submitting}>
              {submitting ? (
                <>
                  <Loader2 className="h-4 w-4 animate-spin" />
                  {isEdit ? "Saving..." : "Adding..."}
                </>
              ) : isEdit ? (
                "Save Changes"
              ) : (
                "Add Rule"
              )}
            </Button>
          </div>
        </form>
      </div>
    </div>
  );
}
