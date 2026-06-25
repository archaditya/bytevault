import { Metadata } from "next";
import { TransfersList } from "@/features/transfers/components/transfers-list";

export const metadata: Metadata = {
  title: "Transfers — ByteVault",
};

export default function TransfersPage() {
  return <TransfersList />;
}
