"use client";

const sampleChunks = Array.from({ length: 96 }, (_, i) => {
  if (i < 58) return "complete";
  if (i === 58) return "uploading";
  if (i === 59) return "retrying";
  if (i === 71) return "failed";
  return "pending";
});

const chunkColor: Record<string, string> = {
  complete: "#5E6AD2",
  uploading: "#F5A623",
  retrying: "#F5A623",
  failed: "#E5484D",
  pending: "#1F1F23",
};

export function TransferEngineShowcase() {
  return (
    <section className="border-b border-border py-20">
      <div className="container grid grid-cols-1 gap-10 lg:grid-cols-2 lg:items-center">
        <div>
          <p className="label-eyebrow">The transfer engine</p>
          <h2 className="mt-2 text-2xl font-semibold tracking-tight text-ink sm:text-3xl">
            See the chunk that's actually moving.
          </h2>
          <p className="mt-4 max-w-md text-[14px] leading-relaxed text-ink-muted">
            Most tools show you a percentage. ByteVault shows you the grid: which
            chunks landed, which one is in flight, which one is retrying after a
            timeout, and which one needs your attention.
          </p>
          <div className="mt-6 flex flex-wrap gap-x-6 gap-y-2 text-[13px]">
            <span className="flex items-center gap-2 text-ink-muted">
              <span className="h-2 w-2 rounded-[2px]" style={{ backgroundColor: chunkColor.complete }} /> Complete
            </span>
            <span className="flex items-center gap-2 text-ink-muted">
              <span className="h-2 w-2 rounded-[2px]" style={{ backgroundColor: chunkColor.uploading }} /> In flight
            </span>
            <span className="flex items-center gap-2 text-ink-muted">
              <span className="h-2 w-2 rounded-[2px]" style={{ backgroundColor: chunkColor.failed }} /> Failed
            </span>
            <span className="flex items-center gap-2 text-ink-muted">
              <span className="h-2 w-2 rounded-[2px]" style={{ backgroundColor: chunkColor.pending }} /> Pending
            </span>
          </div>
        </div>

        <div className="rounded-md border border-border bg-bg-surface p-5">
          <div className="mb-3 flex items-center justify-between font-mono text-[12px] text-ink-muted">
            <span>warehouse-events-export.csv</span>
            <span>768 MB / 1.2 GB</span>
          </div>
          <div className="flex flex-wrap gap-[3px]">
            {sampleChunks.map((status, i) => (
              <span
                key={i}
                className="h-2.5 w-2.5 rounded-[2px]"
                style={{
                  backgroundColor: chunkColor[status],
                  animation: status === "uploading" || status === "retrying" ? "pulse-live 1.4s ease-in-out infinite" : undefined,
                }}
              />
            ))}
          </div>
        </div>
      </div>
    </section>
  );
}
