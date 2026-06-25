import { TransferLogEntry } from "@/types";
import { cn, formatDateTime } from "@/lib/utils";
import { Info, AlertTriangle, XCircle } from "lucide-react";

const levelConfig = {
  info: { icon: Info, color: "text-info" },
  warn: { icon: AlertTriangle, color: "text-live" },
  error: { icon: XCircle, color: "text-danger" },
};

export function TransferTimeline({ logs }: { logs: TransferLogEntry[] }) {
  return (
    <div className="flex flex-col">
      {logs.map((log, i) => {
        const config = levelConfig[log.level];
        const Icon = config.icon;
        return (
          <div key={log.id} className="relative flex gap-3 pb-4 last:pb-0">
            {i < logs.length - 1 && (
              <span className="absolute left-[9px] top-5 h-full w-px bg-border" />
            )}
            <Icon className={cn("z-10 mt-0.5 h-[18px] w-[18px] shrink-0 rounded-full bg-bg-surface", config.color)} />
            <div className="flex-1 pb-0.5">
              <p className="text-[13px] text-ink">{log.message}</p>
              <p className="mt-0.5 font-mono text-[11px] text-ink-faint">{formatDateTime(log.timestamp)}</p>
            </div>
          </div>
        );
      })}
    </div>
  );
}
