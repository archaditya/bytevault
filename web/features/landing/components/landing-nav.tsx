import Link from "next/link";
import { Box } from "lucide-react";
import { Button } from "@/components/ui/button";

export function LandingNav() {
  return (
    <header className="sticky top-0 z-40 border-b border-border bg-bg/90 backdrop-blur-md">
      <div className="container flex h-14 items-center justify-between">
        <Link href="/" className="flex items-center gap-2">
          <div className="flex h-6 w-6 items-center justify-center rounded-sm bg-accent">
            <Box className="h-3.5 w-3.5 text-white" />
          </div>
          <span className="text-[14px] font-semibold tracking-tight text-ink">ByteVault</span>
        </Link>
        <nav className="hidden items-center gap-7 text-[13px] text-ink-muted md:flex">
          <Link href="#features" className="hover:text-ink">Features</Link>
          <Link href="#architecture" className="hover:text-ink">Architecture</Link>
          <Link href="#faq" className="hover:text-ink">FAQ</Link>
        </nav>
        <div className="flex items-center gap-2">
          <Button size="sm" variant="ghost" asChild>
            <Link href="/dashboard">Sign in</Link>
          </Button>
          <Button size="sm" asChild>
            <Link href="/dashboard">Get started</Link>
          </Button>
        </div>
      </div>
    </header>
  );
}
