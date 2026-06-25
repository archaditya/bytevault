"use client";

import { ChunkState } from "@/types";
import { cn } from "@/lib/utils";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";

const statusClass: Record<ChunkState["status"], string> = {
  complete: "bg-accent",
  uploading: "bg-live animate-pulse-live",
  retrying: "bg-live/70 animate-chunk-fail",
  failed: "bg-danger",
  pending: "bg-bg-overlay",
};

interface ChunkVisualizerProps {
  chunks: ChunkState[];
  totalChunks: number;
  className?: string;
}

/**
 * Renders one block per chunk. This is ByteVault's signature visualization —
 * a transfer isn't a single bar filling up, it's a grid of discrete units of
 * work, each independently retried, completed, or failed.
 */
export function ChunkVisualizer({ chunks, totalChunks, className }: ChunkVisualizerProps) {
  const overflow = totalChunks - chunks.length;

  return (
    <TooltipProvider delayDuration={150}>
      <div className={cn("flex flex-wrap gap-[3px]", className)}>
        {chunks.map((chunk) => (
          <Tooltip key={chunk.index}>
            <TooltipTrigger asChild>
              <div
                className={cn(
                  "h-2.5 w-2.5 rounded-[2px] transition-colors duration-300",
                  statusClass[chunk.status]
                )}
              />
            </TooltipTrigger>
            <TooltipContent>
              <div className="font-mono text-[11px]">
                <p>chunk #{chunk.index}</p>
                <p className="text-ink-muted capitalize">{chunk.status}</p>
                {chunk.retries > 0 && <p className="text-live">{chunk.retries} retr{chunk.retries === 1 ? "y" : "ies"}</p>}
              </div>
            </TooltipContent>
          </Tooltip>
        ))}
        {overflow > 0 && (
          <span className="ml-1 self-center font-mono text-[11px] text-ink-faint">
            +{overflow} more
          </span>
        )}
      </div>
    </TooltipProvider>
  );
}
