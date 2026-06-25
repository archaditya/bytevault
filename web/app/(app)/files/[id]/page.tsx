"use client";

import { use } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import {
  Download,
  Share2,
  Star,
  ChevronLeft,
  Hash,
  Calendar,
  User,
  Clock,
  Trash2,
} from "lucide-react";
import { useFile, useFileHistory, useDeleteFileMutation, useToggleShareMutation } from "@/services";
import { Card, CardHeader, CardTitle, CardContent } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { FileKindIcon } from "@/components/shared/file-kind-icon";
import { ProviderTag } from "@/components/shared/provider-tag";
import { formatBytes, formatDateTime, formatRelativeTime, truncateMiddle } from "@/lib/utils";

const actionLabel: Record<string, string> = {
  uploaded: "Uploaded",
  downloaded: "Downloaded",
  shared: "Shared",
  moved: "Moved",
  replicated: "Replicated",
};

export default function FileDetailsPage({ params }: { params: Promise<{ id: string }> | { id: string } }) {
  // Safe resolution of Next.js 15 params promise
  const resolvedParams = params instanceof Promise ? use(params) : params;
  const id = resolvedParams.id;
  const router = useRouter();

  const { data: file, isLoading } = useFile(id);
  const { data: history = [] } = useFileHistory(id);

  const deleteMutation = useDeleteFileMutation();
  const toggleShareMutation = useToggleShareMutation();

  if (isLoading) {
    return (
      <div className="flex h-64 items-center justify-center bg-bg">
        <div className="flex flex-col items-center gap-2">
          <div className="h-8 w-8 animate-spin rounded-full border-2 border-accent border-t-transparent" />
          <span className="font-mono text-[12px] text-ink-muted">Loading file metadata...</span>
        </div>
      </div>
    );
  }

  if (!file) {
    return (
      <div className="flex flex-col items-center justify-center py-12 text-center">
        <h2 className="text-lg font-semibold text-ink">File not found</h2>
        <p className="mt-1 text-[13px] text-ink-muted">The file you are looking for does not exist or has been deleted.</p>
        <Link href="/files" className="mt-4 inline-flex items-center gap-1 text-[13px] text-accent hover:underline">
          Back to files
        </Link>
      </div>
    );
  }

  const handleDownload = () => {
    window.open(`/api/v1/files/${file.id}/download`, "_blank");
  };

  const handleToggleShare = () => {
    toggleShareMutation.mutate({ id: file.id, isPublic: !file.shared });
  };

  const handleDelete = () => {
    if (confirm(`Are you sure you want to delete "${file.name}"?`)) {
      deleteMutation.mutate(file.id, {
        onSuccess: () => {
          router.push("/files");
        },
      });
    }
  };

  return (
    <div className="flex flex-col gap-6">
      <Link href="/files" className="inline-flex items-center gap-1 text-[13px] text-ink-muted hover:text-ink">
        <ChevronLeft className="h-3.5 w-3.5" /> Back to files
      </Link>

      <div className="flex flex-wrap items-start justify-between gap-4">
        <div className="flex items-start gap-3">
          <div
            className="flex h-12 w-12 items-center justify-center rounded-md"
            style={{ backgroundColor: `${file.thumbnailColor}1A`, color: file.thumbnailColor }}
          >
            <FileKindIcon kind={file.kind} className="h-6 w-6" />
          </div>
          <div>
            <div className="flex items-center gap-2">
              <h2 className="text-lg font-semibold text-ink">{truncateMiddle(file.name, 50)}</h2>
              {file.starred && <Star className="h-4 w-4 fill-live text-live" />}
            </div>
            <p className="mt-0.5 text-[13px] text-ink-muted">{file.path}</p>
          </div>
        </div>
        <div className="flex gap-2">
          <Button size="sm" variant="secondary" onClick={handleToggleShare} disabled={toggleShareMutation.isPending}>
            <Share2 className="h-3.5 w-3.5" /> {file.shared ? "Make Private" : "Share Link"}
          </Button>
          <Button size="sm" onClick={handleDownload}>
            <Download className="h-3.5 w-3.5" /> Download
          </Button>
          <Button size="sm" variant="danger" onClick={handleDelete} disabled={deleteMutation.isPending}>
            <Trash2 className="h-3.5 w-3.5" /> Delete
          </Button>
        </div>
      </div>

      <div className="grid grid-cols-1 gap-4 lg:grid-cols-3">
        <Card className="lg:col-span-2 bg-bg-surface border-border-strong">
          <CardHeader>
            <CardTitle>Metadata</CardTitle>
          </CardHeader>
          <CardContent className="grid grid-cols-2 gap-5 sm:grid-cols-3">
            <Field label="Size" value={formatBytes(file.sizeBytes)} icon={Hash} />
            <Field label="Type" value={file.mimeType} icon={Hash} />
            <Field label="Provider" value={<ProviderTag providerId={file.providerId} />} icon={Hash} />
            <Field label="Uploaded" value={formatDateTime(file.uploadedAt)} icon={Calendar} />
            <Field label="Last updated" value={formatRelativeTime(file.updatedAt)} icon={Clock} />
            <Field label="Owner" value={file.ownerName} icon={User} />
            <Field label="Checksum (SHA-1)" value={<span className="font-mono text-[11px]">{file.checksum ? file.checksum.slice(0, 16) : ""}…</span>} icon={Hash} />
            <Field label="Downloads" value={file.downloads.toLocaleString()} icon={Download} />
          </CardContent>
          {file.tags.length > 0 && (
            <div className="flex flex-wrap gap-1.5 px-4 pb-4">
              {file.tags.map((tag) => (
                <Badge key={tag} variant="muted">{tag}</Badge>
              ))}
            </div>
          )}
        </Card>

        <Card className="bg-bg-surface border-border-strong">
          <CardHeader>
            <CardTitle>Share settings</CardTitle>
          </CardHeader>
          <CardContent className="flex flex-col gap-3">
            <div className="flex items-center justify-between text-[13px]">
              <span className="text-ink-muted">Visibility</span>
              <Badge variant={file.shared ? "info" : "muted"}>{file.shared ? "Public" : "Private"}</Badge>
            </div>
            <div className="flex items-center justify-between text-[13px]">
              <span className="text-ink-muted">Total downloads</span>
              <span className="font-mono text-ink">{file.downloads}</span>
            </div>
            <Button size="sm" variant="secondary" className="mt-1" onClick={handleToggleShare} disabled={toggleShareMutation.isPending}>
              Toggle Sharing
            </Button>
          </CardContent>
        </Card>
      </div>

      {history.length > 0 && (
        <Card className="bg-bg-surface border-border-strong">
          <CardHeader>
            <CardTitle>Transfer history</CardTitle>
          </CardHeader>
          <CardContent className="flex flex-col">
            {history.map((entry: any) => (
              <div key={entry.id} className="flex items-start justify-between gap-4 border-t border-border py-3 first:border-t-0 first:pt-0">
                <div className="flex items-start gap-3">
                  <Badge variant="muted" className="mt-0.5">{actionLabel[entry.action]}</Badge>
                  <div>
                    <p className="text-[13px] text-ink">{entry.detail}</p>
                    <p className="mt-0.5 text-[12px] text-ink-faint">by {entry.actor}</p>
                  </div>
                </div>
                <span className="whitespace-nowrap font-mono text-[12px] text-ink-faint">
                  {formatRelativeTime(entry.timestamp)}
                </span>
              </div>
            ))}
          </CardContent>
        </Card>
      )}
    </div>
  );
}

function Field({ label, value, icon: Icon }: { label: string; value: React.ReactNode; icon: any }) {
  return (
    <div>
      <p className="flex items-center gap-1 text-[11px] uppercase tracking-wide text-ink-faint">
        <Icon className="h-3 w-3" /> {label}
      </p>
      <div className="mt-1 text-[13px] font-medium text-ink">{value}</div>
    </div>
  );
}
