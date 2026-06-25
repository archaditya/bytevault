import { create } from "zustand";

export type AnalyticsRange = "7d" | "14d" | "30d";

interface AnalyticsState {
  range: AnalyticsRange;
  setRange: (range: AnalyticsRange) => void;
}

export const useAnalyticsStore = create<AnalyticsState>((set) => ({
  range: "30d",
  setRange: (range) => set({ range }),
}));
