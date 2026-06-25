import { useQuery } from "@tanstack/react-query";
import { storageProviders, getProviderById } from "@/lib/mock";

function delay<T>(value: T, ms = 250): Promise<T> {
  return new Promise((resolve) => setTimeout(() => resolve(value), ms));
}

export function useStorageProviders() {
  return useQuery({
    queryKey: ["storage-providers"],
    queryFn: () => delay(storageProviders),
  });
}

export function useStorageProvider(id: string) {
  return useQuery({
    queryKey: ["storage-providers", id],
    queryFn: () => delay(getProviderById(id) ?? null),
    enabled: !!id,
  });
}
