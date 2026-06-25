export type FileKind =
  | "document"
  | "image"
  | "video"
  | "audio"
  | "archive"
  | "code"
  | "dataset"
  | "other";

export type StorageProviderId = "r2" | "s3" | "local";

export interface FileRecord {
  id: string;
  name: string;
  kind: FileKind;
  sizeBytes: number;
  mimeType: string;
  providerId: StorageProviderId;
  uploadedAt: string;
  updatedAt: string;
  ownerName: string;
  ownerAvatar: string;
  checksum: string;
  downloads: number;
  shared: boolean;
  starred: boolean;
  tags: string[];
  path: string;
  thumbnailColor: string;
}

export interface FileTransferHistoryEntry {
  id: string;
  action: "uploaded" | "downloaded" | "shared" | "moved" | "replicated";
  actor: string;
  timestamp: string;
  detail: string;
}
