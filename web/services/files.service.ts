import { useQuery } from "@tanstack/react-query";
import { files, getFileById, getFileHistory } from "@/lib/mock";

const FAKE_LATENCY = 280;

function delay<T>(value: T, ms = FAKE_LATENCY): Promise<T> {
  return new Promise((resolve) => setTimeout(() => resolve(value), ms));
}

export function useFiles() {
  return useQuery({
    queryKey: ["files"],
    queryFn: () => delay(files),
  });
}

export function useFile(id: string) {
  return useQuery({
    queryKey: ["files", id],
    queryFn: () => delay(getFileById(id) ?? null),
    enabled: !!id,
  });
}

export function useFileHistory(id: string) {
  return useQuery({
    queryKey: ["files", id, "history"],
    queryFn: () => delay(getFileHistory(id)),
    enabled: !!id,
  });
}
