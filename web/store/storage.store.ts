import { create } from "zustand";

interface StorageState {
  selectedProviderId: string | null;
  setSelectedProviderId: (id: string | null) => void;
}

export const useStorageStore = create<StorageState>((set) => ({
  selectedProviderId: null,
  setSelectedProviderId: (id) => set({ selectedProviderId: id }),
}));
