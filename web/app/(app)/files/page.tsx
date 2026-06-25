import { Metadata } from "next";
import { FilesExplorer } from "@/features/files/components/files-explorer";

export const metadata: Metadata = {
  title: "Files — ByteVault",
};

export default function FilesPage() {
  return <FilesExplorer />;
}
