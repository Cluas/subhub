import { Plus } from "lucide-react";
import { cn } from "@/lib/utils";

export interface FABProps {
  onClick: () => void;
  "aria-label"?: string;
  className?: string;
}

export function FAB({
  onClick,
  "aria-label": ariaLabel = "Create new",
  className,
}: FABProps) {
  return (
    <button
      type="button"
      aria-label={ariaLabel}
      onClick={onClick}
      className={cn(
        // 48px amber circle
        "fixed bottom-6 right-6 z-50",
        "flex h-12 w-12 items-center justify-center rounded-full",
        "bg-[var(--color-primary)] text-[var(--color-bg)]",
        "shadow-lg hover:opacity-90 active:opacity-80",
        "transition-opacity duration-150",
        // Mobile only — hide on desktop
        "lg:hidden",
        className
      )}
    >
      <Plus className="h-6 w-6" strokeWidth={2.5} />
    </button>
  );
}
