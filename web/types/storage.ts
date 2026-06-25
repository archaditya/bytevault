export type ProviderStatus = "healthy" | "degraded" | "down";

export interface StorageProvider {
  id: "r2" | "s3" | "local";
  name: string;
  vendor: string;
  region: string;
  status: ProviderStatus;
  latencyMs: number;
  usedBytes: number;
  capacityBytes: number;
  availabilityPct: number;
  costPerGbCents: number;
  fileCount: number;
  isDefault: boolean;
  isPrimary: boolean;
  uptimeHistory: { day: string; uptimePct: number }[];
}
