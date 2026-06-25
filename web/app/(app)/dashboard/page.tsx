import { Metadata } from "next";
import { DashboardStats } from "@/features/dashboard/components/dashboard-stats";
import { DashboardCharts } from "@/features/dashboard/components/dashboard-charts";
import { RecentTransfersWidget } from "@/features/dashboard/components/recent-transfers-widget";
import { UsageWidget } from "@/features/storage/components/usage-widget";
import { Card, CardHeader, CardTitle } from "@/components/ui/card";
import { storageProviders } from "@/lib/mock";

export const metadata: Metadata = {
  title: "Dashboard — ByteVault",
};

export default function DashboardPage() {
  return (
    <div className="flex flex-col gap-6">
      <DashboardStats />
      <DashboardCharts />
      <div className="grid grid-cols-1 gap-4 lg:grid-cols-3">
        <div className="lg:col-span-2">
          <RecentTransfersWidget />
        </div>
        <Card>
          <CardHeader className="pb-3">
            <CardTitle>Storage usage</CardTitle>
          </CardHeader>
          <div className="p-4 pt-0">
            <UsageWidget providers={storageProviders} />
          </div>
        </Card>
      </div>
    </div>
  );
}
