import { create } from "zustand";
import { FileKind } from "@/types";

export type FileViewMode = "grid" | "list";
export type FileSortKey = "name" | "size" | "uploadedAt" | "downloads";

interface FilesState {
  viewMode: FileViewMode;
  searchQuery: string;
  sortKey: FileSortKey;
  sortDirection: "asc" | "desc";
  kindFilter: FileKind | "all";
  providerFilter: string | "all";
  setViewMode: (mode: FileViewMode) => void;
  setSearchQuery: (query: string) => void;
  setSort: (key: FileSortKey) => void;
  setKindFilter: (kind: FileKind | "all") => void;
  setProviderFilter: (provider: string | "all") => void;
}

export const useFilesStore = create<FilesState>((set, get) => ({
  viewMode: "grid",
  searchQuery: "",
  sortKey: "uploadedAt",
  sortDirection: "desc",
  kindFilter: "all",
  providerFilter: "all",
  setViewMode: (mode) => set({ viewMode: mode }),
  setSearchQuery: (query) => set({ searchQuery: query }),
  setSort: (key) =>
    set((state) => ({
      sortKey: key,
      sortDirection: state.sortKey === key && state.sortDirection === "desc" ? "asc" : "desc",
    })),
  setKindFilter: (kind) => set({ kindFilter: kind }),
  setProviderFilter: (provider) => set({ providerFilter: provider }),
}));
