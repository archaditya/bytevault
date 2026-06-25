import Link from "next/link";
import { Box } from "lucide-react";
import { Button } from "@/components/ui/button";

export default function NotFound() {
  return (
    <div className="flex min-h-screen flex-col items-center justify-center gap-4 bg-bg px-4 text-center">
      <div className="flex h-10 w-10 items-center justify-center rounded-md bg-accent">
        <Box className="h-5 w-5 text-white" />
      </div>
      <h1 className="font-mono text-5xl font-semibold text-ink">404</h1>
      <p className="max-w-sm text-[14px] text-ink-muted">
        This chunk of the app couldn&apos;t be located. It may have been moved, renamed, or never existed.
      </p>
      <Button asChild>
        <Link href="/dashboard">Back to dashboard</Link>
      </Button>
    </div>
  );
}
