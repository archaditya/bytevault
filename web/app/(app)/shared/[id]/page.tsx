import { notFound } from "next/navigation";
import Link from "next/link";
import { ChevronLeft, Lock, Calendar, Download, Eye, Link2 } from "lucide-react";
import { sharedLinks } from "@/lib/mock";
import { Card, CardHeader, CardTitle, CardContent } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { formatDate, formatBytes } from "@/lib/utils";
import { getFileById } from "@/lib/mock";

export function generateMetadata({ params }: { params: { id: string } }) {
  const link = sharedLinks.find((l) => l.id === params.id);
  return { title: link ? `${link.fileName} share — ByteVault` : "Share not found — ByteVault" };
}

export default function ShareDetailsPage({ params }: { params: { id: string } }) {
  const link = sharedLinks.find((l) => l.id === params.id);
  if (!link) return notFound();
  const file = getFileById(link.fileId);

  return (
    <div className="flex flex-col gap-6">
      <Link href="/shared" className="inline-flex items-center gap-1 text-[13px] text-ink-muted hover:text-ink">
        <ChevronLeft className="h-3.5 w-3.5" /> Back to shared links
      </Link>

      <div className="flex flex-wrap items-start justify-between gap-4">
        <div className="flex items-center gap-3">
          <div className="flex h-10 w-10 items-center justify-center rounded-md bg-accent/10 text-accent-bright">
            <Link2 className="h-5 w-5" />
          </div>
          <div>
            <h2 className="text-lg font-semibold text-ink">{link.fileName}</h2>
            <p className="mt-0.5 font-mono text-[12px] text-ink-muted">{link.url}</p>
          </div>
        </div>
        <Badge variant={link.active ? "success" : "muted"} dot>
          {link.active ? "Active" : "Disabled"}
        </Badge>
      </div>

      <div className="grid grid-cols-2 gap-3 sm:grid-cols-4">
        <Stat icon={Eye} label="Views" value={link.views.toLocaleString()} />
        <Stat icon={Download} label="Downloads" value={link.downloadCount.toLocaleString()} />
        <Stat icon={Calendar} label="Created" value={formatDate(link.createdAt)} />
        <Stat icon={Lock} label="Protected" value={link.passwordProtected ? "Yes" : "No"} />
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Access settings</CardTitle>
        </CardHeader>
        <CardContent className="flex flex-col gap-3">
          <Row label="Password protection" value={link.passwordProtected ? "Enabled" : "Disabled"} />
          <Row label="Expiration" value={link.expiresAt ? formatDate(link.expiresAt) : "Never expires"} />
          <Row
            label="Download limit"
            value={link.downloadLimit ? `${link.downloadCount} / ${link.downloadLimit}` : "Unlimited"}
          />
          {file && <Row label="File size" value={formatBytes(file.sizeBytes)} />}
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Analytics</CardTitle>
        </CardHeader>
        <CardContent>
          <p className="text-[13px] text-ink-muted">
            This link has been viewed {link.views} times and downloaded {link.downloadCount} times since creation.
          </p>
        </CardContent>
      </Card>

      <div className="flex gap-2">
        <Button variant="secondary" size="sm">Copy link</Button>
        <Button variant="danger" size="sm">Revoke link</Button>
      </div>
    </div>
  );
}

function Stat({ icon: Icon, label, value }: { icon: any; label: string; value: string }) {
  return (
    <div className="rounded-md border border-border bg-bg-surface p-3.5">
      <p className="flex items-center gap-1 text-[11px] uppercase tracking-wide text-ink-faint">
        <Icon className="h-3 w-3" /> {label}
      </p>
      <p className="mt-1.5 font-mono text-[15px] font-medium text-ink">{value}</p>
    </div>
  );
}

function Row({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex items-center justify-between text-[13px]">
      <span className="text-ink-muted">{label}</span>
      <span className="font-medium text-ink">{value}</span>
    </div>
  );
}
