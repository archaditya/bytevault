import {
  FileText,
  Image as ImageIcon,
  Video,
  Music,
  Archive,
  Code,
  Database,
  File,
} from "lucide-react";
import { FileKind } from "@/types";
import { cn } from "@/lib/utils";

const iconMap: Record<FileKind, typeof File> = {
  document: FileText,
  image: ImageIcon,
  video: Video,
  audio: Music,
  archive: Archive,
  code: Code,
  dataset: Database,
  other: File,
};

export function FileKindIcon({ kind, className }: { kind: FileKind; className?: string }) {
  const Icon = iconMap[kind];
  return <Icon className={cn("h-4 w-4", className)} strokeWidth={1.75} />;
}
