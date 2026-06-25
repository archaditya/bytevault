"use client";

import { HardDrive, Files, ArrowLeftRight, Activity } from "lucide-react";
import { StatCard } from "@/components/shared/stat-card";
import { formatBytes } from "@/lib/utils";
import { files } from "@/lib/mock";
import { storageProviders } from "@/lib/mock";
import { transferStats } from "@/lib/mock";

export function DashboardStats() {
  const totalUsed = storageProviders.reduce((sum, p) => sum + p.usedBytes, 0);

  return (
    <div className="grid grid-cols-2 gap-3 lg:grid-cols-4">
      <StatCard
        label="Storage used"
        value={formatBytes(totalUsed)}
        icon={HardDrive}
        trend={{ value: "+4.2% this week", direction: "up" }}
      />
      <StatCard
        label="Total files"
        value={files.length.toLocaleString()}
        icon={Files}
        trend={{ value: "+12 today", direction: "up" }}
      />
      <StatCard
        label="Transfers"
        value={String(transferStats.active + transferStats.completed + transferStats.failed + transferStats.queued + transferStats.paused)}
        icon={ArrowLeftRight}
        trend={{ value: "+38 today", direction: "up" }}
      />
      <StatCard
        label="Active sessions"
        value={String(transferStats.active)}
        icon={Activity}
        accent="text-live"
      />
    </div>
  );
}
