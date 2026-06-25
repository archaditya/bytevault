import { cn } from "@/lib/utils";

const providerConfig: Record<string, { label: string; color: string }> = {
  r2: { label: "Cloudflare R2", color: "#F38020" },
  s3: { label: "AWS S3", color: "#FF9900" },
  local: { label: "Local Storage", color: "#5E9DD2" },
};

export function ProviderTag({ providerId, className }: { providerId: string; className?: string }) {
  const config = providerConfig[providerId] ?? { label: providerId, color: "#5C5F66" };
  return (
    <span className={cn("inline-flex items-center gap-1.5 text-[12px] text-ink-muted", className)}>
      <span className="h-1.5 w-1.5 rounded-full" style={{ backgroundColor: config.color }} />
      {config.label}
    </span>
  );
}
