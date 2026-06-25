import { StorageProvider } from "@/types";
import { createSeededRandom, randomFloat } from "@/lib/random";

const rand = createSeededRandom(42);

function buildUptimeHistory(base: number, rnd: () => number) {
  const days = ["Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"];
  return days.map((day) => ({
    day,
    uptimePct: Math.min(100, base + randomFloat(rnd, -0.4, 0.15)),
  }));
}

export const storageProviders: StorageProvider[] = [
  {
    id: "r2",
    name: "Cloudflare R2",
    vendor: "Cloudflare",
    region: "Auto (Global Anycast)",
    status: "healthy",
    latencyMs: 38,
    usedBytes: 482_300_000_000,
    capacityBytes: 2_000_000_000_000,
    availabilityPct: 99.98,
    costPerGbCents: 1.5,
    fileCount: 18432,
    isDefault: true,
    isPrimary: true,
    uptimeHistory: buildUptimeHistory(99.95, rand),
  },
  {
    id: "s3",
    name: "AWS S3",
    vendor: "Amazon Web Services",
    region: "us-east-1",
    status: "healthy",
    latencyMs: 61,
    usedBytes: 891_400_000_000,
    capacityBytes: 5_000_000_000_000,
    availabilityPct: 99.99,
    costPerGbCents: 2.3,
    fileCount: 27115,
    isDefault: false,
    isPrimary: false,
    uptimeHistory: buildUptimeHistory(99.97, rand),
  },
  {
    id: "local",
    name: "Local Storage",
    vendor: "On-Prem NVMe Cluster",
    region: "dc-jaipur-1",
    status: "degraded",
    latencyMs: 4,
    usedBytes: 210_900_000_000,
    capacityBytes: 500_000_000_000,
    availabilityPct: 99.42,
    costPerGbCents: 0,
    fileCount: 6204,
    isDefault: false,
    isPrimary: false,
    uptimeHistory: buildUptimeHistory(98.6, rand),
  },
];

export function getProviderById(id: string): StorageProvider | undefined {
  return storageProviders.find((p) => p.id === id);
}
