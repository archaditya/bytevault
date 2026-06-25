"use client";

import { Card, CardContent } from "@/components/ui/card";
import { Label } from "@/components/ui/label";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { Switch } from "@/components/ui/switch";
import { useAuthStore } from "@/store";

export function SecuritySection() {
  const user = useAuthStore((s) => s.user);

  return (
    <div className="flex max-w-xl flex-col gap-4">
      <Card>
        <CardContent className="flex flex-col gap-4">
          <h3 className="text-[14px] font-medium text-ink">Change password</h3>
          <div className="flex flex-col gap-1.5">
            <Label htmlFor="current-password">Current password</Label>
            <Input id="current-password" type="password" />
          </div>
          <div className="flex flex-col gap-1.5">
            <Label htmlFor="new-password">New password</Label>
            <Input id="new-password" type="password" />
          </div>
          <div className="flex justify-end">
            <Button size="sm">Update password</Button>
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardContent className="flex items-center justify-between">
          <div>
            <h3 className="text-[14px] font-medium text-ink">Two-factor authentication</h3>
            <p className="mt-1 text-[13px] text-ink-muted">
              Require a verification code in addition to your password.
            </p>
          </div>
          <Switch defaultChecked={user.twoFactorEnabled} />
        </CardContent>
      </Card>
    </div>
  );
}
