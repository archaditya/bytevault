"use client";

import Link from "next/link";
import { Star, Share2, MoreVertical, Download } from "lucide-react";
import { FileRecord } from "@/types";
import { Card } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { FileKindIcon } from "@/components/shared/file-kind-icon";
import { ProviderTag } from "@/components/shared/provider-tag";
import { formatBytes, formatRelativeTime } from "@/lib/utils";
import {
  DropdownMenu,
  DropdownMenuTrigger,
  DropdownMenuContent,
  DropdownMenuItem,
} from "@/components/ui/dropdown-menu";

export function FileCard({ file }: { file: FileRecord }) {
  return (
    <Card className="group relative flex flex-col overflow-hidden p-0 transition-colors hover:border-border-strong">
      <Link href={`/files/${file.id}`} className="flex flex-col">
        <div
          className="flex h-24 items-center justify-center"
          style={{ backgroundColor: `${file.thumbnailColor}14` }}
        >
          <div style={{ color: file.thumbnailColor }}>
            <FileKindIcon kind={file.kind} className="h-7 w-7" />
          </div>
        </div>
        <div className="flex flex-col gap-2 p-3.5">
          <p className="truncate text-[13px] font-medium text-ink" title={file.name}>
            {file.name}
          </p>
          <div className="flex items-center justify-between text-[12px] text-ink-muted">
            <span className="font-mono">{formatBytes(file.sizeBytes)}</span>
            <span>{formatRelativeTime(file.uploadedAt)}</span>
          </div>
          <div className="flex items-center justify-between">
            <ProviderTag providerId={file.providerId} />
            {file.shared && (
              <Badge variant="info" className="px-1.5">
                <Share2 className="h-2.5 w-2.5" />
              </Badge>
            )}
          </div>
        </div>
      </Link>

      <div className="absolute right-2 top-2 flex items-center gap-1 opacity-0 transition-opacity group-hover:opacity-100">
        {file.starred && <Star className="h-3.5 w-3.5 fill-live text-live" />}
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <button
              className="flex h-6 w-6 items-center justify-center rounded-sm bg-bg-raised/90 text-ink-muted hover:text-ink"
              aria-label="File actions"
            >
              <MoreVertical className="h-3.5 w-3.5" />
            </button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end">
            <DropdownMenuItem>
              <Download className="h-3.5 w-3.5" /> Download
            </DropdownMenuItem>
            <DropdownMenuItem>
              <Share2 className="h-3.5 w-3.5" /> Share
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </div>
    </Card>
  );
}
