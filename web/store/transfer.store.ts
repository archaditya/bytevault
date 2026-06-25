import { create } from "zustand";
import { TransferStatus } from "@/types";

export type TransferTab = "all" | TransferStatus;

interface TransferState {
  activeTab: TransferTab;
  directionFilter: "all" | "upload" | "download";
  setActiveTab: (tab: TransferTab) => void;
  setDirectionFilter: (direction: "all" | "upload" | "download") => void;
}

export const useTransferStore = create<TransferState>((set) => ({
  activeTab: "all",
  directionFilter: "all",
  setActiveTab: (tab) => set({ activeTab: tab }),
  setDirectionFilter: (direction) => set({ directionFilter: direction }),
}));
