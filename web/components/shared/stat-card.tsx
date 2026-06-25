import { LucideIcon, ArrowUpRight, ArrowDownRight } from "lucide-react";
import { cn } from "@/lib/utils";
import { Card } from "@/components/ui/card";

interface StatCardProps {
  label: string;
  value: string;
  icon: LucideIcon;
  trend?: { value: string; direction: "up" | "down"; positive?: boolean };
  accent?: string;
}

export function StatCard({ label, value, icon: Icon, trend, accent = "text-accent" }: StatCardProps) {
  const TrendIcon = trend?.direction === "up" ? ArrowUpRight : ArrowDownRight;
  const trendIsGood = trend?.positive ?? trend?.direction === "up";

  return (
    <Card className="p-4">
      <div className="flex items-start justify-between">
        <p className="label-eyebrow">{label}</p>
        <Icon className={cn("h-4 w-4", accent)} strokeWidth={1.75} />
      </div>
      <p className="mt-3 font-mono text-2xl font-semibold tracking-tight text-ink">{value}</p>
      {trend && (
        <div
          className={cn(
            "mt-1.5 inline-flex items-center gap-0.5 text-[12px] font-medium",
            trendIsGood ? "text-success" : "text-danger"
          )}
        >
          <TrendIcon className="h-3 w-3" />
          {trend.value}
        </div>
      )}
    </Card>
  );
}
