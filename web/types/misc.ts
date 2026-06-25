export interface SharedLink {
  id: string;
  fileId: string;
  fileName: string;
  url: string;
  createdAt: string;
  expiresAt: string | null;
  passwordProtected: boolean;
  downloadLimit: number | null;
  downloadCount: number;
  views: number;
  active: boolean;
}

export interface AnalyticsPoint {
  date: string;
  uploads: number;
  downloads: number;
  uploadBytes: number;
  downloadBytes: number;
  storageBytes: number;
  successRate: number;
  failedTransfers: number;
}

export interface ProviderComparisonPoint {
  provider: string;
  avgLatencyMs: number;
  successRate: number;
  costPerGb: number;
  throughputMbps: number;
}

export interface User {
  id: string;
  name: string;
  email: string;
  avatar: string;
  role: string;
  plan: "free" | "pro" | "enterprise";
  joinedAt: string;
  apiKeysCount: number;
  twoFactorEnabled: boolean;
}

export interface ApiKey {
  id: string;
  label: string;
  prefix: string;
  createdAt: string;
  lastUsedAt: string | null;
  scopes: string[];
}
