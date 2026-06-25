"use client";

import { useState } from "react";
import { Lock, Link2, Copy, Check, Calendar, Download } from "lucide-react";
import { SharedLink } from "@/types";
import { Card } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { formatDate } from "@/lib/utils";

export function ShareLinkCard({ link }: { link: SharedLink }) {
  const [copied, setCopied] = useState(false);

  const handleCopy = () => {
    navigator.clipboard?.writeText(link.url).catch(() => {});
    setCopied(true);
    setTimeout(() => setCopied(false), 1800);
  };

  return (
    <Card className="flex flex-col gap-3 p-4">
      <div className="flex items-start justify-between gap-3">
        <div className="flex min-w-0 items-center gap-2.5">
          <div className="flex h-8 w-8 shrink-0 items-center justify-center rounded-sm bg-accent/10 text-accent-bright">
            <Link2 className="h-4 w-4" />
          </div>
          <div className="min-w-0">
            <p className="truncate text-[13px] font-medium text-ink">{link.fileName}</p>
            <p className="truncate font-mono text-[12px] text-ink-muted">{link.url}</p>
          </div>
        </div>
        <Badge variant={link.active ? "success" : "muted"} dot>
          {link.active ? "Active" : "Disabled"}
        </Badge>
      </div>

      <div className="flex flex-wrap items-center gap-2">
        {link.passwordProtected && (
          <Badge variant="muted">
            <Lock className="h-2.5 w-2.5" /> Password
          </Badge>
        )}
        {link.expiresAt && (
          <Badge variant="muted">
            <Calendar className="h-2.5 w-2.5" /> Expires {formatDate(link.expiresAt)}
          </Badge>
        )}
        {link.downloadLimit && (
          <Badge variant="muted">
            <Download className="h-2.5 w-2.5" /> Limit {link.downloadCount}/{link.downloadLimit}
          </Badge>
        )}
      </div>

      <div className="flex items-center justify-between border-t border-border pt-3">
        <div className="flex gap-4 font-mono text-[12px] text-ink-muted">
          <span>{link.views} views</span>
          <span>{link.downloadCount} downloads</span>
        </div>
        <Button size="sm" variant="secondary" onClick={handleCopy}>
          {copied ? <Check className="h-3.5 w-3.5 text-success" /> : <Copy className="h-3.5 w-3.5" />}
          {copied ? "Copied" : "Copy link"}
        </Button>
      </div>
    </Card>
  );
}
