"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import {
  LayoutDashboard,
  FolderClosed,
  ArrowUpDown,
  Database,
  Share2,
  BarChart3,
  Settings,
  Box,
} from "lucide-react";
import { cn } from "@/lib/utils";
import { transferStats } from "@/lib/mock";

const navItems = [
  { href: "/dashboard", label: "Dashboard", icon: LayoutDashboard },
  { href: "/files", label: "Files", icon: FolderClosed },
  { href: "/transfers", label: "Transfers", icon: ArrowUpDown, badge: transferStats.active },
  { href: "/storage", label: "Storage", icon: Database },
  { href: "/shared", label: "Shared links", icon: Share2 },
  { href: "/analytics", label: "Analytics", icon: BarChart3 },
];

export function Sidebar() {
  const pathname = usePathname();

  return (
    <aside className="fixed inset-y-0 left-0 z-40 hidden w-60 flex-col border-r border-border bg-bg-surface md:flex">
      <div className="flex h-14 items-center gap-2 border-b border-border px-5">
        <div className="flex h-6 w-6 items-center justify-center rounded-sm bg-accent">
          <Box className="h-3.5 w-3.5 text-white" />
        </div>
        <span className="text-[14px] font-semibold tracking-tight text-ink">ByteVault</span>
      </div>

      <nav className="flex-1 overflow-y-auto px-3 py-4">
        <p className="label-eyebrow px-2 pb-2">Workspace</p>
        <ul className="flex flex-col gap-0.5">
          {navItems.map((item) => {
            const isActive = pathname.startsWith(item.href);
            const Icon = item.icon;
            return (
              <li key={item.href}>
                <Link
                  href={item.href}
                  className={cn(
                    "flex items-center justify-between gap-2.5 rounded-sm px-2.5 py-1.5 text-[13px] font-medium transition-colors",
                    isActive
                      ? "bg-accent/10 text-accent-bright"
                      : "text-ink-muted hover:bg-bg-overlay hover:text-ink"
                  )}
                >
                  <span className="flex items-center gap-2.5">
                    <Icon className="h-[15px] w-[15px]" strokeWidth={2} />
                    {item.label}
                  </span>
                  {item.badge ? (
                    <span className="rounded-full bg-live/15 px-1.5 py-0.5 text-[10px] font-mono font-medium text-live">
                      {item.badge}
                    </span>
                  ) : null}
                </Link>
              </li>
            );
          })}
        </ul>

        <p className="label-eyebrow px-2 pb-2 pt-6">Account</p>
        <ul className="flex flex-col gap-0.5">
          <li>
            <Link
              href="/settings"
              className={cn(
                "flex items-center gap-2.5 rounded-sm px-2.5 py-1.5 text-[13px] font-medium transition-colors",
                pathname.startsWith("/settings")
                  ? "bg-accent/10 text-accent-bright"
                  : "text-ink-muted hover:bg-bg-overlay hover:text-ink"
              )}
            >
              <Settings className="h-[15px] w-[15px]" strokeWidth={2} />
              Settings
            </Link>
          </li>
        </ul>
      </nav>

      <div className="border-t border-border p-3">
        <div className="flex items-center gap-2 rounded-sm px-2 py-1.5">
          <span className="relative flex h-2 w-2">
            <span className="absolute inline-flex h-full w-full animate-pulse-live rounded-full bg-success opacity-75" />
            <span className="relative inline-flex h-2 w-2 rounded-full bg-success" />
          </span>
          <span className="font-mono text-[11px] text-ink-faint">All systems operational</span>
        </div>
      </div>
    </aside>
  );
}
