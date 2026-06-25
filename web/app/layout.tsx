import type { Metadata } from "next";
import { Inter, JetBrains_Mono } from "next/font/google";
import "./globals.css";
import { QueryProvider } from "@/lib/query-provider";
import { TooltipProvider } from "@/components/ui/tooltip";

// Using Inter + JetBrains Mono as the concrete font pairing (Inter approximates Geist Sans'
// grotesk proportions; JetBrains Mono fills the Geist Mono role for all numeric/data display).
// Swap for the `geist` package's GeistSans/GeistMono if you prefer the exact Vercel typeface.
const fontSans = Inter({
  subsets: ["latin"],
  variable: "--font-geist-sans",
  display: "swap",
});

const fontMono = JetBrains_Mono({
  subsets: ["latin"],
  variable: "--font-geist-mono",
  display: "swap",
});

export const metadata: Metadata = {
  title: "ByteVault — Resumable file transfer, down to the chunk",
  description:
    "ByteVault is a high-performance file transfer and storage platform with resumable uploads, chunk-level retries, and multi-provider storage routing across Cloudflare R2, AWS S3, and local disk.",
  keywords: ["file transfer", "resumable upload", "storage", "Cloudflare R2", "AWS S3"],
  openGraph: {
    title: "ByteVault",
    description: "Resumable file transfer, down to the chunk.",
    type: "website",
  },
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en" className={`${fontSans.variable} ${fontMono.variable}`}>
      <body className="bg-bg text-ink font-sans antialiased">
        <QueryProvider>
          <TooltipProvider>{children}</TooltipProvider>
        </QueryProvider>
      </body>
    </html>
  );
}
