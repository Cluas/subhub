import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useRef,
  useState,
  type ReactNode,
} from "react";
import { X } from "lucide-react";

// ── Types ─────────────────────────────────────────────────────────────────────

type ToastVariant = "success" | "error" | "info";

interface Toast {
  id: number;
  message: string;
  variant: ToastVariant;
  /** Errors require manual close; others auto-dismiss after 3s */
  autoDismiss: boolean;
}

interface ToastContextValue {
  showToast: (message: string, variant?: ToastVariant) => void;
  showSuccess: (message: string) => void;
  showError: (message: string) => void;
}

// ── Context ───────────────────────────────────────────────────────────────────

const ToastContext = createContext<ToastContextValue | null>(null);

let nextId = 1;

export function ToastProvider({ children }: { children: ReactNode }) {
  const [toasts, setToasts] = useState<Toast[]>([]);

  const dismiss = useCallback((id: number) => {
    setToasts((prev) => prev.filter((t) => t.id !== id));
  }, []);

  const showToast = useCallback((message: string, variant: ToastVariant = "info") => {
    const id = nextId++;
    const autoDismiss = variant !== "error";
    setToasts((prev) => [...prev, { id, message, variant, autoDismiss }]);
    if (autoDismiss) {
      setTimeout(() => dismiss(id), 3000);
    }
  }, [dismiss]);

  const showSuccess = useCallback((message: string) => showToast(message, "success"), [showToast]);
  const showError = useCallback((message: string) => showToast(message, "error"), [showToast]);

  return (
    <ToastContext.Provider value={{ showToast, showSuccess, showError }}>
      {children}
      <ToastContainer toasts={toasts} onDismiss={dismiss} />
    </ToastContext.Provider>
  );
}

// eslint-disable-next-line react-refresh/only-export-components
export function useToast(): ToastContextValue {
  const ctx = useContext(ToastContext);
  if (!ctx) throw new Error("useToast must be used inside <ToastProvider>");
  return ctx;
}

// ── ToastItem ─────────────────────────────────────────────────────────────────

function ToastItem({ toast, onDismiss }: { toast: Toast; onDismiss: () => void }) {
  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  // Progress bar for auto-dismiss toasts
  useEffect(() => {
    const ref = timerRef.current;
    return () => {
      if (ref) clearTimeout(ref);
    };
  }, []);

  const bgColor =
    toast.variant === "success"
      ? "hsl(142 76% 28%)"
      : toast.variant === "error"
      ? "hsl(0 72% 40%)"
      : "hsl(221 83% 40%)";

  return (
    <div
      role="alert"
      className="flex items-start gap-3 rounded-lg px-4 py-3 shadow-lg text-sm max-w-sm w-full animate-in slide-in-from-right-full duration-200"
      style={{
        backgroundColor: bgColor,
        color: "hsl(0 0% 98%)",
      }}
    >
      <span className="flex-1 leading-snug">{toast.message}</span>
      {(!toast.autoDismiss || toast.variant === "error") && (
        <button
          onClick={onDismiss}
          className="shrink-0 opacity-80 hover:opacity-100 transition-opacity mt-0.5"
          aria-label="Dismiss"
        >
          <X className="h-4 w-4" />
        </button>
      )}
    </div>
  );
}

// ── ToastContainer ────────────────────────────────────────────────────────────

function ToastContainer({
  toasts,
  onDismiss,
}: {
  toasts: Toast[];
  onDismiss: (id: number) => void;
}) {
  if (toasts.length === 0) return null;

  return (
    <div
      className="fixed top-4 right-4 flex flex-col gap-2 z-50 pointer-events-none"
      aria-live="polite"
    >
      {toasts.map((t) => (
        <div key={t.id} className="pointer-events-auto">
          <ToastItem toast={t} onDismiss={() => onDismiss(t.id)} />
        </div>
      ))}
    </div>
  );
}
