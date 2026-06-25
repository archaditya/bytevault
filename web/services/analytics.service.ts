import { useQuery } from "@tanstack/react-query";
import { analyticsHistory, providerComparison, sharedLinks, apiKeys } from "@/lib/mock";

function delay<T>(value: T, ms = 250): Promise<T> {
  return new Promise((resolve) => setTimeout(() => resolve(value), ms));
}

export function useAnalyticsHistory() {
  return useQuery({
    queryKey: ["analytics", "history"],
    queryFn: () => delay(analyticsHistory),
  });
}

export function useProviderComparison() {
  return useQuery({
    queryKey: ["analytics", "provider-comparison"],
    queryFn: () => delay(providerComparison),
  });
}

export function useSharedLinks() {
  return useQuery({
    queryKey: ["shared-links"],
    queryFn: () => delay(sharedLinks),
  });
}

export function useApiKeys() {
  return useQuery({
    queryKey: ["api-keys"],
    queryFn: () => delay(apiKeys),
  });
}
