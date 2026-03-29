import * as React from "react";
import { cn } from "@/lib/utils";

export type SelectProps = React.ComponentProps<"select">;

const Select = React.forwardRef<HTMLSelectElement, SelectProps>(
  ({ className, children, ...props }, ref) => {
    return (
      <select
        className={cn(
          "flex h-10 w-full appearance-none rounded-[var(--radius)] border border-[var(--color-border)]",
          "bg-[var(--color-bg)] px-3 py-2 text-[13px] text-[var(--color-text-primary)]",
          "focus:outline-none focus:ring-1 focus:ring-[var(--color-primary)] focus:border-[var(--color-primary)]",
          "disabled:cursor-not-allowed disabled:opacity-50",
          "transition-colors",
          className
        )}
        ref={ref}
        {...props}
      >
        {children}
      </select>
    );
  }
);
Select.displayName = "Select";

// Wrapper that adds the dropdown arrow icon via absolute positioning
function SelectWrapper({ className, children, ...props }: React.ComponentProps<"div">) {
  return (
    <div className={cn("relative w-full", className)} {...props}>
      {children}
      <div className="pointer-events-none absolute right-3 top-1/2 -translate-y-1/2">
        <svg
          className="h-4 w-4 text-[var(--color-text-muted)]"
          xmlns="http://www.w3.org/2000/svg"
          viewBox="0 0 20 20"
          fill="currentColor"
        >
          <path
            fillRule="evenodd"
            d="M5.293 7.293a1 1 0 011.414 0L10 10.586l3.293-3.293a1 1 0 111.414 1.414l-4 4a1 1 0 01-1.414 0l-4-4a1 1 0 010-1.414z"
            clipRule="evenodd"
          />
        </svg>
      </div>
    </div>
  );
}

export { Select, SelectWrapper };
