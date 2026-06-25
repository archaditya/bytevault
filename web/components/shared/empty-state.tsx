import { LucideIcon } from "lucide-react";
import { cn } from "@/lib/utils";

interface EmptyStateProps {
  icon: LucideIcon;
  title: string;
  description: string;
  action?: React.ReactNode;
  className?: string;
}

export function EmptyState({ icon: Icon, title, description, action, className }: EmptyStateProps) {
  return (
    <div className={cn("flex flex-col items-center justify-center gap-3 rounded-md border border-dashed border-border-strong py-16 text-center", className)}>
      <div className="flex h-10 w-10 items-center justify-center rounded-md bg-bg-overlay">
        <Icon className="h-5 w-5 text-ink-faint" strokeWidth={1.5} />
      </div>
      <div className="space-y-1">
        <p className="text-sm font-medium text-ink">{title}</p>
        <p className="max-w-sm text-[13px] text-ink-muted">{description}</p>
      </div>
      {action}
    </div>
  );
}
