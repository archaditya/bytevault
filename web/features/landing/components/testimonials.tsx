const testimonials = [
  {
    quote:
      "We moved our model training datasets off a homegrown rsync script. Retries that used to mean re-uploading 400GB now resolve in seconds.",
    name: "Priya Sharma",
    role: "Platform Engineering Lead",
  },
  {
    quote:
      "The chunk visualizer sounds like a gimmick until your transfer fails at 94% and you can see exactly which piece to retry.",
    name: "Daniel Cohen",
    role: "Site Reliability Engineer",
  },
  {
    quote:
      "Switching our backups from S3 to local storage took an afternoon, not a migration project. No re-uploads required.",
    name: "Hana Kobayashi",
    role: "Infrastructure Architect",
  },
];

export function Testimonials() {
  return (
    <section className="border-b border-border py-20">
      <div className="container">
        <div className="mb-12 max-w-xl">
          <p className="label-eyebrow">From engineering teams</p>
          <h2 className="mt-2 text-2xl font-semibold tracking-tight text-ink sm:text-3xl">
            Trusted by teams who move a lot of data.
          </h2>
        </div>
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
          {testimonials.map((t) => (
            <div key={t.name} className="flex flex-col justify-between rounded-md border border-border bg-bg-surface p-6">
              <p className="text-[13px] leading-relaxed text-ink">&ldquo;{t.quote}&rdquo;</p>
              <div className="mt-5">
                <p className="text-[13px] font-medium text-ink">{t.name}</p>
                <p className="text-[12px] text-ink-muted">{t.role}</p>
              </div>
            </div>
          ))}
        </div>
      </div>
    </section>
  );
}
