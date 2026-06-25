import { FileKind, FileRecord, FileTransferHistoryEntry, StorageProviderId } from "@/types";
import { createSeededRandom, daysAgo, pick, pickWeighted, randomInt } from "@/lib/random";

const rand = createSeededRandom(7);

const kindByExt: Record<string, FileKind> = {
  pdf: "document",
  docx: "document",
  md: "document",
  png: "image",
  jpg: "image",
  jpeg: "image",
  webp: "image",
  mp4: "video",
  mov: "video",
  mp3: "audio",
  wav: "audio",
  zip: "archive",
  tar: "archive",
  gz: "archive",
  ts: "code",
  tsx: "code",
  py: "code",
  csv: "dataset",
  parquet: "dataset",
  json: "dataset",
};

const fileTemplates: { name: string; ext: string; minMb: number; maxMb: number }[] = [
  { name: "Q3-engineering-roadmap", ext: "pdf", minMb: 1, maxMb: 6 },
  { name: "production-incident-postmortem", ext: "md", minMb: 0.1, maxMb: 1 },
  { name: "infra-architecture-diagram", ext: "png", minMb: 2, maxMb: 12 },
  { name: "customer-onboarding-demo", ext: "mp4", minMb: 80, maxMb: 640 },
  { name: "release-notes-v4.2", ext: "pdf", minMb: 0.2, maxMb: 2 },
  { name: "user-research-interviews", ext: "mp3", minMb: 20, maxMb: 90 },
  { name: "monorepo-backup", ext: "tar", minMb: 400, maxMb: 2200 },
  { name: "billing-service", ext: "tar", minMb: 50, maxMb: 300 },
  { name: "transfer-api-client", ext: "ts", minMb: 0.01, maxMb: 0.3 },
  { name: "chunked-upload-handler", ext: "ts", minMb: 0.01, maxMb: 0.2 },
  { name: "etl-pipeline", ext: "py", minMb: 0.05, maxMb: 0.4 },
  { name: "warehouse-events-export", ext: "csv", minMb: 120, maxMb: 900 },
  { name: "model-training-dataset", ext: "parquet", minMb: 600, maxMb: 4200 },
  { name: "brand-guidelines-2026", ext: "pdf", minMb: 8, maxMb: 24 },
  { name: "keynote-recording-fulltalk", ext: "mp4", minMb: 900, maxMb: 3400 },
  { name: "design-system-tokens", ext: "json", minMb: 0.02, maxMb: 0.1 },
  { name: "press-kit-assets", ext: "zip", minMb: 40, maxMb: 220 },
  { name: "server-logs-archive", ext: "gz", minMb: 200, maxMb: 1100 },
  { name: "team-offsite-photos", ext: "zip", minMb: 300, maxMb: 1800 },
  { name: "api-load-test-results", ext: "csv", minMb: 5, maxMb: 60 },
  { name: "hero-banner-final", ext: "webp", minMb: 0.5, maxMb: 4 },
  { name: "product-walkthrough-narrated", ext: "mov", minMb: 500, maxMb: 2600 },
  { name: "support-ticket-export", ext: "json", minMb: 10, maxMb: 80 },
  { name: "schema-migration-snapshot", ext: "sql", minMb: 4, maxMb: 40 },
  { name: "feature-flag-rollout-plan", ext: "docx", minMb: 0.1, maxMb: 1.2 },
];

const owners = [
  { name: "Aarav Mehta", avatar: "AM" },
  { name: "Priya Sharma", avatar: "PS" },
  { name: "Liu Wei", avatar: "LW" },
  { name: "Sofia Rossi", avatar: "SR" },
  { name: "Kwame Boateng", avatar: "KB" },
  { name: "Hana Kobayashi", avatar: "HK" },
  { name: "Daniel Cohen", avatar: "DC" },
  { name: "Fatima Al-Sayed", avatar: "FA" },
];

const thumbColors = ["#5E6AD2", "#F5A623", "#4CB782", "#E5484D", "#5E9DD2", "#8A6420"];

const providers: StorageProviderId[] = ["r2", "s3", "local"];
const providerWeights: [StorageProviderId, number][] = [
  ["r2", 0.5],
  ["s3", 0.38],
  ["local", 0.12],
];

function hexChecksum(rnd: () => number): string {
  const chars = "0123456789abcdef";
  let out = "";
  for (let i = 0; i < 40; i++) out += chars[Math.floor(rnd() * chars.length)];
  return out;
}

function buildFile(idx: number): FileRecord {
  const t = fileTemplates[idx % fileTemplates.length];
  const variant = Math.floor(idx / fileTemplates.length);
  const owner = pick(rand, owners);
  const sizeBytes = Math.round(
    (t.minMb + rand() * (t.maxMb - t.minMb)) * 1024 * 1024
  );
  const uploadedDaysAgo = randomInt(rand, 0, 95);
  const uploadedAt = daysAgo(uploadedDaysAgo, 20, rand);
  const updatedAt = daysAgo(Math.max(0, uploadedDaysAgo - randomInt(rand, 0, 5)), 12, rand);

  return {
    id: `file_${idx.toString(36)}`,
    name: variant > 0 ? `${t.name}-v${variant + 1}.${t.ext}` : `${t.name}.${t.ext}`,
    kind: kindByExt[t.ext] ?? "other",
    sizeBytes,
    mimeType: `application/${t.ext}`,
    providerId: pickWeighted(rand, providerWeights),
    uploadedAt,
    updatedAt,
    ownerName: owner.name,
    ownerAvatar: owner.avatar,
    checksum: hexChecksum(rand),
    downloads: randomInt(rand, 0, 480),
    shared: rand() > 0.72,
    starred: rand() > 0.85,
    tags: pickTags(),
    path: `/${pick(rand, ["engineering", "design", "marketing", "data", "ops"])}/${t.name}.${t.ext}`,
    thumbnailColor: pick(rand, thumbColors),
  };
}

function pickTags(): string[] {
  const pool = ["backup", "release", "internal", "client-facing", "archived", "training-data", "media", "compliance"];
  const count = randomInt(rand, 0, 3);
  const tags = new Set<string>();
  for (let i = 0; i < count; i++) tags.add(pick(rand, pool));
  return Array.from(tags);
}

export const files: FileRecord[] = Array.from({ length: 86 }, (_, i) => buildFile(i));

export function getFileById(id: string): FileRecord | undefined {
  return files.find((f) => f.id === id);
}

export function getFileHistory(fileId: string): FileTransferHistoryEntry[] {
  const file = getFileById(fileId);
  if (!file) return [];
  const hRand = createSeededRandom(fileId.length * 991 + 3);
  const actions: FileTransferHistoryEntry["action"][] = ["uploaded", "downloaded", "shared", "downloaded", "replicated", "downloaded"];
  return actions.slice(0, randomInt(hRand, 3, 6)).map((action, i) => ({
    id: `${fileId}_hist_${i}`,
    action,
    actor: i === 0 ? file.ownerName : pick(hRand, owners).name,
    timestamp: daysAgo(randomInt(hRand, 0, 60) - i * 2, 10, hRand),
    detail:
      action === "uploaded"
        ? `Initial upload via ${file.providerId.toUpperCase()} chunked transfer`
        : action === "downloaded"
        ? `Downloaded ${(0.4 + hRand() * 0.6) > 0.5 ? "full file" : "partial range"}`
        : action === "shared"
        ? "Created a shareable link"
        : "Replicated to secondary storage provider",
  }));
}
