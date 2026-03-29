import { useCallback, useEffect, useRef, type ReactNode } from "react";
import { X } from "lucide-react";
import { Button } from "./button";

// ── ConfirmDialog ─────────────────────────────────────────────────────────────

export interface ConfirmDialogProps {
  open: boolean;
  title: string;
  message: ReactNode;
  confirmLabel?: string;
  cancelLabel?: string;
  onConfirm: () => void;
  onCancel: () => void;
}

export function ConfirmDialog({
  open,
  title,
  message,
  confirmLabel = "Confirm Delete",
  cancelLabel = "Cancel",
  onConfirm,
  onCancel,
}: ConfirmDialogProps) {
  const dialogRef = useRef<HTMLDivElement>(null);

  // Close on Escape
  useEffect(() => {
    if (!open) return;
    function handleKey(e: KeyboardEvent) {
      if (e.key === "Escape") onCancel();
    }
    document.addEventListener("keydown", handleKey);
    return () => document.removeEventListener("keydown", handleKey);
  }, [open, onCancel]);

  // Focus trap — focus the cancel button when dialog opens
  const cancelRef = useCallback((el: HTMLButtonElement | null) => {
    if (el && open) {
      setTimeout(() => el.focus(), 0);
    }
  }, [open]);

  if (!open) return null;

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center"
      role="dialog"
      aria-modal="true"
      aria-labelledby="confirm-dialog-title"
    >
      {/* Backdrop */}
      <div
        className="absolute inset-0 bg-black/50"
        onClick={onCancel}
      />

      {/* Panel */}
      <div
        ref={dialogRef}
        className="relative rounded-xl border shadow-xl p-6 flex flex-col gap-4 max-w-md w-full mx-4"
        style={{
          backgroundColor: "var(--color-card)",
          borderColor: "var(--color-border)",
        }}
      >
        {/* Header */}
        <div className="flex items-start justify-between gap-3">
          <h2
            id="confirm-dialog-title"
            className="font-semibold text-base leading-snug"
            style={{ color: "var(--color-foreground)" }}
          >
            {title}
          </h2>
          <button
            onClick={onCancel}
            className="shrink-0 rounded p-1 hover:bg-[var(--color-muted)] transition-colors"
            aria-label="Close"
          >
            <X className="h-4 w-4" style={{ color: "var(--color-muted-foreground)" }} />
          </button>
        </div>

        {/* Message */}
        <div
          className="text-sm leading-relaxed"
          style={{ color: "var(--color-muted-foreground)" }}
        >
          {message}
        </div>

        {/* Actions */}
        <div className="flex gap-2 justify-end">
          <Button
            ref={cancelRef}
            type="button"
            variant="outline"
            onClick={onCancel}
          >
            {cancelLabel}
          </Button>
          <Button
            type="button"
            variant="destructive"
            onClick={onConfirm}
          >
            {confirmLabel}
          </Button>
        </div>
      </div>
    </div>
  );
}
