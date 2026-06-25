import { cn } from "@/lib/utils";
import { HTMLAttributes } from "react";

type BadgeVariant = "default" | "live" | "success" | "danger" | "info" | "muted";

interface BadgeProps extends HTMLAttributes<HTMLSpanElement> {
  variant?: BadgeVariant;
  dot?: boolean;
}

const variantClasses: Record<BadgeVariant, string> = {
  default: "bg-accent/10 text-accent-bright border-accent/20",
  live: "bg-live/10 text-live border-live/25",
  success: "bg-success/10 text-success border-success/20",
  danger: "bg-danger/10 text-danger border-danger/20",
  info: "bg-info/10 text-info border-info/20",
  muted: "bg-bg-overlay text-ink-muted border-border-strong",
};

const dotClasses: Record<BadgeVariant, string> = {
  default: "bg-accent",
  live: "bg-live animate-pulse-live",
  success: "bg-success",
  danger: "bg-danger",
  info: "bg-info",
  muted: "bg-ink-faint",
};

export function Badge({ variant = "default", dot = false, className, children, ...props }: BadgeProps) {
  return (
    <span
      className={cn(
        "inline-flex items-center gap-1.5 rounded-full border px-2 py-0.5 text-[11px] font-medium leading-4",
        variantClasses[variant],
        className
      )}
      {...props}
    >
      {dot && <span className={cn("h-1.5 w-1.5 rounded-full", dotClasses[variant])} />}
      {children}
    </span>
  );
}
