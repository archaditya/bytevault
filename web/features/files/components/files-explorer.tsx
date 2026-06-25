"use client";

import { useMemo } from "react";
import { useFiles } from "@/services";
import { useFilesStore } from "@/store";
import { FilesToolbar } from "./files-toolbar";
import { FileCard } from "./file-card";
import { FileListRow } from "./file-list-row";
import { EmptyState } from "@/components/shared/empty-state";
import { Skeleton } from "@/components/ui/skeleton";
import { FolderSearch } from "lucide-react";

export function FilesExplorer() {
  const { data: files, isLoading } = useFiles();
  const { viewMode, searchQuery, kindFilter, providerFilter, sortKey, sortDirection } = useFilesStore();

  const filtered = useMemo(() => {
    if (!files) return [];
    let result = files.filter((f) => {
      const matchesSearch = f.name.toLowerCase().includes(searchQuery.toLowerCase());
      const matchesKind = kindFilter === "all" || f.kind === kindFilter;
      const matchesProvider = providerFilter === "all" || f.providerId === providerFilter;
      return matchesSearch && matchesKind && matchesProvider;
    });

    result = [...result].sort((a, b) => {
      let cmp = 0;
      if (sortKey === "name") cmp = a.name.localeCompare(b.name);
      else if (sortKey === "size") cmp = a.sizeBytes - b.sizeBytes;
      else if (sortKey === "downloads") cmp = a.downloads - b.downloads;
      else cmp = new Date(a.uploadedAt).getTime() - new Date(b.uploadedAt).getTime();
      return sortDirection === "asc" ? cmp : -cmp;
    });

    return result;
  }, [files, searchQuery, kindFilter, providerFilter, sortKey, sortDirection]);

  return (
    <div className="flex flex-col gap-4">
      <FilesToolbar />

      {isLoading ? (
        <div className="grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-4 xl:grid-cols-5">
          {Array.from({ length: 10 }).map((_, i) => (
            <Skeleton key={i} className="h-44" />
          ))}
        </div>
      ) : filtered.length === 0 ? (
        <EmptyState
          icon={FolderSearch}
          title="No files match your filters"
          description="Try a different search term, or clear the type and provider filters to see more results."
        />
      ) : viewMode === "grid" ? (
        <div className="grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-4 xl:grid-cols-5">
          {filtered.map((file) => (
            <FileCard key={file.id} file={file} />
          ))}
        </div>
      ) : (
        <div className="overflow-hidden rounded-md border border-border bg-bg-surface">
          <div className="grid grid-cols-[1fr_110px_140px_110px_40px] gap-4 border-b border-border px-4 py-2.5 text-[11px] font-medium uppercase tracking-wide text-ink-faint">
            <span>Name</span>
            <span>Size</span>
            <span>Provider</span>
            <span>Uploaded</span>
            <span />
          </div>
          {filtered.map((file) => (
            <FileListRow key={file.id} file={file} />
          ))}
        </div>
      )}
    </div>
  );
}
