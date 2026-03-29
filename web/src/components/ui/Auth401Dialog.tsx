import { useCallback, useEffect, useRef, useState } from "react";
import { X } from "lucide-react";
import { useAuth } from "@/context/AuthContext";
import { useToast } from "@/components/ui/toast";
import { registerTokenPrompt, unregisterTokenPrompt } from "@/lib/api";
import { Button } from "./button";

// ── Auth401Dialog ─────────────────────────────────────────────────────────────
// Mounts alongside the router tree. When apiFetch receives a 401, it calls the
// registered prompt callback which opens this dialog. The user enters a new
// token; on Save the dialog persists it, fires a success toast, and resolves
// the promise with the token so apiFetch can retry the original request.

interface DialogState {
  open: boolean;
  resolve: ((token: string | null) => void) | null;
}

export function Auth401Dialog() {
  const [state, setState] = useState<DialogState>({ open: false, resolve: null });
  const [inputVal, setInputVal] = useState("");
  const { setToken } = useAuth();
  const { showSuccess } = useToast();
  const inputRef = useRef<HTMLInputElement>(null);

  // Build the prompt function once (stable reference for effect deps).
  const prompt = useCallback((): Promise<string | null> => {
    return new Promise((resolve) => {
      setState({ open: true, resolve });
      setInputVal("");
    });
  }, []);

  // Register / unregister with the api module.
  useEffect(() => {
    registerTokenPrompt(prompt);
    return () => unregisterTokenPrompt();
  }, [prompt]);

  // Auto-focus the input whenever the dialog opens.
  useEffect(() => {
    if (state.open) {
      setTimeout(() => inputRef.current?.focus(), 0);
    }
  }, [state.open]);

  // Close on Escape.
  useEffect(() => {
    if (!state.open) return;
    function handleKey(e: KeyboardEvent) {
      if (e.key === "Escape") handleCancel();
    }
    document.addEventListener("keydown", handleKey);
    return () => document.removeEventListener("keydown", handleKey);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [state.open]);

  const handleSave = () => {
    const t = inputVal.trim();
    if (t) {
      setToken(t);
      showSuccess("Token saved — retrying…");
      state.resolve?.(t);
    } else {
      // Empty input → resolve null (no retry, no setToken)
      state.resolve?.(null);
    }
    setState({ open: false, resolve: null });
  };

  const handleCancel = () => {
    state.resolve?.(null);
    setState({ open: false, resolve: null });
  };

  if (!state.open) return null;

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center"
      role="dialog"
      aria-modal="true"
      aria-labelledby="auth401-dialog-title"
    >
      {/* Backdrop */}
      <div className="absolute inset-0 bg-black/50" onClick={handleCancel} />

      {/* Panel */}
      <div
        className="relative rounded-xl border shadow-xl p-6 flex flex-col gap-4 max-w-md w-full mx-4"
        style={{
          backgroundColor: "var(--color-card)",
          borderColor: "var(--color-border)",
        }}
      >
        {/* Header */}
        <div className="flex items-start justify-between gap-3">
          <h2
            id="auth401-dialog-title"
            className="font-semibold text-base leading-snug"
            style={{ color: "var(--color-foreground)" }}
          >
            API Token Required
          </h2>
          <button
            onClick={handleCancel}
            className="shrink-0 rounded p-1 hover:bg-[var(--color-muted)] transition-colors"
            aria-label="Close"
          >
            <X className="h-4 w-4" style={{ color: "var(--color-muted-foreground)" }} />
          </button>
        </div>

        {/* Description */}
        <p
          className="text-sm leading-relaxed"
          style={{ color: "var(--color-muted-foreground)" }}
        >
          The request was rejected (401 Unauthorized). Enter your API token below to
          authenticate and retry.
        </p>

        {/* Token input */}
        <input
          ref={inputRef}
          type="password"
          value={inputVal}
          onChange={(e) => setInputVal(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === "Enter") handleSave();
          }}
          placeholder="Paste your API token…"
          className="w-full rounded-md border px-3 py-2 text-sm outline-none focus:ring-2 focus:ring-[var(--color-ring)]"
          style={{
            backgroundColor: "var(--color-background)",
            borderColor: "var(--color-input)",
            color: "var(--color-foreground)",
          }}
          aria-label="API Token"
        />

        {/* Actions */}
        <div className="flex gap-2 justify-end">
          <Button type="button" variant="outline" onClick={handleCancel}>
            Cancel
          </Button>
          <Button type="button" onClick={handleSave}>
            Save Token
          </Button>
        </div>
      </div>
    </div>
  );
}
