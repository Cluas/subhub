import * as React from "react";
import { cn } from "@/lib/utils";
import { Card, CardContent, CardFooter } from "@/components/ui/card";

interface DataCardProps extends Omit<React.ComponentProps<"div">, "title"> {
  /** Content for the top title row */
  title: React.ReactNode;
  /** Optional subtitle / metadata line below title */
  subtitle?: React.ReactNode;
}

interface DataCardFieldProps {
  label: string;
  value: React.ReactNode;
  className?: string;
}

type DataCardActionsProps = React.ComponentProps<"div">;

function DataCard({ title, subtitle, children, className, ...props }: DataCardProps) {
  return (
    <Card className={cn("gap-0 py-4", className)} {...props}>
      <CardContent className="pb-3">
        {/* Title row */}
        <div className="flex items-start justify-between gap-2">
          <div className="text-[14px] font-semibold text-[var(--color-text-primary)] leading-snug">
            {title}
          </div>
        </div>
        {/* Subtitle / metadata row */}
        {subtitle && (
          <div className="mt-1 text-[12px] text-[var(--color-text-muted)]">
            {subtitle}
          </div>
        )}
        {/* Children (badges, additional fields) */}
        {children}
      </CardContent>
    </Card>
  );
}

function DataCardField({ label, value, className }: DataCardFieldProps) {
  return (
    <div className={cn("mt-2 flex items-center justify-between", className)}>
      <span className="text-[11px] font-medium uppercase tracking-wide text-[var(--color-text-muted)]">
        {label}
      </span>
      <span className="text-[13px] text-[var(--color-text-secondary)]">
        {value}
      </span>
    </div>
  );
}

function DataCardActions({ children, className, ...props }: DataCardActionsProps) {
  return (
    <CardFooter className={cn("mt-3 gap-2 border-t border-[var(--color-border)]/50 pt-3", className)} {...props}>
      {children}
    </CardFooter>
  );
}

export { DataCard, DataCardField, DataCardActions };
