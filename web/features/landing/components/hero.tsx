"use client";

import Link from "next/link";
import { ArrowRight } from "lucide-react";
import { Button } from "@/components/ui/button";

const liveChunks = Array.from({ length: 64 }, (_, i) => i);

export function Hero() {
  return (
    <section className="relative overflow-hidden border-b border-border">
      <div className="absolute inset-0 bg-grid-pattern bg-[size:42px_42px] [mask-image:radial-gradient(ellipse_60%_50%_at_50%_0%,black_40%,transparent_100%)]" />
      <div className="container relative pt-28 pb-20">
        <div className="mx-auto flex max-w-3xl flex-col items-center text-center">
          <div className="mb-6 inline-flex items-center gap-2 rounded-full border border-border-strong bg-bg-surface px-3 py-1 text-[12px] text-ink-muted">
            <span className="relative flex h-1.5 w-1.5">
              <span className="absolute inline-flex h-full w-full animate-pulse-live rounded-full bg-live opacity-75" />
              <span className="relative inline-flex h-1.5 w-1.5 rounded-full bg-live" />
            </span>
            Now streaming 14,200 chunks/sec across 3 providers
          </div>

          <h1 className="text-4xl font-semibold tracking-tight text-ink sm:text-5xl lg:text-6xl">
            File transfer, down to{" "}
            <span className="bg-gradient-to-r from-accent to-accent-bright bg-clip-text text-transparent">
              the chunk.
            </span>
          </h1>
          <p className="mt-5 max-w-xl text-[16px] leading-relaxed text-ink-muted">
            ByteVault resumes every upload exactly where it stopped, retries only the
            piece that failed, and shows you precisely which chunk is in flight —
            across R2, S3, or your own disks.
          </p>

          <div className="mt-8 flex items-center gap-3">
            <Button size="lg" asChild>
              <Link href="/dashboard">
                Open dashboard <ArrowRight className="h-4 w-4" />
              </Link>
            </Button>
            <Button size="lg" variant="secondary" asChild>
              <Link href="/transfers">See live transfers</Link>
            </Button>
          </div>
        </div>

        {/* Signature element: a live chunk grid standing in for an actual transfer */}
        <div className="mx-auto mt-16 max-w-2xl rounded-md border border-border bg-bg-surface p-5">
          <div className="mb-3 flex items-center justify-between font-mono text-[12px] text-ink-muted">
            <span>model-training-dataset.parquet</span>
            <span className="text-live">38.2 MB/s</span>
          </div>
          <div className="flex flex-wrap gap-[3px]">
            {liveChunks.map((i) => (
              <span
                key={i}
                className="h-3 w-3 rounded-[2px] bg-accent"
                style={{
                  opacity: i < 41 ? 1 : i === 41 ? undefined : 0.18,
                  animation: i === 41 ? "pulse-live 1.4s ease-in-out infinite" : undefined,
                  backgroundColor: i === 41 ? "#F5A623" : undefined,
                }}
              />
            ))}
          </div>
        </div>
      </div>
    </section>
  );
}
