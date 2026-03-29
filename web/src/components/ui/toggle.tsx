import { cn } from "@/lib/utils";

export interface ToggleProps {
  checked: boolean;
  onChange: (checked: boolean) => void;
  disabled?: boolean;
  id?: string;
  className?: string;
  "aria-label"?: string;
}

export function Toggle({
  checked,
  onChange,
  disabled = false,
  id,
  className,
  "aria-label": ariaLabel,
}: ToggleProps) {
  return (
    <button
      type="button"
      role="switch"
      aria-checked={checked}
      aria-label={ariaLabel}
      id={id}
      disabled={disabled}
      onClick={() => onChange(!checked)}
      className={cn(
        // Pill container: 44px wide × 24px tall
        "relative inline-flex h-6 w-11 shrink-0 cursor-pointer items-center rounded-full",
        "border-2 border-transparent",
        "transition-colors duration-150 ease-in-out",
        "focus:outline-none focus-visible:ring-2 focus-visible:ring-[var(--color-primary)] focus-visible:ring-offset-2 focus-visible:ring-offset-[var(--color-bg)]",
        "disabled:cursor-not-allowed disabled:opacity-50",
        // Track color: amber when on, muted when off
        checked ? "bg-[var(--color-primary)]" : "bg-[var(--color-bg-accent)]",
        className
      )}
    >
      {/* Knob: 20px white circle */}
      <span
        aria-hidden="true"
        className={cn(
          "pointer-events-none inline-block h-5 w-5 rounded-full bg-white shadow-sm",
          "transform transition-transform duration-150 ease-in-out",
          // Slide right when on, left when off
          checked ? "translate-x-5" : "translate-x-0"
        )}
      />
    </button>
  );
}
