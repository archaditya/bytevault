"use client";

import { useRef } from "react";
import { Grid3x3, List, Search, Upload, Loader2 } from "lucide-react";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import {
  Select,
  SelectTrigger,
  SelectContent,
  SelectItem,
  SelectValue,
} from "@/components/ui/select";
import { useFilesStore } from "@/store";
import { cn } from "@/lib/utils";
import { useUploadFileMutation } from "@/services";

const kindOptions = [
  { value: "all", label: "All types" },
  { value: "document", label: "Documents" },
  { value: "image", label: "Images" },
  { value: "video", label: "Video" },
  { value: "audio", label: "Audio" },
  { value: "archive", label: "Archives" },
  { value: "code", label: "Code" },
  { value: "dataset", label: "Datasets" },
];

const providerOptions = [
  { value: "all", label: "All providers" },
  { value: "r2", label: "Cloudflare R2" },
  { value: "s3", label: "AWS S3" },
  { value: "local", label: "Local Storage" },
];

export function FilesToolbar() {
  const fileInputRef = useRef<HTMLInputElement>(null);
  const uploadMutation = useUploadFileMutation();

  const {
    viewMode,
    setViewMode,
    searchQuery,
    setSearchQuery,
    kindFilter,
    setKindFilter,
    providerFilter,
    setProviderFilter,
  } = useFilesStore();

  const handleUploadClick = () => {
    fileInputRef.current?.click();
  };

  const handleFileChange = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const selectedFile = e.target.files?.[0];
    if (!selectedFile) return;

    try {
      await uploadMutation.mutateAsync(selectedFile);
    } catch (err) {
      console.error("Upload failed:", err);
    } finally {
      if (fileInputRef.current) {
        fileInputRef.current.value = "";
      }
    }
  };

  return (
    <div className="flex flex-wrap items-center gap-2.5">
      {/* Invisible File Input */}
      <input
        type="file"
        ref={fileInputRef}
        onChange={handleFileChange}
        className="hidden"
      />

      <div className="relative min-w-[220px] flex-1 sm:max-w-xs">
        <Search className="pointer-events-none absolute left-2.5 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-ink-faint" />
        <Input
          placeholder="Search files..."
          className="pl-8"
          value={searchQuery}
          onChange={(e) => setSearchQuery(e.target.value)}
        />
      </div>

      <Select value={kindFilter} onValueChange={(v) => setKindFilter(v as any)}>
        <SelectTrigger className="w-[150px]">
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          {kindOptions.map((opt) => (
            <SelectItem key={opt.value} value={opt.value}>
              {opt.label}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>

      <Select value={providerFilter} onValueChange={(v) => setProviderFilter(v)}>
        <SelectTrigger className="w-[160px]">
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          {providerOptions.map((opt) => (
            <SelectItem key={opt.value} value={opt.value}>
              {opt.label}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>

      <div className="ml-auto flex items-center gap-2">
        <div className="flex items-center rounded-md border border-border-strong bg-bg-surface p-0.5">
          <button
            onClick={() => setViewMode("grid")}
            className={cn(
              "flex h-7 w-7 items-center justify-center rounded-sm transition-colors",
              viewMode === "grid" ? "bg-bg-overlay text-ink" : "text-ink-faint hover:text-ink"
            )}
            aria-label="Grid view"
          >
            <Grid3x3 className="h-3.5 w-3.5" />
          </button>
          <button
            onClick={() => setViewMode("list")}
            className={cn(
              "flex h-7 w-7 items-center justify-center rounded-sm transition-colors",
              viewMode === "list" ? "bg-bg-overlay text-ink" : "text-ink-faint hover:text-ink"
            )}
            aria-label="List view"
          >
            <List className="h-3.5 w-3.5" />
          </button>
        </div>
        <Button
          size="sm"
          onClick={handleUploadClick}
          disabled={uploadMutation.isPending}
        >
          {uploadMutation.isPending ? (
            <Loader2 className="h-3.5 w-3.5 animate-spin" />
          ) : (
            <Upload className="h-3.5 w-3.5" />
          )}
          {uploadMutation.isPending ? "Uploading..." : "Upload"}
        </Button>
      </div>
    </div>
  );
}