import { Badge } from "@/components/ui/badge";
import { TransferStatus } from "@/types";
import { ProviderStatus } from "@/types";

const transferStatusConfig: Record<TransferStatus, { label: string; variant: "live" | "muted" | "success" | "danger" | "info" }> = {
  active: { label: "Active", variant: "live" },
  paused: { label: "Paused", variant: "muted" },
  completed: { label: "Completed", variant: "success" },
  failed: { label: "Failed", variant: "danger" },
  queued: { label: "Queued", variant: "info" },
};

const providerStatusConfig: Record<ProviderStatus, { label: string; variant: "success" | "live" | "danger" }> = {
  healthy: { label: "Healthy", variant: "success" },
  degraded: { label: "Degraded", variant: "live" },
  down: { label: "Down", variant: "danger" },
};

export function TransferStatusBadge({ status }: { status: TransferStatus }) {
  const config = transferStatusConfig[status];
  return (
    <Badge variant={config.variant} dot>
      {config.label}
    </Badge>
  );
}

export function ProviderStatusBadge({ status }: { status: ProviderStatus }) {
  const config = providerStatusConfig[status];
  return (
    <Badge variant={config.variant} dot>
      {config.label}
    </Badge>
  );
}
