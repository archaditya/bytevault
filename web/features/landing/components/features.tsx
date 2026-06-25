import { RefreshCw, GitCommitHorizontal, Gauge, ShieldCheck, Share2, BarChart3 } from "lucide-react";

const features = [
  {
    icon: RefreshCw,
    title: "Resumable by design",
    desc: "Every upload is chunked and checksummed. Drop your connection mid-transfer and ByteVault picks up at the last completed chunk — never the start.",
  },
  {
    icon: GitCommitHorizontal,
    title: "Chunk-level retries",
    desc: "When one piece fails, only that piece retries. Watch retry counts climb on individual chunks instead of restarting the whole file.",
  },
  {
    icon: Gauge,
    title: "Live speed telemetry",
    desc: "Real-time throughput, ETA, and per-session speed graphs so you know exactly when a 40GB dataset will actually land.",
  },
  {
    icon: ShieldCheck,
    title: "Provider-aware routing",
    desc: "Send files to Cloudflare R2, AWS S3, or local disk based on latency, cost, or your own policy — switch providers without re-uploading.",
  },
  {
    icon: Share2,
    title: "Controlled sharing",
    desc: "Password-protect links, cap download counts, set expirations. Revoke access instantly without touching the underlying file.",
  },
  {
    icon: BarChart3,
    title: "Transfer analytics",
    desc: "Upload and download trends, success rates, and provider comparisons — the operational picture your infra team actually needs.",
  },
];

export function Features() {
  return (
    <section className="border-b border-border py-20">
      <div className="container">
        <div className="mb-12 max-w-xl">
          <p className="label-eyebrow">Built for transfer, not storage</p>
          <h2 className="mt-2 text-2xl font-semibold tracking-tight text-ink sm:text-3xl">
            Every layer is visible, all the way down to one chunk.
          </h2>
        </div>
        <div className="grid grid-cols-1 gap-px overflow-hidden rounded-md border border-border bg-border sm:grid-cols-2 lg:grid-cols-3">
          {features.map((f) => {
            const Icon = f.icon;
            return (
              <div key={f.title} className="bg-bg-surface p-6">
                <Icon className="h-5 w-5 text-accent-bright" strokeWidth={1.75} />
                <h3 className="mt-4 text-[15px] font-medium text-ink">{f.title}</h3>
                <p className="mt-2 text-[13px] leading-relaxed text-ink-muted">{f.desc}</p>
              </div>
            );
          })}
        </div>
      </div>
    </section>
  );
}
