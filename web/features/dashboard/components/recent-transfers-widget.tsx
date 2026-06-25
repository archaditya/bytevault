"use client";

import Link from "next/link";
import { useTransfers } from "@/services";
import { Card, CardHeader, CardTitle } from "@/components/ui/card";
import { TransferStatusBadge } from "@/components/shared/status-badge";
import { ProviderTag } from "@/components/shared/provider-tag";
import { formatRelativeTime, truncateMiddle } from "@/lib/utils";
import { Skeleton } from "@/components/ui/skeleton";
import { ArrowRight } from "lucide-react";

export function RecentTransfersWidget() {
  const { data: transfers, isLoading } = useTransfers();
  const recent = transfers?.slice(0, 6) ?? [];

  return (
    <Card>
      <CardHeader className="pb-3">
        <CardTitle>Recent transfers</CardTitle>
        <Link href="/transfers" className="flex items-center gap-1 text-[12px] text-accent-bright hover:underline">
          View all <ArrowRight className="h-3 w-3" />
        </Link>
      </CardHeader>
      <div className="flex flex-col">
        {isLoading
          ? Array.from({ length: 5 }).map((_, i) => <Skeleton key={i} className="m-3 h-12" />)
          : recent.map((t) => (
              <Link
                key={t.id}
                href={`/transfers/${t.id}`}
                className="flex items-center justify-between gap-3 border-t border-border px-4 py-3 text-[13px] transition-colors hover:bg-bg-overlay/60"
              >
                <div className="min-w-0">
                  <p className="truncate font-medium text-ink">{truncateMiddle(t.fileName, 32)}</p>
                  <div className="mt-0.5 flex items-center gap-2 text-[11px] text-ink-muted">
                    <ProviderTag providerId={t.providerId} />
                    <span>·</span>
                    <span>{formatRelativeTime(t.updatedAt)}</span>
                  </div>
                </div>
                <TransferStatusBadge status={t.status} />
              </Link>
            ))}
      </div>
    </Card>
  );
}
