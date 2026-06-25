import { StorageProvider } from "@/types";
import { Card } from "@/components/ui/card";
import { ProviderStatusBadge } from "@/components/shared/status-badge";
import { Progress } from "@/components/ui/progress";
import { Badge } from "@/components/ui/badge";
import { formatBytes } from "@/lib/utils";
import { Zap, MapPin } from "lucide-react";
import Link from "next/link";

export function ProviderCard({ provider }: { provider: StorageProvider }) {
  const usagePct = Math.round((provider.usedBytes / provider.capacityBytes) * 100);

  return (
    <Link href={`/storage/providers?provider=${provider.id}`}>
      <Card className="flex flex-col gap-4 p-5 transition-colors hover:border-border-strong">
        <div className="flex items-start justify-between">
          <div>
            <div className="flex items-center gap-2">
              <p className="text-[14px] font-semibold text-ink">{provider.name}</p>
              {provider.isPrimary && <Badge variant="default">Primary</Badge>}
            </div>
            <p className="mt-0.5 flex items-center gap-1 text-[12px] text-ink-muted">
              <MapPin className="h-3 w-3" /> {provider.region}
            </p>
          </div>
          <ProviderStatusBadge status={provider.status} />
        </div>

        <div>
          <div className="flex items-center justify-between text-[12px] text-ink-muted">
            <span>Storage used</span>
            <span className="font-mono text-ink">
              {formatBytes(provider.usedBytes)} / {formatBytes(provider.capacityBytes)}
            </span>
          </div>
          <Progress value={usagePct} className="mt-1.5" />
        </div>

        <div className="grid grid-cols-3 gap-3 border-t border-border pt-3 text-[12px]">
          <div>
            <p className="flex items-center gap-1 text-ink-faint">
              <Zap className="h-3 w-3" /> Latency
            </p>
            <p className="mt-0.5 font-mono font-medium text-ink">{provider.latencyMs}ms</p>
          </div>
          <div>
            <p className="text-ink-faint">Availability</p>
            <p className="mt-0.5 font-mono font-medium text-ink">{provider.availabilityPct}%</p>
          </div>
          <div>
            <p className="text-ink-faint">Files</p>
            <p className="mt-0.5 font-mono font-medium text-ink">{provider.fileCount.toLocaleString()}</p>
          </div>
        </div>
      </Card>
    </Link>
  );
}
