import {
  ChunkState,
  SpeedSample,
  TransferLogEntry,
  TransferSession,
  TransferStatus,
} from "@/types";
import {
  createSeededRandom,
  minutesAgo,
  pick,
  pickWeighted,
  randomFloat,
  randomInt,
} from "@/lib/random";
import { files } from "./files";

const rand = createSeededRandom(101);

const CHUNK_SIZE = 8 * 1024 * 1024; // 8 MB chunks

function buildChunks(totalChunks: number, status: TransferStatus, progress: number, rnd: () => number): ChunkState[] {
  const completedCount = Math.floor(totalChunks * progress);
  return Array.from({ length: totalChunks }, (_, i) => {
    if (i < completedCount) return { index: i, status: "complete" as const, retries: rnd() > 0.85 ? 1 : 0 };
    if (status === "failed" && i === completedCount) {
      return { index: i, status: "failed" as const, retries: randomInt(rnd, 2, 5) };
    }
    if (status === "active" && i === completedCount) {
      return { index: i, status: "uploading" as const, retries: 0 };
    }
    if (status === "active" && i === completedCount + 1 && rnd() > 0.7) {
      return { index: i, status: "retrying" as const, retries: 1 };
    }
    return { index: i, status: "pending" as const, retries: 0 };
  });
}

function buildLogs(status: TransferStatus, fileName: string, rnd: () => number, retryCount: number): TransferLogEntry[] {
  const logs: TransferLogEntry[] = [];
  let t = 0;
  const push = (level: TransferLogEntry["level"], message: string) => {
    t += randomInt(rnd, 4, 40);
    logs.push({ id: `log_${logs.length}`, timestamp: minutesAgo(t), level, message });
  };
  push("info", `Transfer session initialized for ${fileName}`);
  push("info", "Negotiated chunk size: 8 MB, parallel streams: 4");
  for (let i = 0; i < retryCount; i++) {
    push("warn", `Chunk upload timed out, retrying (attempt ${i + 1})`);
  }
  if (status === "failed") {
    push("error", "Exceeded max retry threshold, marking transfer as failed");
  } else if (status === "completed") {
    push("info", "Checksum verified, transfer marked complete");
  } else if (status === "paused") {
    push("warn", "Transfer paused by user");
  } else {
    push("info", "Streaming chunk data to provider endpoint");
  }
  return logs.reverse();
}

function buildSpeedHistory(rnd: () => number, baseline: number, points: number): SpeedSample[] {
  return Array.from({ length: points }, (_, i) => ({
    t: i * 5,
    bytesPerSecond: Math.max(0, baseline + randomFloat(rnd, -baseline * 0.35, baseline * 0.35)),
  }));
}

const statusWeights: [TransferStatus, number][] = [
  ["active", 0.28],
  ["queued", 0.1],
  ["paused", 0.12],
  ["completed", 0.38],
  ["failed", 0.12],
];

function buildSession(idx: number): TransferSession {
  const sRand = createSeededRandom(idx * 733 + 11);
  const file = pick(sRand, files);
  const status = pickWeighted(sRand, statusWeights);
  const direction = sRand() > 0.42 ? "upload" : "download";
  const totalChunks = Math.max(1, Math.ceil(file.sizeBytes / CHUNK_SIZE));

  let progress: number;
  if (status === "completed") progress = 1;
  else if (status === "queued") progress = 0;
  else if (status === "failed") progress = randomFloat(sRand, 0.1, 0.75);
  else progress = randomFloat(sRand, 0.05, 0.92);

  const transferredBytes = Math.round(file.sizeBytes * progress);
  const speedBps =
    status === "active"
      ? randomFloat(sRand, 1.5, 42) * 1024 * 1024
      : status === "paused"
      ? 0
      : 0;
  const remaining = file.sizeBytes - transferredBytes;
  const eta = status === "active" && speedBps > 0 ? remaining / speedBps : null;
  const retryCount =
    status === "failed" ? randomInt(sRand, 2, 6) : status === "active" ? randomInt(sRand, 0, 2) : 0;

  const startedMinutesAgo = randomInt(sRand, 2, 4200);
  const startedAt = minutesAgo(startedMinutesAgo);
  const updatedAt = minutesAgo(status === "active" ? randomInt(sRand, 0, 2) : randomInt(sRand, 5, startedMinutesAgo));
  const completedAt = status === "completed" ? minutesAgo(randomInt(sRand, 1, startedMinutesAgo)) : null;

  return {
    id: `txn_${idx.toString(36)}`,
    fileName: file.name,
    fileId: file.id,
    direction,
    status,
    providerId: file.providerId,
    sizeBytes: file.sizeBytes,
    transferredBytes,
    speedBytesPerSecond: speedBps,
    etaSeconds: eta,
    retryCount,
    startedAt,
    updatedAt,
    completedAt,
    chunkSize: CHUNK_SIZE,
    totalChunks,
    chunks: buildChunks(Math.min(totalChunks, 240), status, progress, sRand),
    logs: buildLogs(status, file.name, sRand, retryCount),
    speedHistory: buildSpeedHistory(sRand, status === "active" ? speedBps : 8 * 1024 * 1024, 24),
    initiatedBy: file.ownerName,
  };
}

export const transferSessions: TransferSession[] = Array.from({ length: 64 }, (_, i) => buildSession(i));

export function getTransferById(id: string): TransferSession | undefined {
  return transferSessions.find((t) => t.id === id);
}

export function getTransfersByStatus(status: TransferStatus): TransferSession[] {
  return transferSessions.filter((t) => t.status === status);
}

export const transferStats = {
  active: transferSessions.filter((t) => t.status === "active").length,
  paused: transferSessions.filter((t) => t.status === "paused").length,
  completed: transferSessions.filter((t) => t.status === "completed").length,
  failed: transferSessions.filter((t) => t.status === "failed").length,
  queued: transferSessions.filter((t) => t.status === "queued").length,
};
