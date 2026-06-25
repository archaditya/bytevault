import { notFound } from "next/navigation";
import Link from "next/link";
import { ChevronLeft, ArrowUp, ArrowDown, RotateCcw, Pause, Play, X } from "lucide-react";
import { getTransferById } from "@/lib/mock";
import { Card, CardHeader, CardTitle, CardContent } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { TransferStatusBadge } from "@/components/shared/status-badge";
import { TransferProgress } from "@/components/shared/transfer-progress";
import { ProviderTag } from "@/components/shared/provider-tag";
import { ChunkVisualizer } from "@/components/shared/chunk-visualizer";
import { TransferTimeline } from "@/features/transfers/components/transfer-timeline";
import { SpeedGraph } from "@/features/transfers/components/speed-graph";
import { formatBytes, formatDateTime, truncateMiddle } from "@/lib/utils";

export function generateMetadata({ params }: { params: { id: string } }) {
  const transfer = getTransferById(params.id);
  return { title: transfer ? `${transfer.fileName} transfer — ByteVault` : "Transfer not found — ByteVault" };
}

export default function TransferDetailsPage({ params }: { params: { id: string } }) {
  const transfer = getTransferById(params.id);
  if (!transfer) return notFound();

  const DirectionIcon = transfer.direction === "upload" ? ArrowUp : ArrowDown;
  const completedChunks = transfer.chunks.filter((c) => c.status === "complete").length;

  return (
    <div className="flex flex-col gap-6">
      <Link href="/transfers" className="inline-flex items-center gap-1 text-[13px] text-ink-muted hover:text-ink">
        <ChevronLeft className="h-3.5 w-3.5" /> Back to transfers
      </Link>

      <div className="flex flex-wrap items-start justify-between gap-4">
        <div className="flex items-start gap-3">
          <div className="mt-0.5 flex h-10 w-10 items-center justify-center rounded-md bg-accent/10 text-accent-bright">
            <DirectionIcon className="h-5 w-5" />
          </div>
          <div>
            <h2 className="text-lg font-semibold text-ink">{truncateMiddle(transfer.fileName, 48)}</h2>
            <div className="mt-1 flex items-center gap-2 text-[13px] text-ink-muted">
              <ProviderTag providerId={transfer.providerId} />
              <span>·</span>
              <span className="capitalize">{transfer.direction}</span>
              <span>·</span>
              <span>Started by {transfer.initiatedBy}</span>
            </div>
          </div>
        </div>
        <div className="flex items-center gap-2">
          <TransferStatusBadge status={transfer.status} />
          {transfer.status === "active" && (
            <Button size="sm" variant="secondary">
              <Pause className="h-3.5 w-3.5" /> Pause
            </Button>
          )}
          {transfer.status === "paused" && (
            <Button size="sm" variant="secondary">
              <Play className="h-3.5 w-3.5" /> Resume
            </Button>
          )}
          {transfer.status === "failed" && (
            <Button size="sm" variant="secondary">
              <RotateCcw className="h-3.5 w-3.5" /> Retry
            </Button>
          )}
          {(transfer.status === "active" || transfer.status === "queued") && (
            <Button size="sm" variant="danger">
              <X className="h-3.5 w-3.5" /> Cancel
            </Button>
          )}
        </div>
      </div>

      <Card>
        <CardContent>
          <TransferProgress
            status={transfer.status}
            transferredBytes={transfer.transferredBytes}
            sizeBytes={transfer.sizeBytes}
            speedBytesPerSecond={transfer.speedBytesPerSecond}
            etaSeconds={transfer.etaSeconds}
          />
        </CardContent>
      </Card>

      <div className="grid grid-cols-2 gap-3 sm:grid-cols-4">
        <MiniStat label="Total size" value={formatBytes(transfer.sizeBytes)} />
        <MiniStat label="Chunks" value={`${completedChunks}/${transfer.totalChunks}`} />
        <MiniStat label="Retries" value={String(transfer.retryCount)} highlight={transfer.retryCount > 0} />
        <MiniStat label="Started" value={formatDateTime(transfer.startedAt)} />
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Chunk progress</CardTitle>
        </CardHeader>
        <CardContent>
          <ChunkVisualizer chunks={transfer.chunks} totalChunks={transfer.totalChunks} />
        </CardContent>
      </Card>

      <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
        <Card>
          <CardHeader>
            <CardTitle>Speed graph</CardTitle>
          </CardHeader>
          <CardContent>
            <SpeedGraph data={transfer.speedHistory} />
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Transfer timeline</CardTitle>
          </CardHeader>
          <CardContent>
            <TransferTimeline logs={transfer.logs} />
          </CardContent>
        </Card>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Transfer logs</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="overflow-hidden rounded-sm border border-border bg-bg">
            <div className="max-h-72 overflow-y-auto p-3 font-mono text-[12px] leading-relaxed">
              {[...transfer.logs].reverse().map((log) => (
                <div key={log.id} className="flex gap-3 py-0.5">
                  <span className="shrink-0 text-ink-faint">{formatDateTime(log.timestamp)}</span>
                  <span
                    className={
                      log.level === "error"
                        ? "text-danger"
                        : log.level === "warn"
                        ? "text-live"
                        : "text-ink-muted"
                    }
                  >
                    [{log.level.toUpperCase()}]
                  </span>
                  <span className="text-ink">{log.message}</span>
                </div>
              ))}
            </div>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}

function MiniStat({ label, value, highlight }: { label: string; value: string; highlight?: boolean }) {
  return (
    <div className="rounded-md border border-border bg-bg-surface p-3.5">
      <p className="label-eyebrow">{label}</p>
      <p className={`mt-1.5 font-mono text-[15px] font-medium ${highlight ? "text-live" : "text-ink"}`}>{value}</p>
    </div>
  );
}
