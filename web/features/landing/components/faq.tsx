"use client";

import { useState } from "react";
import { Plus } from "lucide-react";
import { cn } from "@/lib/utils";

const faqs = [
  {
    q: "What happens if my connection drops mid-upload?",
    a: "ByteVault tracks completion at the chunk level, not the file level. When you reconnect, the transfer resumes from the last completed chunk — nothing already uploaded gets sent again.",
  },
  {
    q: "Can I move files between storage providers later?",
    a: "Yes. Files can be replicated or moved between Cloudflare R2, AWS S3, and local storage without re-uploading from your machine — ByteVault transfers provider-to-provider directly.",
  },
  {
    q: "How are failed chunks retried?",
    a: "Each chunk has its own retry counter and backoff schedule. A failure on one chunk never restarts the rest of the transfer; you can see retry counts per chunk in the transfer detail view.",
  },
  {
    q: "Is there a limit on file size?",
    a: "No hard limit from ByteVault. Practical limits come from your chosen storage provider's object size limits, which we surface before you start a transfer.",
  },
  {
    q: "How are shared links secured?",
    a: "Links can require a password, expire on a schedule, and cap total downloads. Revoking a link takes effect immediately and doesn't touch the underlying file.",
  },
];

export function FAQ() {
  const [openIndex, setOpenIndex] = useState<number | null>(0);

  return (
    <section className="border-b border-border py-20">
      <div className="container">
        <div className="mb-10 max-w-xl">
          <p className="label-eyebrow">Questions</p>
          <h2 className="mt-2 text-2xl font-semibold tracking-tight text-ink sm:text-3xl">
            Frequently asked
          </h2>
        </div>

        <div className="mx-auto max-w-2xl divide-y divide-border rounded-md border border-border bg-bg-surface">
          {faqs.map((item, i) => {
            const isOpen = openIndex === i;
            return (
              <div key={item.q} className="px-5">
                <button
                  onClick={() => setOpenIndex(isOpen ? null : i)}
                  className="flex w-full items-center justify-between gap-4 py-4 text-left text-[14px] font-medium text-ink focus-visible:outline-none"
                  aria-expanded={isOpen}
                >
                  {item.q}
                  <Plus
                    className={cn(
                      "h-4 w-4 shrink-0 text-ink-faint transition-transform duration-200",
                      isOpen && "rotate-45"
                    )}
                  />
                </button>
                <div
                  className={cn(
                    "overflow-hidden transition-all duration-200",
                    isOpen ? "max-h-40 pb-4 opacity-100" : "max-h-0 opacity-0"
                  )}
                >
                  <p className="text-[13px] leading-relaxed text-ink-muted">{item.a}</p>
                </div>
              </div>
            );
          })}
        </div>
      </div>
    </section>
  );
}
