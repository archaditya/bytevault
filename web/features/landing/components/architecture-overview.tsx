const layers = [
  { name: "Client SDK", detail: "Chunked uploads, 8MB segments, 4 parallel streams, exponential backoff" },
  { name: "Transfer Engine", detail: "Session orchestration, chunk scheduling, checksum verification" },
  { name: "Provider Adapters", detail: "Unified interface across R2, S3, and local disk, swap without re-upload" },
  { name: "Storage Layer", detail: "Cloudflare R2 · AWS S3 · On-prem NVMe cluster" },
];

export function ArchitectureOverview() {
  return (
    <section className="border-b border-border py-20">
      <div className="container">
        <div className="mb-12 max-w-xl">
          <p className="label-eyebrow">How a file actually moves</p>
          <h2 className="mt-2 text-2xl font-semibold tracking-tight text-ink sm:text-3xl">
            Four layers, one transfer.
          </h2>
        </div>

        <div className="flex flex-col">
          {layers.map((layer, i) => (
            <div key={layer.name} className="flex gap-5">
              <div className="flex flex-col items-center">
                <span className="font-mono text-[11px] text-ink-faint">{String(i + 1).padStart(2, "0")}</span>
                <span className="mt-2 h-2 w-2 rounded-full bg-accent" />
                {i < layers.length - 1 && <span className="mt-2 w-px flex-1 bg-border" />}
              </div>
              <div className="pb-10">
                <h3 className="text-[15px] font-medium text-ink">{layer.name}</h3>
                <p className="mt-1 max-w-md text-[13px] text-ink-muted">{layer.detail}</p>
              </div>
            </div>
          ))}
        </div>
      </div>
    </section>
  );
}
