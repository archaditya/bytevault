import { Metadata } from "next";
import { storageProviders } from "@/lib/mock";
import { Card, CardContent } from "@/components/ui/card";
import { ProviderStatusBadge } from "@/components/shared/status-badge";
import { Switch } from "@/components/ui/switch";
import { Badge } from "@/components/ui/badge";
import { formatBytes } from "@/lib/utils";

export const metadata: Metadata = {
  title: "Manage storage providers — ByteVault",
};

export default function ProviderManagementPage() {
  return (
    <div className="flex flex-col gap-4">
      <p className="max-w-lg text-[13px] text-ink-muted">
        Enable or disable providers, and review uptime over the past 7 days.
      </p>

      {storageProviders.map((provider) => (
        <Card key={provider.id}>
          <CardContent className="flex flex-col gap-4">
            <div className="flex flex-wrap items-center justify-between gap-3">
              <div className="flex items-center gap-3">
                <div>
                  <div className="flex items-center gap-2">
                    <p className="text-[14px] font-semibold text-ink">{provider.name}</p>
                    {provider.isDefault && <Badge variant="default">Default</Badge>}
                  </div>
                  <p className="text-[12px] text-ink-muted">{provider.vendor} · {provider.region}</p>
                </div>
              </div>
              <div className="flex items-center gap-4">
                <ProviderStatusBadge status={provider.status} />
                <div className="flex items-center gap-2">
                  <span className="text-[12px] text-ink-muted">Enabled</span>
                  <Switch defaultChecked={provider.status !== "down"} />
                </div>
              </div>
            </div>

            <div className="grid grid-cols-2 gap-4 border-t border-border pt-4 sm:grid-cols-5">
              <Stat label="Latency" value={`${provider.latencyMs}ms`} />
              <Stat label="Availability" value={`${provider.availabilityPct}%`} />
              <Stat label="Used" value={formatBytes(provider.usedBytes)} />
              <Stat label="Capacity" value={formatBytes(provider.capacityBytes)} />
              <Stat label="Cost/GB" value={provider.costPerGbCents === 0 ? "Free" : `$${(provider.costPerGbCents / 100).toFixed(3)}`} />
            </div>

            <div>
              <p className="mb-2 text-[12px] text-ink-muted">7-day uptime</p>
              <div className="flex items-end gap-1.5">
                {provider.uptimeHistory.map((d) => (
                  <div key={d.day} className="flex flex-1 flex-col items-center gap-1">
                    <div className="flex h-12 w-full items-end overflow-hidden rounded-sm bg-bg-overlay">
                      <div
                        className="w-full bg-accent"
                        style={{ height: `${Math.max(4, ((d.uptimePct - 95) / 5) * 100)}%` }}
                      />
                    </div>
                    <span className="text-[10px] text-ink-faint">{d.day}</span>
                  </div>
                ))}
              </div>
            </div>
          </CardContent>
        </Card>
      ))}
    </div>
  );
}

function Stat({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <p className="text-[11px] uppercase tracking-wide text-ink-faint">{label}</p>
      <p className="mt-1 font-mono text-[14px] font-medium text-ink">{value}</p>
    </div>
  );
}
