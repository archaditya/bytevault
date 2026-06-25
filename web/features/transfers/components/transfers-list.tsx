"use client";

import { useMemo } from "react";
import { useTransfers, useTransferStats } from "@/services";
import { useTransferStore } from "@/store";
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { TransferCard } from "./transfer-card";
import { EmptyState } from "@/components/shared/empty-state";
import { Skeleton } from "@/components/ui/skeleton";
import { Inbox, ArrowUp, ArrowDown } from "lucide-react";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { TransferTab } from "@/store/transfer.store";

const tabs: { value: TransferTab; label: string }[] = [
  { value: "all", label: "All" },
  { value: "active", label: "Active" },
  { value: "queued", label: "Queued" },
  { value: "paused", label: "Paused" },
  { value: "completed", label: "Completed" },
  { value: "failed", label: "Failed" },
];

export function TransfersList() {
  const { data: transfers, isLoading } = useTransfers();
  const { data: stats } = useTransferStats();
  const { activeTab, setActiveTab, directionFilter, setDirectionFilter } = useTransferStore();

  const filtered = useMemo(() => {
    if (!transfers) return [];
    return transfers.filter((t) => {
      const matchesTab = activeTab === "all" || t.status === activeTab;
      const matchesDirection = directionFilter === "all" || t.direction === directionFilter;
      return matchesTab && matchesDirection;
    });
  }, [transfers, activeTab, directionFilter]);

  return (
    <div className="flex flex-col gap-4">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <Tabs value={activeTab} onValueChange={(v) => setActiveTab(v as TransferTab)}>
          <TabsList>
            {tabs.map((tab) => (
              <TabsTrigger key={tab.value} value={tab.value}>
                {tab.label}
                {stats && tab.value !== "all" && tab.value in stats && (
                  <span className="font-mono text-[10px] text-ink-faint">
                    {(stats as Record<string, number>)[tab.value]}
                  </span>
                )}
              </TabsTrigger>
            ))}
          </TabsList>
        </Tabs>

        <div className="flex items-center rounded-md border border-border-strong bg-bg-surface p-0.5">
          <Button
            size="sm"
            variant="ghost"
            className={cn("h-7 px-2.5", directionFilter === "all" && "bg-bg-overlay text-ink")}
            onClick={() => setDirectionFilter("all")}
          >
            All
          </Button>
          <Button
            size="sm"
            variant="ghost"
            className={cn("h-7 px-2.5", directionFilter === "upload" && "bg-bg-overlay text-ink")}
            onClick={() => setDirectionFilter("upload")}
          >
            <ArrowUp className="h-3 w-3" /> Up
          </Button>
          <Button
            size="sm"
            variant="ghost"
            className={cn("h-7 px-2.5", directionFilter === "download" && "bg-bg-overlay text-ink")}
            onClick={() => setDirectionFilter("download")}
          >
            <ArrowDown className="h-3 w-3" /> Down
          </Button>
        </div>
      </div>

      {isLoading ? (
        <div className="grid grid-cols-1 gap-3 lg:grid-cols-2">
          {Array.from({ length: 6 }).map((_, i) => (
            <Skeleton key={i} className="h-[148px]" />
          ))}
        </div>
      ) : filtered.length === 0 ? (
        <EmptyState
          icon={Inbox}
          title="No transfers here"
          description="Nothing matches this filter right now. New transfers will show up here as they start."
        />
      ) : (
        <div className="grid grid-cols-1 gap-3 lg:grid-cols-2">
          {filtered.map((transfer) => (
            <TransferCard key={transfer.id} transfer={transfer} />
          ))}
        </div>
      )}
    </div>
  );
}
