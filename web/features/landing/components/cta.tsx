import Link from "next/link";
import { ArrowRight } from "lucide-react";
import { Button } from "@/components/ui/button";

export function CTA() {
  return (
    <section className="py-24">
      <div className="container">
        <div className="relative overflow-hidden rounded-lg border border-border bg-bg-surface px-8 py-16 text-center">
          <div className="absolute inset-0 bg-grid-pattern bg-[size:36px_36px] [mask-image:radial-gradient(ellipse_70%_70%_at_50%_50%,black_0%,transparent_100%)] opacity-60" />
          <div className="relative">
            <h2 className="text-2xl font-semibold tracking-tight text-ink sm:text-3xl">
              Start moving files the resumable way.
            </h2>
            <p className="mx-auto mt-3 max-w-md text-[14px] text-ink-muted">
              No credit card. Connect a storage provider and run your first transfer in minutes.
            </p>
            <div className="mt-7 flex justify-center">
              <Button size="lg" asChild>
                <Link href="/dashboard">
                  Open dashboard <ArrowRight className="h-4 w-4" />
                </Link>
              </Button>
            </div>
          </div>
        </div>
      </div>
    </section>
  );
}
