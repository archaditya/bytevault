const providers = [
  { name: "Cloudflare R2", detail: "Zero egress fees, global anycast", latency: "38ms" },
  { name: "AWS S3", detail: "Industry-standard durability, deep ecosystem", latency: "61ms" },
  { name: "Local Storage", detail: "Your own disks, zero cloud cost", latency: "4ms" },
];

export function StorageProvidersShowcase() {
  return (
    <section className="border-b border-border py-20">
      <div className="container">
        <div className="mb-12 max-w-xl">
          <p className="label-eyebrow">Bring your own storage</p>
          <h2 className="mt-2 text-2xl font-semibold tracking-tight text-ink sm:text-3xl">
            Three providers. One transfer engine.
          </h2>
        </div>
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
          {providers.map((p) => (
            <div key={p.name} className="rounded-md border border-border bg-bg-surface p-6">
              <h3 className="text-[15px] font-medium text-ink">{p.name}</h3>
              <p className="mt-2 text-[13px] text-ink-muted">{p.detail}</p>
              <p className="mt-4 font-mono text-[12px] text-ink-faint">~{p.latency} avg latency</p>
            </div>
          ))}
        </div>
      </div>
    </section>
  );
}
