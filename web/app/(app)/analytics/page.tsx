import { Metadata } from "next";
import { analyticsHistory, providerComparison } from "@/lib/mock";
import { Card, CardHeader, CardTitle, CardContent } from "@/components/ui/card";
import { AnalyticsChart } from "@/features/analytics/components/analytics-chart";
import { ProviderComparisonChart } from "@/features/analytics/components/provider-comparison-chart";
import { StatCard } from "@/components/shared/stat-card";
import { Upload, Download, TrendingUp, AlertTriangle } from "lucide-react";

export const metadata: Metadata = {
  title: "Analytics — ByteVault",
};

export default function AnalyticsPage() {
  const totalUploads = analyticsHistory.reduce((sum, d) => sum + d.uploads, 0);
  const totalDownloads = analyticsHistory.reduce((sum, d) => sum + d.downloads, 0);
  const avgSuccess = (analyticsHistory.reduce((sum, d) => sum + d.successRate, 0) / analyticsHistory.length).toFixed(1);
  const totalFailed = analyticsHistory.reduce((sum, d) => sum + d.failedTransfers, 0);

  return (
    <div className="flex flex-col gap-6">
      <div className="grid grid-cols-2 gap-3 lg:grid-cols-4">
        <StatCard label="Uploads (30d)" value={totalUploads.toLocaleString()} icon={Upload} />
        <StatCard label="Downloads (30d)" value={totalDownloads.toLocaleString()} icon={Download} />
        <StatCard label="Avg success rate" value={`${avgSuccess}%`} icon={TrendingUp} accent="text-success" />
        <StatCard label="Failed transfers" value={totalFailed.toLocaleString()} icon={AlertTriangle} accent="text-danger" />
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Upload trends</CardTitle>
        </CardHeader>
        <CardContent>
          <AnalyticsChart data={analyticsHistory} metric="transfers" />
        </CardContent>
      </Card>

      <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
        <Card>
          <CardHeader>
            <CardTitle>Storage growth</CardTitle>
          </CardHeader>
          <CardContent>
            <AnalyticsChart data={analyticsHistory} metric="bytes" />
          </CardContent>
        </Card>
        <Card>
          <CardHeader>
            <CardTitle>Transfer performance</CardTitle>
          </CardHeader>
          <CardContent>
            <AnalyticsChart data={analyticsHistory} metric="successRate" />
          </CardContent>
        </Card>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Provider comparison</CardTitle>
        </CardHeader>
        <CardContent>
          <ProviderComparisonChart data={providerComparison} />
          <div className="mt-4 grid grid-cols-1 gap-3 sm:grid-cols-3">
            {providerComparison.map((p) => (
              <div key={p.provider} className="rounded-md border border-border p-3">
                <p className="text-[13px] font-medium text-ink">{p.provider}</p>
                <div className="mt-2 flex flex-col gap-1 text-[12px] text-ink-muted">
                  <span>Success rate: <span className="font-mono text-ink">{p.successRate}%</span></span>
                  <span>Throughput: <span className="font-mono text-ink">{p.throughputMbps} Mbps</span></span>
                  <span>Cost/GB: <span className="font-mono text-ink">{p.costPerGb === 0 ? "Free" : `$${p.costPerGb.toFixed(3)}`}</span></span>
                </div>
              </div>
            ))}
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
