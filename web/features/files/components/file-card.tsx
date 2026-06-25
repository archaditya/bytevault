"use client";

import Link from "next/link";
import { Star, Share2, MoreVertical, Download, Trash2, Globe } from "lucide-react";
import { FileRecord } from "@/types";
import { Card } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { FileKindIcon } from "@/components/shared/file-kind-icon";
import { ProviderTag } from "@/components/shared/provider-tag";
import { formatBytes, formatRelativeTime } from "@/lib/utils";
import { useDeleteFileMutation, useToggleShareMutation } from "@/services";
import {
  DropdownMenu,
  DropdownMenuTrigger,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
} from "@/components/ui/dropdown-menu";

export function FileCard({ file }: { file: FileRecord }) {
  const deleteMutation = useDeleteFileMutation();
  const toggleShareMutation = useToggleShareMutation();

  const handleDownload = () => {
    // Navigate window to download route, which responds with file stream attachment
    window.open(`/api/v1/files/${file.id}/download`, "_blank");
  };

  const handleToggleShare = () => {
    toggleShareMutation.mutate({ id: file.id, isPublic: !file.shared });
  };

  const handleDelete = () => {
    if (confirm(`Are you sure you want to delete "${file.name}"?`)) {
      deleteMutation.mutate(file.id);
    }
  };

  return (
    <Card className="group relative flex flex-col overflow-hidden p-0 transition-colors hover:border-border-strong bg-bg-surface">
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
              <Badge variant="info" className="px-1.5 flex items-center gap-1">
                <Globe className="h-2.5 w-2.5" />
                <span>Public</span>
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
              className="flex h-6 w-6 items-center justify-center rounded-sm bg-bg-raised/90 text-ink-muted hover:text-ink border border-border"
              aria-label="File actions"
            >
              <MoreVertical className="h-3.5 w-3.5" />
            </button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end" className="bg-bg-surface border-border-strong">
            <DropdownMenuItem onClick={handleDownload} className="cursor-pointer hover:bg-bg-overlay">
              <Download className="h-3.5 w-3.5 mr-2" /> Download
            </DropdownMenuItem>
            <DropdownMenuItem onClick={handleToggleShare} className="cursor-pointer hover:bg-bg-overlay">
              <Share2 className="h-3.5 w-3.5 mr-2" /> {file.shared ? "Make Private" : "Share Link"}
            </DropdownMenuItem>
            <DropdownMenuSeparator className="bg-border-strong" />
            <DropdownMenuItem onClick={handleDelete} className="cursor-pointer text-danger hover:bg-danger/10">
              <Trash2 className="h-3.5 w-3.5 mr-2" /> Delete
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </div>
    </Card>
  );
}