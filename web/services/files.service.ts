import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { apiClient } from "@/lib/api-client";
import { FileRecord, FileKind } from "@/types";

// Determine file category kind based on mimeType/contentType
function determineFileKind(contentType: string): FileKind {
  const mime = contentType.toLowerCase();
  if (mime.startsWith("image/")) return "image";
  if (mime.startsWith("video/")) return "video";
  if (mime.startsWith("audio/")) return "audio";
  if (
    mime.startsWith("text/") ||
    mime.includes("javascript") ||
    mime.includes("json") ||
    mime.includes("typescript") ||
    mime.includes("html") ||
    mime.includes("css") ||
    mime.includes("xml")
  ) {
    return "code";
  }
  if (
    mime.includes("pdf") ||
    mime.includes("msword") ||
    mime.includes("wordprocessing") ||
    mime.includes("officedocument.word") ||
    mime.includes("epub") ||
    mime.includes("rtf")
  ) {
    return "document";
  }
  if (
    mime.includes("zip") ||
    mime.includes("x-tar") ||
    mime.includes("gzip") ||
    mime.includes("rar") ||
    mime.includes("7z")
  ) {
    return "archive";
  }
  if (
    mime.includes("csv") ||
    mime.includes("excel") ||
    mime.includes("spreadsheet") ||
    mime.includes("parquet") ||
    mime.includes("octet-stream") && mime.endsWith("db")
  ) {
    return "dataset";
  }
  return "other";
}

// Generate high-aesthetic theme colors for the file thumbnails matching the styling palette
function getThumbnailColor(kind: FileKind): string {
  switch (kind) {
    case "image":
      return "#4CB782"; // success green
    case "video":
      return "#E5484D"; // danger red
    case "audio":
      return "#F5A623"; // live amber
    case "document":
      return "#5E6AD2"; // accent purple-blue
    case "code":
      return "#5E9DD2"; // info light blue
    case "archive":
      return "#8A8F98"; // muted slate
    case "dataset":
      return "#10B981"; // emerald
    default:
      return "#5C5F66"; // faint gray
  }
}

// Map Go backend File schema to Next.js Frontend FileRecord
export function mapBackendFileToFrontend(f: any): FileRecord {
  const kind = determineFileKind(f.content_type || "");
  return {
    id: f.id,
    name: f.filename,
    kind,
    sizeBytes: parseInt(f.file_size || "0", 10),
    mimeType: f.content_type,
    providerId: (f.storage_provider as any) || "local",
    uploadedAt: f.created_at,
    updatedAt: f.updated_at,
    ownerName: "Me",
    ownerAvatar: "ME",
    checksum: f.storage_key || "",
    downloads: 0,
    shared: !!f.is_public,
    starred: false,
    tags: [],
    path: f.storage_key,
    thumbnailColor: getThumbnailColor(kind),
    status: f.status || "READY",
  };
}

export function useFiles() {
  return useQuery<FileRecord[]>({
    queryKey: ["files"],
    queryFn: async () => {
      const data = await apiClient("/api/v1/files");
      if (Array.isArray(data.files)) {
        return data.files.map(mapBackendFileToFrontend);
      }
      return [];
    },
  });
}

export function useFile(id: string) {
  return useQuery<FileRecord | null>({
    queryKey: ["files", id],
    queryFn: async () => {
      const data = await apiClient("/api/v1/files");
      if (Array.isArray(data.files)) {
        const file = data.files.find((f: any) => f.id === id);
        return file ? mapBackendFileToFrontend(file) : null;
      }
      return null;
    },
    enabled: !!id,
  });
}

export function useFileHistory(id: string) {
  return useQuery<any[]>({
    queryKey: ["files", id, "history"],
    queryFn: async () => {
      return [];
    },
    enabled: !!id,
  });
}

// Mutation to toggle public share link state
export function useToggleShareMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async ({ id, isPublic }: { id: string; isPublic: boolean }) => {
      return apiClient(`/api/v1/files/${id}/share`, {
        method: "PATCH",
        body: JSON.stringify({ is_public: isPublic }),
      });
    },
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({ queryKey: ["files"] });
      queryClient.invalidateQueries({ queryKey: ["files", variables.id] });
    },
  });
}

// Mutation to delete a file
export function useDeleteFileMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (id: string) => {
      return apiClient(`/api/v1/files/${id}`, {
        method: "DELETE",
      });
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["files"] });
    },
  });
}

// Direct-to-storage presigned URL upload flow (Version 1 Rule 1 & Rule 2 compliance)
export function useUploadFileMutation() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (file: File) => {
      // 1. Create upload session (backend inserts metadata as UPLOADING and returns signed URL)
      const session = await apiClient("/api/v1/files/upload-session", {
        method: "POST",
        body: JSON.stringify({
          filename: file.name,
          file_size: file.size,
          content_type: file.type || "application/octet-stream",
        }),
      });

      const { file_id, upload_url } = session;

      // 2. Direct upload raw bytes to presigned URL via PUT
      // IMPORTANT: DO NOT send Authorization or custom headers to signed R2/S3 URLs,
      // as they validate signatures via exact URL query strings. Passing auth token headers will mismatch R2.
      const uploadHeaders: Record<string, string> = {
        "Content-Type": file.type || "application/octet-stream",
      };

      const uploadResponse = await fetch(upload_url, {
        method: "PUT",
        headers: uploadHeaders,
        body: file,
      });

      if (!uploadResponse.ok) {
        throw new Error(`Direct upload to cloud storage failed! Status: ${uploadResponse.status}`);
      }

      // 3. Inform backend that raw bytes upload is complete (marks state as READY)
      await apiClient(`/api/v1/files/${file_id}/complete`, {
        method: "POST",
      });

      return file_id;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["files"] });
    },
  });
}
