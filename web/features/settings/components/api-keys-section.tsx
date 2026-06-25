"use client";

import { useApiKeys } from "@/services";
import { Card } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { KeyRound, Plus, Trash2 } from "lucide-react";
import { formatDate, formatRelativeTime } from "@/lib/utils";
import { Skeleton } from "@/components/ui/skeleton";

export function ApiKeysSection() {
  const { data: keys, isLoading } = useApiKeys();

  return (
    <div className="max-w-2xl">
      <div className="mb-4 flex items-center justify-between">
        <p className="text-[13px] text-ink-muted">
          API keys grant programmatic access to transfers and files.
        </p>
        <Button size="sm">
          <Plus className="h-3.5 w-3.5" /> New key
        </Button>
      </div>

      <Card className="divide-y divide-border p-0">
        {isLoading
          ? Array.from({ length: 4 }).map((_, i) => <Skeleton key={i} className="m-3 h-14" />)
          : keys?.map((key) => (
              <div key={key.id} className="flex items-center justify-between gap-4 p-4">
                <div className="flex items-center gap-3">
                  <div className="flex h-8 w-8 items-center justify-center rounded-sm bg-accent/10 text-accent-bright">
                    <KeyRound className="h-4 w-4" />
                  </div>
                  <div>
                    <p className="text-[13px] font-medium text-ink">{key.label}</p>
                    <p className="font-mono text-[12px] text-ink-muted">{key.prefix}••••••••••••</p>
                    <div className="mt-1 flex flex-wrap gap-1">
                      {key.scopes.map((scope) => (
                        <Badge key={scope} variant="muted">{scope}</Badge>
                      ))}
                    </div>
                  </div>
                </div>
                <div className="flex items-center gap-3">
                  <p className="text-right text-[12px] text-ink-faint">
                    {key.lastUsedAt ? `Used ${formatRelativeTime(key.lastUsedAt)}` : "Never used"}
                    <br />
                    Created {formatDate(key.createdAt)}
                  </p>
                  <Button size="icon" variant="ghost" aria-label="Revoke key">
                    <Trash2 className="h-4 w-4 text-danger" />
                  </Button>
                </div>
              </div>
            ))}
      </Card>
    </div>
  );
}
