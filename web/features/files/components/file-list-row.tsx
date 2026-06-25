"use client";

import Link from "next/link";
import { Star, Share2 } from "lucide-react";
import { FileRecord } from "@/types";
import { FileKindIcon } from "@/components/shared/file-kind-icon";
import { ProviderTag } from "@/components/shared/provider-tag";
import { formatBytes, formatRelativeTime } from "@/lib/utils";

export function FileListRow({ file }: { file: FileRecord }) {
  return (
    <Link
      href={`/files/${file.id}`}
      className="grid grid-cols-[1fr_110px_140px_110px_40px] items-center gap-4 border-b border-border px-4 py-3 text-[13px] transition-colors hover:bg-bg-overlay/60 last:border-b-0"
    >
      <div className="flex min-w-0 items-center gap-3">
        <div style={{ color: file.thumbnailColor }}>
          <FileKindIcon kind={file.kind} />
        </div>
        <span className="truncate font-medium text-ink">{file.name}</span>
        {file.starred && <Star className="h-3 w-3 shrink-0 fill-live text-live" />}
      </div>
      <span className="font-mono text-ink-muted">{formatBytes(file.sizeBytes)}</span>
      <ProviderTag providerId={file.providerId} />
      <span className="text-ink-muted">{formatRelativeTime(file.uploadedAt)}</span>
      <div className="flex justify-end">
        {file.shared && <Share2 className="h-3.5 w-3.5 text-info" />}
      </div>
    </Link>
  );
}
