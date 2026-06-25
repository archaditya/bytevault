"use client";

import { useRef } from "react";
import { Search, Upload, Bell, Loader2 } from "lucide-react";
import { usePathname } from "next/navigation";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { Avatar, AvatarFallback } from "@/components/ui/avatar";
import { useAuthStore } from "@/store";
import { useUploadFileMutation } from "@/services";

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
  const fileInputRef = useRef<HTMLInputElement>(null);
  const uploadMutation = useUploadFileMutation();
  const pathname = usePathname();
  const user = useAuthStore((s) => s.user);

  const title =
    Object.entries(titleMap).find(([path]) => pathname.startsWith(path))?.[1] ?? "ByteVault";

  const handleUploadClick = () => {
    fileInputRef.current?.click();
  };

  const handleFileChange = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const selectedFile = e.target.files?.[0];
    if (!selectedFile) return;

    try {
      await uploadMutation.mutateAsync(selectedFile);
    } catch (err) {
      console.error("Upload failed:", err);
    } finally {
      if (fileInputRef.current) {
        fileInputRef.current.value = "";
      }
    }
  };

  return (
    <header className="sticky top-0 z-30 flex h-14 items-center gap-4 border-b border-border bg-bg/95 px-4 backdrop-blur-sm md:px-6">
      {/* Invisible File Input */}
      <input
        type="file"
        ref={fileInputRef}
        onChange={handleFileChange}
        className="hidden"
      />

      <h1 className="text-[15px] font-semibold text-ink whitespace-nowrap">{title}</h1>

      <div className="hidden flex-1 items-center lg:flex">
        <div className="relative w-full max-w-xs">
          <Search className="pointer-events-none absolute left-2.5 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-ink-faint" />
          <Input placeholder="Search files, transfers..." className="pl-8 bg-bg-surface" />
        </div>
      </div>

      <div className="ml-auto flex items-center gap-2">
        <Button
          size="sm"
          variant="primary"
          className="hidden sm:inline-flex"
          onClick={handleUploadClick}
          disabled={uploadMutation.isPending}
        >
          {uploadMutation.isPending ? (
            <Loader2 className="h-3.5 w-3.5 animate-spin" />
          ) : (
            <Upload className="h-3.5 w-3.5" />
          )}
          {uploadMutation.isPending ? "Uploading..." : "Upload"}
        </Button>
        <Button size="icon" variant="ghost" aria-label="Notifications">
          <Bell className="h-4 w-4" />
        </Button>
        <Avatar>
          <AvatarFallback>{user?.avatar || "ME"}</AvatarFallback>
        </Avatar>
      </div>
    </header>
  );
}