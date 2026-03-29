import { useCallback, useEffect, useState } from "react";
import { Eye, EyeOff } from "lucide-react";
import { useAuth } from "@/context/AuthContext";
import { apiFetch } from "@/lib/api";
import { Button } from "@/components/ui/button";

interface Settings {
  port: string;
  db_path: string;
  base_url: string;
  cache_ttl_seconds: number;
  cache_max_entries: number;
  cors_origins: string;
  log_level: string;
  api_token_set: boolean;
}

// ── Shared input styles ───────────────────────────────────────────────────────

const inputClass =
  "h-10 w-full rounded-[var(--radius)] bg-[var(--color-bg)] border border-[var(--color-border)] px-3 text-[13px] text-[var(--color-text-primary)] outline-none focus:border-[var(--color-primary)] transition-colors";

const readOnlyClass =
  "h-10 w-full rounded-[var(--radius)] bg-[var(--color-bg)] border border-[var(--color-border)] px-3 text-[13px] text-[var(--color-text-primary)] outline-none opacity-60 cursor-not-allowed";

// ── Section card ──────────────────────────────────────────────────────────────

function SectionCard({
  title,
  subtitle,
  children,
}: {
  title: string;
  subtitle: string;
  children: React.ReactNode;
}) {
  return (
    <div
      className="rounded-xl border p-6 mb-4"
      style={{
        backgroundColor: "var(--color-bg-card)",
        borderColor: "var(--color-border)",
      }}
    >
      <h2 className="text-[15px] font-semibold text-[var(--color-text-primary)] mb-1">{title}</h2>
      <p className="text-[12px] text-[var(--color-text-muted)] mb-4">{subtitle}</p>
      {children}
    </div>
  );
}

// ── Field label ───────────────────────────────────────────────────────────────

function FieldLabel({ children }: { children: React.ReactNode }) {
  return (
    <label className="block text-[12px] text-[var(--color-text-secondary)] mb-1.5">
      {children}
    </label>
  );
}

// ── SettingsPage ──────────────────────────────────────────────────────────────

export function SettingsPage() {
  const { token, setToken, clearToken } = useAuth();
  const [settings, setSettings] = useState<Settings | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [tokenInput, setTokenInput] = useState("");
  const [showToken, setShowToken] = useState(false);
  const [saved, setSaved] = useState(false);

  // Base URL state
  const [baseUrl, setBaseUrl] = useState("");
  const [baseUrlSaving, setBaseUrlSaving] = useState(false);
  const [baseUrlSaved, setBaseUrlSaved] = useState(false);
  const [baseUrlError, setBaseUrlError] = useState<string | null>(null);

  const loadSettings = useCallback((currentToken: string) => {
    setError(null);
    apiFetch<Settings>("/api/settings", { token: currentToken })
      .then((s) => {
        setSettings(s);
        setBaseUrl(s.base_url || "");
      })
      .catch((e) => setError(e.message));
  }, []);

  useEffect(() => {
    loadSettings(token);
  }, [token, loadSettings]);

  const handleSave = () => {
    const trimmed = tokenInput.trim();
    if (!trimmed) return;
    setToken(trimmed);
    setSaved(true);
    setTimeout(() => setSaved(false), 2000);
  };

  const handleClear = () => {
    clearToken();
    setTokenInput("");
  };

  async function handleBaseUrlSave() {
    setBaseUrlSaving(true);
    setBaseUrlError(null);
    setBaseUrlSaved(false);
    try {
      await apiFetch("/api/settings", {
        token,
        method: "PUT",
        body: { base_url: baseUrl },
      });
      setBaseUrlSaved(true);
      if (settings) setSettings({ ...settings, base_url: baseUrl });
      setTimeout(() => setBaseUrlSaved(false), 2000);
    } catch (err: unknown) {
      setBaseUrlError(err instanceof Error ? err.message : "Failed to save");
    } finally {
      setBaseUrlSaving(false);
    }
  }

  return (
    <div className="flex flex-col">
      {/* Page header */}
      <h1 className="text-[26px] font-bold text-[var(--color-text-primary)] mb-1">Settings</h1>
      <p className="text-[13px] text-[var(--color-text-secondary)] mb-8">Configure server and application settings</p>

      {/* Error banner */}
      {error && (
        <div
          className="rounded-md border px-4 py-3 text-[13px] mb-4"
          style={{
            backgroundColor: "var(--color-danger-bg)",
            color: "var(--color-danger)",
            borderColor: "var(--color-danger)",
          }}
        >
          Failed to load settings: {error}
        </div>
      )}

      {/* Section 1: API Token */}
      <SectionCard title="API Token" subtitle="Authentication token for API access">
        <FieldLabel>Bearer Token</FieldLabel>
        <div className="relative">
          <input
            type={showToken ? "text" : "password"}
            placeholder="Enter your API token…"
            value={tokenInput}
            onChange={(e) => setTokenInput(e.target.value)}
            onKeyDown={(e) => e.key === "Enter" && handleSave()}
            className={inputClass + " pr-10"}
          />
          <button
            type="button"
            onClick={() => setShowToken((v) => !v)}
            className="absolute right-3 top-1/2 -translate-y-1/2 text-[var(--color-text-muted)] hover:text-[var(--color-text-secondary)] transition-colors"
            aria-label={showToken ? "Hide token" : "Show token"}
          >
            {showToken ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
          </button>
        </div>
        {token && (
          <div className="mt-3 flex items-center gap-2">
            <span className="text-[12px] text-[var(--color-success)]">Token is set</span>
            <button
              onClick={handleClear}
              className="text-[12px] text-[var(--color-danger)] underline hover:no-underline"
            >
              Clear token
            </button>
          </div>
        )}
      </SectionCard>

      {/* Section 2: Base URL */}
      <SectionCard title="Base URL" subtitle="Public address used in generated proxy-provider and rule-provider URLs">
        <div className="flex flex-col gap-2">
          <FieldLabel>Base URL</FieldLabel>
          <div className="flex gap-2">
            <input
              type="text"
              value={baseUrl}
              onChange={(e) => { setBaseUrl(e.target.value); setBaseUrlSaved(false); }}
              placeholder="http://192.168.2.222:9000"
              className={inputClass + " flex-1"}
            />
            <Button
              size="sm"
              disabled={baseUrlSaving || baseUrl === (settings?.base_url ?? "")}
              onClick={handleBaseUrlSave}
            >
              {baseUrlSaving ? "Saving..." : baseUrlSaved ? "Saved ✓" : "Save"}
            </Button>
          </div>
          {baseUrlError && <p className="text-[12px] text-[var(--color-danger)]">{baseUrlError}</p>}
          <p className="text-[11px] text-[var(--color-text-muted)]">
            Profile and endpoint URLs will use this address (e.g. {baseUrl || "http://localhost:9000"}/profile/daily)
          </p>
        </div>
      </SectionCard>

      {/* Section 3: Server Configuration */}
      <SectionCard title="Server Configuration" subtitle="Core server settings (read-only, set via environment variables)">
        <div className="flex flex-col gap-3">
          <div>
            <FieldLabel>Port</FieldLabel>
            <input
              type="text"
              readOnly
              value={settings?.port ?? "—"}
              className={readOnlyClass}
            />
          </div>
          <div>
            <FieldLabel>Database Path</FieldLabel>
            <input
              type="text"
              readOnly
              value={settings?.db_path ?? "—"}
              className={readOnlyClass}
            />
          </div>
          <div>
            <FieldLabel>Cache TTL</FieldLabel>
            <input
              type="text"
              readOnly
              value={settings?.cache_ttl_seconds !== undefined ? String(settings.cache_ttl_seconds) : "—"}
              className={readOnlyClass}
            />
          </div>
        </div>
      </SectionCard>

      {/* Section 3: CORS Settings */}
      <SectionCard title="CORS Settings" subtitle="Cross-origin resource sharing">
        <FieldLabel>Allowed Origins</FieldLabel>
        <input
          type="text"
          readOnly
          value={settings?.cors_origins ?? "—"}
          className={readOnlyClass}
        />
      </SectionCard>

      {/* Section 4: Logging */}
      <SectionCard title="Logging" subtitle="Log level and output settings">
        <FieldLabel>Log Level</FieldLabel>
        <div className="relative">
          <select
            className={inputClass + " appearance-none pr-8 cursor-pointer"}
            defaultValue={settings?.log_level ?? "info"}
          >
            <option value="info">info</option>
            <option value="debug">debug</option>
            <option value="warn">warn</option>
            <option value="error">error</option>
          </select>
          <div className="pointer-events-none absolute right-3 top-1/2 -translate-y-1/2 text-[var(--color-text-muted)]">
            <svg width="10" height="6" viewBox="0 0 10 6" fill="none">
              <path d="M1 1l4 4 4-4" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round"/>
            </svg>
          </div>
        </div>
      </SectionCard>

      {/* Save button */}
      <Button onClick={handleSave} disabled={!tokenInput.trim()} className="mt-2 w-fit">
        {saved ? "Saved!" : "Save Settings"}
      </Button>
    </div>
  );
}
