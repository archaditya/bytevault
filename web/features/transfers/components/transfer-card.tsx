"use client";

import Link from "next/link";
import { ArrowUp, ArrowDown, RotateCcw } from "lucide-react";
import { TransferSession } from "@/types";
import { Card } from "@/components/ui/card";
import { TransferStatusBadge } from "@/components/shared/status-badge";
import { TransferProgress } from "@/components/shared/transfer-progress";
import { ProviderTag } from "@/components/shared/provider-tag";
import { ChunkVisualizer } from "@/components/shared/chunk-visualizer";
import { formatRelativeTime, cn } from "@/lib/utils";

export function TransferCard({ transfer }: { transfer: TransferSession }) {
  const DirectionIcon = transfer.direction === "upload" ? ArrowUp : ArrowDown;

  return (
    <Link href={`/transfers/${transfer.id}`}>
      <Card className="flex flex-col gap-3 p-4 transition-colors hover:border-border-strong">
        <div className="flex items-start justify-between gap-3">
          <div className="flex items-start gap-2.5 min-w-0">
            <div
              className={cn(
                "mt-0.5 flex h-7 w-7 shrink-0 items-center justify-center rounded-sm",
                transfer.direction === "upload" ? "bg-accent/10 text-accent-bright" : "bg-info/10 text-info"
              )}
            >
              <DirectionIcon className="h-3.5 w-3.5" />
            </div>
            <div className="min-w-0">
              <p className="truncate text-[13px] font-medium text-ink">{transfer.fileName}</p>
              <div className="mt-0.5 flex items-center gap-2 text-[12px] text-ink-muted">
                <ProviderTag providerId={transfer.providerId} />
                <span>·</span>
                <span>{formatRelativeTime(transfer.updatedAt)}</span>
              </div>
            </div>
          </div>
          <TransferStatusBadge status={transfer.status} />
        </div>

        <TransferProgress
          status={transfer.status}
          transferredBytes={transfer.transferredBytes}
          sizeBytes={transfer.sizeBytes}
          speedBytesPerSecond={transfer.speedBytesPerSecond}
          etaSeconds={transfer.etaSeconds}
        />

        <ChunkVisualizer chunks={transfer.chunks} totalChunks={transfer.totalChunks} className="max-h-[22px] overflow-hidden" />

        {transfer.retryCount > 0 && (
          <div className="flex items-center gap-1.5 text-[11px] text-live">
            <RotateCcw className="h-3 w-3" />
            {transfer.retryCount} retr{transfer.retryCount === 1 ? "y" : "ies"} on this transfer
          </div>
        )}
      </Card>
    </Link>
  );
}
