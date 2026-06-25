import { AnalyticsPoint, ApiKey, ProviderComparisonPoint, SharedLink, User } from "@/types";
import { createSeededRandom, daysAgo, randomFloat, randomInt } from "@/lib/random";
import { files } from "./files";

const rand = createSeededRandom(303);

export const analyticsHistory: AnalyticsPoint[] = Array.from({ length: 30 }, (_, i) => {
  const dayIdx = 29 - i;
  const weekendDip = [0, 6].includes(new Date(daysAgo(dayIdx)).getDay()) ? 0.7 : 1;
  const uploads = Math.round(randomFloat(rand, 60, 180) * weekendDip);
  const downloads = Math.round(randomFloat(rand, 140, 420) * weekendDip);
  return {
    date: daysAgo(dayIdx),
    uploads,
    downloads,
    uploadBytes: Math.round(uploads * randomFloat(rand, 40, 220) * 1024 * 1024),
    downloadBytes: Math.round(downloads * randomFloat(rand, 30, 180) * 1024 * 1024),
    storageBytes: Math.round(1_400_000_000_000 + i * randomFloat(rand, 4_000_000_000, 9_000_000_000)),
    successRate: randomFloat(rand, 96.2, 99.8),
    failedTransfers: randomInt(rand, 0, 14),
  };
});

export const providerComparison: ProviderComparisonPoint[] = [
  { provider: "Cloudflare R2", avgLatencyMs: 38, successRate: 99.4, costPerGb: 0.015, throughputMbps: 312 },
  { provider: "AWS S3", avgLatencyMs: 61, successRate: 99.7, costPerGb: 0.023, throughputMbps: 268 },
  { provider: "Local Storage", avgLatencyMs: 4, successRate: 97.1, costPerGb: 0, throughputMbps: 890 },
];

export const sharedLinks: SharedLink[] = Array.from({ length: 14 }, (_, i) => {
  const file = files[(i * 5) % files.length];
  const sRand = createSeededRandom(i * 211 + 9);
  const hasExpiry = sRand() > 0.4;
  const hasLimit = sRand() > 0.55;
  const active = sRand() > 0.18;
  return {
    id: `link_${i.toString(36)}`,
    fileId: file.id,
    fileName: file.name,
    url: `https://bytevault.sh/s/${file.id.slice(5)}${i}`,
    createdAt: daysAgo(randomInt(sRand, 0, 60), 12, sRand),
    expiresAt: hasExpiry ? daysAgo(-randomInt(sRand, 1, 30)) : null,
    passwordProtected: sRand() > 0.5,
    downloadLimit: hasLimit ? randomInt(sRand, 5, 200) : null,
    downloadCount: randomInt(sRand, 0, 180),
    views: randomInt(sRand, 0, 540),
    active,
  };
});

export const currentUser: User = {
  id: "user_1",
  name: "Aarav Mehta",
  email: "aarav@bytevault.sh",
  avatar: "AM",
  role: "Staff Engineer",
  plan: "enterprise",
  joinedAt: daysAgo(412),
  apiKeysCount: 4,
  twoFactorEnabled: true,
};

export const apiKeys: ApiKey[] = [
  { id: "key_1", label: "Production CI", prefix: "bv_live_7k2p", createdAt: daysAgo(180), lastUsedAt: daysAgo(0, 2, rand), scopes: ["transfer:write", "files:read"] },
  { id: "key_2", label: "Local development", prefix: "bv_test_9a3x", createdAt: daysAgo(95), lastUsedAt: daysAgo(1, 5, rand), scopes: ["transfer:write", "files:read", "files:delete"] },
  { id: "key_3", label: "Analytics export job", prefix: "bv_live_4m8q", createdAt: daysAgo(260), lastUsedAt: daysAgo(7, 0, rand), scopes: ["analytics:read"] },
  { id: "key_4", label: "Mobile app (staging)", prefix: "bv_test_1z6v", createdAt: daysAgo(40), lastUsedAt: null, scopes: ["transfer:write"] },
];
