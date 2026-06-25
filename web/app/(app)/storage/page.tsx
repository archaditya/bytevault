import { Metadata } from "next";
import Link from "next/link";
import { storageProviders } from "@/lib/mock";
import { ProviderCard } from "@/features/storage/components/provider-card";
import { Button } from "@/components/ui/button";
import { ArrowRight, Settings2 } from "lucide-react";

export const metadata: Metadata = {
  title: "Storage providers — ByteVault",
};

export default function StoragePage() {
  return (
    <div className="flex flex-col gap-6">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <p className="max-w-lg text-[13px] text-ink-muted">
          ByteVault routes uploads across these providers based on latency, cost, and
          your storage preferences.
        </p>
        <div className="flex gap-2">
          <Button size="sm" variant="secondary" asChild>
            <Link href="/storage/usage">
              Usage breakdown <ArrowRight className="h-3.5 w-3.5" />
            </Link>
          </Button>
          <Button size="sm" variant="secondary" asChild>
            <Link href="/storage/providers">
              <Settings2 className="h-3.5 w-3.5" /> Manage providers
            </Link>
          </Button>
        </div>
      </div>

      <div className="grid grid-cols-1 gap-4 lg:grid-cols-3">
        {storageProviders.map((provider) => (
          <ProviderCard key={provider.id} provider={provider} />
        ))}
      </div>
    </div>
  );
}
