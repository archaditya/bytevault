"use client";

import { storageProviders } from "@/lib/mock";
import { Card, CardContent } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";

export function StoragePreferencesSection() {
  return (
    <div className="flex max-w-xl flex-col gap-3">
      <p className="text-[13px] text-ink-muted">
        Choose which provider new uploads default to. You can override this per-upload.
      </p>
      {storageProviders.map((p) => (
        <Card key={p.id}>
          <CardContent className="flex items-center justify-between">
            <div className="flex items-center gap-3">
              <div>
                <p className="text-[13px] font-medium text-ink">{p.name}</p>
                <p className="text-[12px] text-ink-muted">{p.region}</p>
              </div>
              {p.isDefault && <Badge variant="default">Default</Badge>}
            </div>
            <Button size="sm" variant={p.isDefault ? "secondary" : "outline"} disabled={p.isDefault}>
              {p.isDefault ? "Current default" : "Set as default"}
            </Button>
          </CardContent>
        </Card>
      ))}
    </div>
  );
}
