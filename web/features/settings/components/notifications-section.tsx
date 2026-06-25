"use client";

import { Card } from "@/components/ui/card";
import { Switch } from "@/components/ui/switch";
import { useState } from "react";

const notifPrefs = [
  { id: "transfer-complete", label: "Transfer completed", desc: "Get notified when an upload or download finishes." },
  { id: "transfer-failed", label: "Transfer failed", desc: "Get notified when a transfer fails after all retries." },
  { id: "storage-threshold", label: "Storage threshold reached", desc: "Alert when a provider crosses 90% capacity." },
  { id: "shared-link-expiring", label: "Shared link expiring soon", desc: "Reminder 24 hours before a link expires." },
  { id: "weekly-digest", label: "Weekly digest", desc: "Summary of transfer volume and storage growth." },
];

export function NotificationsSection() {
  const [enabled, setEnabled] = useState<Record<string, boolean>>({
    "transfer-complete": true,
    "transfer-failed": true,
    "storage-threshold": true,
    "shared-link-expiring": false,
    "weekly-digest": true,
  });

  return (
    <Card className="max-w-xl divide-y divide-border p-0">
      {notifPrefs.map((pref) => (
        <div key={pref.id} className="flex items-center justify-between p-4">
          <div>
            <p className="text-[13px] font-medium text-ink">{pref.label}</p>
            <p className="mt-0.5 text-[12px] text-ink-muted">{pref.desc}</p>
          </div>
          <Switch
            checked={enabled[pref.id]}
            onCheckedChange={(v) => setEnabled((prev) => ({ ...prev, [pref.id]: v }))}
          />
        </div>
      ))}
    </Card>
  );
}
