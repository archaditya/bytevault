import { useQuery } from "@tanstack/react-query";
import { transferSessions, getTransferById, transferStats } from "@/lib/mock";

const FAKE_LATENCY = 250;

function delay<T>(value: T, ms = FAKE_LATENCY): Promise<T> {
  return new Promise((resolve) => setTimeout(() => resolve(value), ms));
}

export function useTransfers() {
  return useQuery({
    queryKey: ["transfers"],
    queryFn: () => delay(transferSessions),
    refetchInterval: 4000,
  });
}

export function useTransfer(id: string) {
  return useQuery({
    queryKey: ["transfers", id],
    queryFn: () => delay(getTransferById(id) ?? null),
    enabled: !!id,
    refetchInterval: 4000,
  });
}

export function useTransferStats() {
  return useQuery({
    queryKey: ["transfers", "stats"],
    queryFn: () => delay(transferStats),
    refetchInterval: 4000,
  });
}
