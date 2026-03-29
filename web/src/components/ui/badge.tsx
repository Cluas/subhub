import * as React from "react";
import { cn } from "@/lib/utils";

type BadgeVariant = "default" | "success" | "warning" | "danger" | "type";

const variantClasses: Record<BadgeVariant, string> = {
  default: "bg-[var(--color-bg-accent)] text-[var(--color-text-secondary)]",
  success: "bg-[var(--color-success-bg)] text-[var(--color-success)]",
  warning: "bg-[var(--color-warning-bg)] text-[var(--color-warning)]",
  danger:  "bg-[var(--color-danger-bg)] text-[var(--color-danger)]",
  type:    "bg-[var(--color-primary-bg)] text-[var(--color-primary-light)]",
};

export interface BadgeProps extends React.HTMLAttributes<HTMLSpanElement> {
  variant?: BadgeVariant;
}

export function Badge({ className, variant = "default", style, ...props }: BadgeProps) {
  return (
    <span
      className={cn(
        "inline-flex items-center rounded-full px-2 py-0.5 text-[11px] font-semibold leading-none",
        variantClasses[variant],
        className
      )}
      style={style}
      {...props}
    />
  );
}
