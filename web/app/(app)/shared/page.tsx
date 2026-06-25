import { Metadata } from "next";
import { sharedLinks } from "@/lib/mock";
import { ShareLinkCard } from "@/features/shared/components/share-link-card";
import { Button } from "@/components/ui/button";
import { Plus } from "lucide-react";

export const metadata: Metadata = {
  title: "Shared links — ByteVault",
};

export default function SharedLinksPage() {
  return (
    <div className="flex flex-col gap-4">
      <div className="flex items-center justify-between">
        <p className="text-[13px] text-ink-muted">
          {sharedLinks.filter((l) => l.active).length} active links out of {sharedLinks.length} total.
        </p>
        <Button size="sm">
          <Plus className="h-3.5 w-3.5" /> Create link
        </Button>
      </div>
      <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-3">
        {sharedLinks.map((link) => (
          <ShareLinkCard key={link.id} link={link} />
        ))}
      </div>
    </div>
  );
}
