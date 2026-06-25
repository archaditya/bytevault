"use client";

import { Card, CardHeader, CardTitle, CardContent } from "@/components/ui/card";
import { AnalyticsChart } from "@/features/analytics/components/analytics-chart";
import { useAnalyticsHistory } from "@/services";
import { Skeleton } from "@/components/ui/skeleton";

export function DashboardCharts() {
  const { data, isLoading } = useAnalyticsHistory();

  if (isLoading || !data) {
    return (
      <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
        <Skeleton className="h-[320px]" />
        <Skeleton className="h-[320px]" />
      </div>
    );
  }

  return (
    <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
      <Card>
        <CardHeader>
          <CardTitle>Upload &amp; download volume</CardTitle>
        </CardHeader>
        <CardContent>
          <AnalyticsChart data={data} metric="transfers" />
        </CardContent>
      </Card>
      <Card>
        <CardHeader>
          <CardTitle>Transfer success rate</CardTitle>
        </CardHeader>
        <CardContent>
          <AnalyticsChart data={data} metric="successRate" />
        </CardContent>
      </Card>
    </div>
  );
}
