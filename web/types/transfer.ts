export type TransferDirection = "upload" | "download";

export type TransferStatus =
  | "active"
  | "paused"
  | "completed"
  | "failed"
  | "queued";

export type ChunkStatus = "pending" | "uploading" | "complete" | "failed" | "retrying";

export interface ChunkState {
  index: number;
  status: ChunkStatus;
  retries: number;
}

export interface TransferLogEntry {
  id: string;
  timestamp: string;
  level: "info" | "warn" | "error";
  message: string;
}

export interface SpeedSample {
  t: number; // seconds since start
  bytesPerSecond: number;
}

export interface TransferSession {
  id: string;
  fileName: string;
  fileId: string;
  direction: TransferDirection;
  status: TransferStatus;
  providerId: "r2" | "s3" | "local";
  sizeBytes: number;
  transferredBytes: number;
  speedBytesPerSecond: number;
  etaSeconds: number | null;
  retryCount: number;
  startedAt: string;
  updatedAt: string;
  completedAt: string | null;
  chunkSize: number;
  totalChunks: number;
  chunks: ChunkState[];
  logs: TransferLogEntry[];
  speedHistory: SpeedSample[];
  initiatedBy: string;
}
