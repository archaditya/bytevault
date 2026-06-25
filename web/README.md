# ByteVault

A high-performance file transfer and storage platform — resumable uploads, chunk-level retries, and multi-provider storage routing across Cloudflare R2, AWS S3, and local disk.

This is not a Drive/Dropbox clone. It's an engineering-focused console for **transfer sessions**, not file collaboration.

## Stack

- **Next.js 15** (App Router) + **TypeScript**
- **Tailwind CSS** with a custom dark-theme token system
- **shadcn/ui**-style primitives built on **Radix UI**
- **Zustand** for client state (filters, view modes, tabs)
- **React Query** for data-fetching simulation
- **Recharts** for analytics/speed graphs
- **Lucide** icons

## Getting started

```bash
npm install
npm run dev
```

Open [http://localhost:3000](http://localhost:3000).

## Project structure

```
app/                    # App Router routes
  page.tsx              # Landing page
  (app)/                # Authenticated app shell (sidebar + navbar)
    dashboard/
    files/[id]/
    transfers/[id]/
    storage/providers/  storage/usage/
    shared/[id]/
    analytics/
    settings/
    profile/
components/
  ui/                   # shadcn-style primitives (Button, Card, Dialog, ...)
  layout/               # Sidebar, Navbar, AppShell
  shared/               # ChunkVisualizer, StatusBadge, StatCard, EmptyState...
features/               # Feature-scoped components, grouped by domain
  dashboard/ files/ transfers/ storage/ shared/ analytics/ settings/ landing/
hooks/                  # (reserved for shared hooks)
lib/
  mock/                 # Deterministic, seeded mock data generators
  utils.ts  random.ts  query-provider.tsx
services/               # React Query hooks per domain
store/                  # Zustand stores per domain
types/                  # Shared TypeScript types
```

## Notable pieces

- **ChunkVisualizer** (`components/shared/chunk-visualizer.tsx`) — the app's signature
  visualization. Every transfer renders as a grid of individually-tracked chunks
  (pending / uploading / retrying / failed / complete) instead of a single progress bar.
- **Mock data** is seeded with a deterministic PRNG (`lib/random.ts`) so server and client
  renders match exactly — no hydration mismatches, and the dataset is stable across reloads.
- **Services** wrap the mock data in React Query hooks with simulated latency, so swapping
  in real API calls later is a matter of changing `services/*.service.ts` only.

## Notes on fonts

The design spec calls for a Geist-style pairing (grotesk sans + matching mono for all
numeric data). This build uses `next/font/google` with **Inter** + **JetBrains Mono** to
avoid an extra dependency. Swap in the `geist` package's `GeistSans`/`GeistMono` in
`app/layout.tsx` if you want the exact Vercel typeface.

## Known limitations

- All data is mocked client-side; there is no backend or persistence.
- "Upload" and transfer-control buttons (pause/resume/retry/cancel) are UI affordances
  without wired-up mutations — hook them into real endpoints via `services/`.
