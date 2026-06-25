"use client";

import { Search, Upload, Bell } from "lucide-react";
import { usePathname } from "next/navigation";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { Avatar, AvatarFallback } from "@/components/ui/avatar";
import { useAuthStore } from "@/store";

const titleMap: Record<string, string> = {
  "/dashboard": "Dashboard",
  "/files": "Files",
  "/transfers": "Transfers",
  "/storage": "Storage providers",
  "/shared": "Shared links",
  "/analytics": "Analytics",
  "/settings": "Settings",
  "/profile": "Profile",
};

export function Navbar() {
  const pathname = usePathname();
  const user = useAuthStore((s) => s.user);

  const title =
    Object.entries(titleMap).find(([path]) => pathname.startsWith(path))?.[1] ?? "ByteVault";

  return (
    <header className="sticky top-0 z-30 flex h-14 items-center gap-4 border-b border-border bg-bg/95 px-4 backdrop-blur-sm md:px-6">
      <h1 className="text-[15px] font-semibold text-ink whitespace-nowrap">{title}</h1>

      <div className="hidden flex-1 items-center lg:flex">
        <div className="relative w-full max-w-xs">
          <Search className="pointer-events-none absolute left-2.5 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-ink-faint" />
          <Input placeholder="Search files, transfers..." className="pl-8 bg-bg-surface" />
        </div>
      </div>

      <div className="ml-auto flex items-center gap-2">
        <Button size="sm" variant="primary" className="hidden sm:inline-flex">
          <Upload className="h-3.5 w-3.5" />
          Upload
        </Button>
        <Button size="icon" variant="ghost" aria-label="Notifications">
          <Bell className="h-4 w-4" />
        </Button>
        <Avatar>
          <AvatarFallback>{user.avatar}</AvatarFallback>
        </Avatar>
      </div>
    </header>
  );
}
