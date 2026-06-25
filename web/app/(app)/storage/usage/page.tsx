import { Metadata } from "next";
import { storageProviders, analyticsHistory } from "@/lib/mock";
import { Card, CardHeader, CardTitle, CardContent } from "@/components/ui/card";
import { UsageWidget } from "@/features/storage/components/usage-widget";
import { AnalyticsChart } from "@/features/analytics/components/analytics-chart";
import { formatBytes } from "@/lib/utils";

export const metadata: Metadata = {
  title: "Storage usage — ByteVault",
};

export default function StorageUsagePage() {
  const totalUsed = storageProviders.reduce((sum, p) => sum + p.usedBytes, 0);
  const totalCapacity = storageProviders.reduce((sum, p) => sum + p.capacityBytes, 0);

  return (
    <div className="flex flex-col gap-4">
      <div className="grid grid-cols-1 gap-4 lg:grid-cols-3">
        <Card className="lg:col-span-2">
          <CardHeader>
            <CardTitle>Storage growth (30 days)</CardTitle>
          </CardHeader>
          <CardContent>
            <AnalyticsChart data={analyticsHistory} metric="bytes" />
          </CardContent>
        </Card>
        <Card>
          <CardHeader>
            <CardTitle>By provider</CardTitle>
          </CardHeader>
          <CardContent>
            <UsageWidget providers={storageProviders} />
          </CardContent>
        </Card>
      </div>

      <Card>
        <CardContent className="flex flex-wrap items-center justify-between gap-4">
          <div>
            <p className="label-eyebrow">Total capacity used</p>
            <p className="mt-1 font-mono text-xl font-semibold text-ink">
              {formatBytes(totalUsed)} <span className="text-ink-faint">/ {formatBytes(totalCapacity)}</span>
            </p>
          </div>
          <p className="text-[13px] text-ink-muted">
            {Math.round((totalUsed / totalCapacity) * 100)}% of total provisioned capacity in use across all providers.
          </p>
        </CardContent>
      </Card>
    </div>
  );
}
