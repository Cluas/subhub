import { useCallback, useEffect, useRef, useState } from "react";
import { ChevronDown, Loader2, X } from "lucide-react";
import { Button } from "@/components/ui/button";
import type { CreateProxyInput, Proxy } from "@/types/api";

// ── Types ─────────────────────────────────────────────────────────────────────

export interface AddNodeDialogProps {
  open: boolean;
  /** When set, dialog is in edit mode and fields are pre-populated. */
  editProxy?: Proxy | null;
  onSubmit: (data: CreateProxyInput) => Promise<void>;
  onCancel: () => void;
}

const PROXY_TYPES = ["ss", "vmess", "vless", "trojan", "hysteria2", "socks5"];

type ProxyConfig = Record<string, unknown>;

const EMPTY_FORM: CreateProxyInput = {
  name: "",
  type: "ss",
  server: "",
  port: 443,
  region: "",
  config: {},
};

// ── Default configs per protocol ─────────────────────────────────────────────

function defaultConfigForType(type: string): ProxyConfig {
  switch (type) {
    case "ss":
      return { cipher: "chacha20-ietf-poly1305", password: "", udp: false };
    case "vmess":
      return {
        uuid: "",
        alterId: 0,
        cipher: "auto",
        network: "tcp",
        tls: false,
        servername: "",
        "ws-opts": { path: "", headers: { Host: "" } },
      };
    case "vless":
      return {
        uuid: "",
        flow: "",
        network: "tcp",
        tls: true,
        servername: "",
        "client-fingerprint": "chrome",
        "reality-opts": { "public-key": "", "short-id": "" },
        "ws-opts": { path: "", headers: { Host: "" } },
        udp: true,
      };
    case "trojan":
      return {
        password: "",
        sni: "",
        "skip-cert-verify": false,
        udp: true,
      };
    case "hysteria2":
      return {
        password: "",
        obfs: "",
        "obfs-password": "",
        sni: "",
        "skip-cert-verify": false,
      };
    case "socks5":
      return { username: "", password: "", udp: true };
    default:
      return {};
  }
}

// ── Helpers ──────────────────────────────────────────────────────────────────

/** Deep-get a nested value by dot-separated path */
function getNestedValue(obj: ProxyConfig, path: string): unknown {
  const keys = path.split(".");
  let cur: unknown = obj;
  for (const k of keys) {
    if (cur == null || typeof cur !== "object") return undefined;
    cur = (cur as Record<string, unknown>)[k];
  }
  return cur;
}

/** Deep-set a nested value by dot-separated path, returns a new object */
function setNestedValue(obj: ProxyConfig, path: string, value: unknown): ProxyConfig {
  const keys = path.split(".");
  if (keys.length === 1) {
    return { ...obj, [keys[0]]: value };
  }
  const [head, ...rest] = keys;
  const child = (obj[head] as ProxyConfig) ?? {};
  return { ...obj, [head]: setNestedValue(child, rest.join("."), value) };
}

// ── Styling ──────────────────────────────────────────────────────────────────

const inputStyle: React.CSSProperties = {
  borderColor: "var(--color-input)",
  backgroundColor: "var(--color-background)",
  color: "var(--color-foreground)",
};

const inputClass =
  "rounded-md border px-3 py-2 text-sm outline-none focus:ring-2 w-full";

const selectClass =
  "rounded-md border px-3 py-2 text-sm outline-none focus:ring-2 w-full";

// ── Field component ──────────────────────────────────────────────────────────

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

// ── Collapsible section ──────────────────────────────────────────────────────

function CollapsibleSection({
  title,
  defaultOpen = false,
  children,
}: {
  title: string;
  defaultOpen?: boolean;
  children: React.ReactNode;
}) {
  const [isOpen, setIsOpen] = useState(defaultOpen);

  return (
    <div
      className="rounded-md border"
      style={{ borderColor: "var(--color-border)" }}
    >
      <button
        type="button"
        onClick={() => setIsOpen((o) => !o)}
        className="flex w-full items-center justify-between px-3 py-2 text-sm font-medium hover:bg-[var(--color-muted)] transition-colors rounded-md"
        style={{ color: "var(--color-muted-foreground)" }}
      >
        {title}
        <ChevronDown
          className={`h-4 w-4 transition-transform ${isOpen ? "rotate-180" : ""}`}
        />
      </button>
      {isOpen && (
        <div className="px-3 pb-3 pt-1 flex flex-col gap-3">{children}</div>
      )}
    </div>
  );
}

// ── Checkbox field ───────────────────────────────────────────────────────────

function CheckboxField({
  label,
  checked,
  onChange,
}: {
  label: string;
  checked: boolean;
  onChange: (v: boolean) => void;
}) {
  return (
    <label className="flex items-center gap-2 text-sm cursor-pointer select-none">
      <input
        type="checkbox"
        checked={checked}
        onChange={(e) => onChange(e.target.checked)}
        className="rounded border"
      />
      {label}
    </label>
  );
}

// ── Protocol-specific fields ─────────────────────────────────────────────────

interface ProtocolFieldsProps {
  type: string;
  config: ProxyConfig;
  onChange: (config: ProxyConfig) => void;
}

function ProtocolFields({ type, config, onChange }: ProtocolFieldsProps) {
  const get = (path: string) => getNestedValue(config, path);
  const set = (path: string, value: unknown) =>
    onChange(setNestedValue(config, path, value));

  switch (type) {
    // ── Shadowsocks ────────────────────────────────────────────────────────
    case "ss":
      return (
        <>
          <Field label="Cipher" required>
            <select
              value={(get("cipher") as string) ?? "chacha20-ietf-poly1305"}
              onChange={(e) => set("cipher", e.target.value)}
              className={selectClass}
              style={inputStyle}
            >
              {[
                "chacha20-ietf-poly1305",
                "aes-256-gcm",
                "aes-128-gcm",
                "aes-256-cfb",
                "aes-128-cfb",
                "rc4-md5",
              ].map((c) => (
                <option key={c} value={c}>
                  {c}
                </option>
              ))}
            </select>
          </Field>
          <Field label="Password" required>
            <input
              type="text"
              required
              value={(get("password") as string) ?? ""}
              onChange={(e) => set("password", e.target.value)}
              placeholder="Password"
              className={inputClass}
              style={inputStyle}
            />
          </Field>
          <CheckboxField
            label="UDP"
            checked={(get("udp") as boolean) ?? false}
            onChange={(v) => set("udp", v)}
          />
        </>
      );

    // ── VMess ──────────────────────────────────────────────────────────────
    case "vmess": {
      const network = (get("network") as string) ?? "tcp";
      const tls = (get("tls") as boolean) ?? false;
      return (
        <>
          <Field label="UUID" required>
            <input
              type="text"
              required
              value={(get("uuid") as string) ?? ""}
              onChange={(e) => set("uuid", e.target.value)}
              placeholder="xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
              className={inputClass}
              style={inputStyle}
            />
          </Field>
          <div className="grid grid-cols-2 gap-3">
            <Field label="Alter ID">
              <input
                type="number"
                min={0}
                value={(get("alterId") as number) ?? 0}
                onChange={(e) => set("alterId", Number(e.target.value))}
                className={inputClass}
                style={{ ...inputStyle, fontVariantNumeric: "tabular-nums" }}
              />
            </Field>
            <Field label="Cipher">
              <select
                value={(get("cipher") as string) ?? "auto"}
                onChange={(e) => set("cipher", e.target.value)}
                className={selectClass}
                style={inputStyle}
              >
                {["auto", "aes-128-gcm", "chacha20-poly1305", "none"].map(
                  (c) => (
                    <option key={c} value={c}>
                      {c}
                    </option>
                  )
                )}
              </select>
            </Field>
          </div>
          <Field label="Network">
            <select
              value={network}
              onChange={(e) => set("network", e.target.value)}
              className={selectClass}
              style={inputStyle}
            >
              {["tcp", "ws", "grpc", "h2"].map((n) => (
                <option key={n} value={n}>
                  {n}
                </option>
              ))}
            </select>
          </Field>
          <CheckboxField
            label="TLS"
            checked={tls}
            onChange={(v) => set("tls", v)}
          />
          {tls && (
            <Field label="Server Name (SNI)">
              <input
                type="text"
                value={(get("servername") as string) ?? ""}
                onChange={(e) => set("servername", e.target.value)}
                placeholder="example.com"
                className={inputClass}
                style={inputStyle}
              />
            </Field>
          )}
          {network === "ws" && (
            <CollapsibleSection title="WebSocket Options" defaultOpen>
              <Field label="Path">
                <input
                  type="text"
                  value={(get("ws-opts.path") as string) ?? ""}
                  onChange={(e) => set("ws-opts.path", e.target.value)}
                  placeholder="/path"
                  className={inputClass}
                  style={inputStyle}
                />
              </Field>
              <Field label="Host Header">
                <input
                  type="text"
                  value={(get("ws-opts.headers.Host") as string) ?? ""}
                  onChange={(e) => set("ws-opts.headers.Host", e.target.value)}
                  placeholder="example.com"
                  className={inputClass}
                  style={inputStyle}
                />
              </Field>
            </CollapsibleSection>
          )}
        </>
      );
    }

    // ── VLESS ──────────────────────────────────────────────────────────────
    case "vless": {
      const network = (get("network") as string) ?? "tcp";
      return (
        <>
          <Field label="UUID" required>
            <input
              type="text"
              required
              value={(get("uuid") as string) ?? ""}
              onChange={(e) => set("uuid", e.target.value)}
              placeholder="xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
              className={inputClass}
              style={inputStyle}
            />
          </Field>
          <div className="grid grid-cols-2 gap-3">
            <Field label="Flow">
              <select
                value={(get("flow") as string) ?? ""}
                onChange={(e) => set("flow", e.target.value)}
                className={selectClass}
                style={inputStyle}
              >
                <option value="">None</option>
                <option value="xtls-rprx-vision">xtls-rprx-vision</option>
              </select>
            </Field>
            <Field label="Network">
              <select
                value={network}
                onChange={(e) => set("network", e.target.value)}
                className={selectClass}
                style={inputStyle}
              >
                {["tcp", "ws", "grpc", "h2"].map((n) => (
                  <option key={n} value={n}>
                    {n}
                  </option>
                ))}
              </select>
            </Field>
          </div>
          <CheckboxField
            label="TLS"
            checked={(get("tls") as boolean) ?? true}
            onChange={(v) => set("tls", v)}
          />
          <div className="grid grid-cols-2 gap-3">
            <Field label="Server Name">
              <input
                type="text"
                value={(get("servername") as string) ?? ""}
                onChange={(e) => set("servername", e.target.value)}
                placeholder="example.com"
                className={inputClass}
                style={inputStyle}
              />
            </Field>
            <Field label="Client Fingerprint">
              <select
                value={
                  (get("client-fingerprint") as string) ?? "chrome"
                }
                onChange={(e) => set("client-fingerprint", e.target.value)}
                className={selectClass}
                style={inputStyle}
              >
                {["chrome", "firefox", "safari", "edge", "random"].map((f) => (
                  <option key={f} value={f}>
                    {f}
                  </option>
                ))}
              </select>
            </Field>
          </div>
          <CollapsibleSection title="Reality Options">
            <Field label="Public Key">
              <input
                type="text"
                value={(get("reality-opts.public-key") as string) ?? ""}
                onChange={(e) =>
                  set("reality-opts.public-key", e.target.value)
                }
                placeholder="Public key"
                className={inputClass}
                style={inputStyle}
              />
            </Field>
            <Field label="Short ID">
              <input
                type="text"
                value={(get("reality-opts.short-id") as string) ?? ""}
                onChange={(e) =>
                  set("reality-opts.short-id", e.target.value)
                }
                placeholder="Short ID"
                className={inputClass}
                style={inputStyle}
              />
            </Field>
          </CollapsibleSection>
          {network === "ws" && (
            <CollapsibleSection title="WebSocket Options" defaultOpen>
              <Field label="Path">
                <input
                  type="text"
                  value={(get("ws-opts.path") as string) ?? ""}
                  onChange={(e) => set("ws-opts.path", e.target.value)}
                  placeholder="/path"
                  className={inputClass}
                  style={inputStyle}
                />
              </Field>
              <Field label="Host Header">
                <input
                  type="text"
                  value={(get("ws-opts.headers.Host") as string) ?? ""}
                  onChange={(e) =>
                    set("ws-opts.headers.Host", e.target.value)
                  }
                  placeholder="example.com"
                  className={inputClass}
                  style={inputStyle}
                />
              </Field>
            </CollapsibleSection>
          )}
          <CheckboxField
            label="UDP"
            checked={(get("udp") as boolean) ?? true}
            onChange={(v) => set("udp", v)}
          />
        </>
      );
    }

    // ── Trojan ─────────────────────────────────────────────────────────────
    case "trojan":
      return (
        <>
          <Field label="Password" required>
            <input
              type="text"
              required
              value={(get("password") as string) ?? ""}
              onChange={(e) => set("password", e.target.value)}
              placeholder="Password"
              className={inputClass}
              style={inputStyle}
            />
          </Field>
          <Field label="SNI">
            <input
              type="text"
              value={(get("sni") as string) ?? ""}
              onChange={(e) => set("sni", e.target.value)}
              placeholder="example.com"
              className={inputClass}
              style={inputStyle}
            />
          </Field>
          <CheckboxField
            label="Skip Certificate Verify"
            checked={(get("skip-cert-verify") as boolean) ?? false}
            onChange={(v) => set("skip-cert-verify", v)}
          />
          <CheckboxField
            label="UDP"
            checked={(get("udp") as boolean) ?? true}
            onChange={(v) => set("udp", v)}
          />
        </>
      );

    // ── Hysteria2 ──────────────────────────────────────────────────────────
    case "hysteria2": {
      const obfs = (get("obfs") as string) ?? "";
      return (
        <>
          <Field label="Password" required>
            <input
              type="text"
              required
              value={(get("password") as string) ?? ""}
              onChange={(e) => set("password", e.target.value)}
              placeholder="Password"
              className={inputClass}
              style={inputStyle}
            />
          </Field>
          <Field label="Obfuscation">
            <select
              value={obfs}
              onChange={(e) => set("obfs", e.target.value)}
              className={selectClass}
              style={inputStyle}
            >
              <option value="">None</option>
              <option value="salamander">salamander</option>
            </select>
          </Field>
          {obfs !== "" && (
            <Field label="Obfuscation Password">
              <input
                type="text"
                value={(get("obfs-password") as string) ?? ""}
                onChange={(e) => set("obfs-password", e.target.value)}
                placeholder="Obfuscation password"
                className={inputClass}
                style={inputStyle}
              />
            </Field>
          )}
          <Field label="SNI">
            <input
              type="text"
              value={(get("sni") as string) ?? ""}
              onChange={(e) => set("sni", e.target.value)}
              placeholder="example.com"
              className={inputClass}
              style={inputStyle}
            />
          </Field>
          <CheckboxField
            label="Skip Certificate Verify"
            checked={(get("skip-cert-verify") as boolean) ?? false}
            onChange={(v) => set("skip-cert-verify", v)}
          />
        </>
      );
    }

    // ── SOCKS5 ─────────────────────────────────────────────────────────────
    case "socks5":
      return (
        <>
          <Field label="Username">
            <input
              type="text"
              value={(get("username") as string) ?? ""}
              onChange={(e) => set("username", e.target.value)}
              placeholder="Username"
              className={inputClass}
              style={inputStyle}
            />
          </Field>
          <Field label="Password">
            <input
              type="text"
              value={(get("password") as string) ?? ""}
              onChange={(e) => set("password", e.target.value)}
              placeholder="Password"
              className={inputClass}
              style={inputStyle}
            />
          </Field>
          <CheckboxField
            label="UDP"
            checked={(get("udp") as boolean) ?? true}
            onChange={(v) => set("udp", v)}
          />
        </>
      );

    default:
      return null;
  }
}

// ── Clean config before submission ───────────────────────────────────────────

/** Strip empty strings and empty nested objects so we only send meaningful values */
function cleanConfig(config: ProxyConfig): ProxyConfig | undefined {
  const result: ProxyConfig = {};
  for (const [key, value] of Object.entries(config)) {
    if (value === "" || value === undefined || value === null) continue;
    if (typeof value === "object" && !Array.isArray(value)) {
      const nested = cleanConfig(value as ProxyConfig);
      if (nested && Object.keys(nested).length > 0) {
        result[key] = nested;
      }
    } else {
      result[key] = value;
    }
  }
  return Object.keys(result).length > 0 ? result : undefined;
}

// ── Validate protocol-specific required fields ───────────────────────────────

function validateConfig(type: string, config: ProxyConfig): string | null {
  switch (type) {
    case "ss":
      if (!config.password) return "Password is required for Shadowsocks.";
      break;
    case "vmess":
      if (!config.uuid) return "UUID is required for VMess.";
      break;
    case "vless":
      if (!config.uuid) return "UUID is required for VLESS.";
      break;
    case "trojan":
      if (!config.password) return "Password is required for Trojan.";
      break;
    case "hysteria2":
      if (!config.password) return "Password is required for Hysteria2.";
      break;
  }
  return null;
}

// ── Dialog ────────────────────────────────────────────────────────────────────

export function AddNodeDialog({
  open,
  editProxy,
  onSubmit,
  onCancel,
}: AddNodeDialogProps) {
  const isEdit = !!editProxy;
  const [form, setForm] = useState<CreateProxyInput>(EMPTY_FORM);
  const [config, setConfig] = useState<ProxyConfig>(
    defaultConfigForType("ss")
  );
  const [formError, setFormError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  // Sync form when dialog opens/switches between add and edit mode
  useEffect(() => {
    if (!open) return;
    if (editProxy) {
      setForm({
        name: editProxy.name,
        type: editProxy.type,
        server: editProxy.server,
        port: editProxy.port,
        region: editProxy.region ?? "",
      });
      // Merge defaults with saved config so all fields have values
      setConfig({
        ...defaultConfigForType(editProxy.type),
        ...(editProxy.config ?? {}),
      });
    } else {
      setForm(EMPTY_FORM);
      setConfig(defaultConfigForType("ss"));
    }
    setFormError(null);
  }, [open, editProxy]);

  // When protocol type changes, reset config to defaults for that protocol
  function handleTypeChange(newType: string) {
    setForm((f) => ({ ...f, type: newType }));
    setConfig(defaultConfigForType(newType));
  }

  // Escape key closes dialog
  useEffect(() => {
    if (!open) return;
    function handleKey(e: KeyboardEvent) {
      if (e.key === "Escape") onCancel();
    }
    document.addEventListener("keydown", handleKey);
    return () => document.removeEventListener("keydown", handleKey);
  }, [open, onCancel]);

  // Auto-focus the name field on open
  const nameRef = useCallback(
    (el: HTMLInputElement | null) => {
      if (el && open) {
        setTimeout(() => el.focus(), 50);
      }
    },
    [open]
  );

  const panelRef = useRef<HTMLDivElement>(null);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();

    if (!form.name.trim()) {
      setFormError("Name is required.");
      return;
    }
    if (!form.server.trim()) {
      setFormError("Server is required.");
      return;
    }
    const port = Number(form.port);
    if (!port || port < 1 || port > 65535) {
      setFormError("Port must be between 1 and 65535.");
      return;
    }

    // Validate protocol-specific required fields
    const configError = validateConfig(form.type, config);
    if (configError) {
      setFormError(configError);
      return;
    }

    setFormError(null);
    setSubmitting(true);
    try {
      await onSubmit({
        ...form,
        port,
        region: form.region?.trim() || undefined,
        config: cleanConfig(config),
      });
    } catch (err: unknown) {
      setFormError(err instanceof Error ? err.message : "Failed to save node.");
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
      aria-labelledby="add-node-dialog-title"
    >
      {/* Backdrop -- click outside to dismiss */}
      <div
        className="absolute inset-0 bg-black/50 backdrop-blur-[2px]"
        style={{ transition: "opacity 150ms ease" }}
        onClick={onCancel}
      />

      {/* Panel */}
      <div
        ref={panelRef}
        className="relative rounded-xl border shadow-2xl p-6 flex flex-col gap-5 w-full max-w-lg mx-4 max-h-[90vh] overflow-y-auto"
        style={{
          backgroundColor: "var(--color-card)",
          borderColor: "var(--color-border)",
          boxShadow: "0 0 0 1px rgba(0,0,0,0.06), 0 8px 32px rgba(0,0,0,0.18)",
        }}
      >
        {/* Header */}
        <div className="flex items-center justify-between gap-3">
          <h2
            id="add-node-dialog-title"
            className="font-semibold text-base"
            style={{
              color: "var(--color-foreground)",
              textWrap: "balance",
            } as React.CSSProperties}
          >
            {isEdit ? "Edit Node" : "Add Node"}
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
            This node will be self-managed (not bound to any subscription).
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
          {/* Common fields */}
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
            {/* Name */}
            <div className="sm:col-span-2">
              <Field label="Name" required>
                <input
                  ref={nameRef}
                  type="text"
                  required
                  value={form.name}
                  onChange={(e) => setForm((f) => ({ ...f, name: e.target.value }))}
                  placeholder="My Node"
                  className={inputClass}
                  style={inputStyle}
                />
              </Field>
            </div>

            {/* Type */}
            <Field label="Type" required>
              <select
                value={form.type}
                onChange={(e) => handleTypeChange(e.target.value)}
                className={selectClass}
                style={inputStyle}
              >
                {PROXY_TYPES.map((t) => (
                  <option key={t} value={t}>
                    {t}
                  </option>
                ))}
              </select>
            </Field>

            {/* Region */}
            <Field label="Region">
              <input
                type="text"
                value={form.region ?? ""}
                onChange={(e) => setForm((f) => ({ ...f, region: e.target.value }))}
                placeholder="US, HK, JP..."
                className={inputClass}
                style={inputStyle}
              />
            </Field>

            {/* Server */}
            <Field label="Server" required>
              <input
                type="text"
                required
                value={form.server}
                onChange={(e) => setForm((f) => ({ ...f, server: e.target.value }))}
                placeholder="1.2.3.4 or hostname"
                className={inputClass}
                style={inputStyle}
              />
            </Field>

            {/* Port */}
            <Field label="Port" required>
              <input
                type="number"
                required
                min={1}
                max={65535}
                value={form.port}
                onChange={(e) =>
                  setForm((f) => ({ ...f, port: Number(e.target.value) }))
                }
                placeholder="443"
                className="rounded-md border px-3 py-2 text-sm outline-none focus:ring-2 w-full"
                style={{ ...inputStyle, fontVariantNumeric: "tabular-nums" }}
              />
            </Field>
          </div>

          {/* Protocol-specific fields */}
          <div
            className="flex flex-col gap-3 rounded-md border p-4"
            style={{
              borderColor: "var(--color-border)",
              backgroundColor: "var(--color-muted)",
            }}
          >
            <p
              className="text-xs font-medium uppercase tracking-wide"
              style={{ color: "var(--color-muted-foreground)" }}
            >
              {form.type} Settings
            </p>
            <ProtocolFields
              type={form.type}
              config={config}
              onChange={setConfig}
            />
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
                "Add Node"
              )}
            </Button>
          </div>
        </form>
      </div>
    </div>
  );
}
