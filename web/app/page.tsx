import { LandingNav } from "@/features/landing/components/landing-nav";
import { Hero } from "@/features/landing/components/hero";
import { Features } from "@/features/landing/components/features";
import { ArchitectureOverview } from "@/features/landing/components/architecture-overview";
import { TransferEngineShowcase } from "@/features/landing/components/transfer-engine-showcase";
import { StorageProvidersShowcase } from "@/features/landing/components/storage-providers-showcase";
import { Testimonials } from "@/features/landing/components/testimonials";
import { FAQ } from "@/features/landing/components/faq";
import { CTA } from "@/features/landing/components/cta";
import { Footer } from "@/features/landing/components/footer";

export default function HomePage() {
  return (
    <div className="min-h-screen bg-bg">
      <LandingNav />
      <Hero />
      <div id="features">
        <Features />
      </div>
      <div id="architecture">
        <ArchitectureOverview />
      </div>
      <TransferEngineShowcase />
      <StorageProvidersShowcase />
      <Testimonials />
      <div id="faq">
        <FAQ />
      </div>
      <CTA />
      <Footer />
    </div>
  );
}
