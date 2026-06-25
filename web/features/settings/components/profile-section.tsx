"use client";

import { Card, CardContent } from "@/components/ui/card";
import { Label } from "@/components/ui/label";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { Avatar, AvatarFallback } from "@/components/ui/avatar";
import { useAuthStore } from "@/store";

export function ProfileSection() {
  const user = useAuthStore((s) => s.user);

  return (
    <Card className="max-w-xl">
      <CardContent className="flex flex-col gap-5">
        <div className="flex items-center gap-4">
          <Avatar className="h-14 w-14">
            <AvatarFallback className="text-base">{user.avatar}</AvatarFallback>
          </Avatar>
          <div>
            <Button size="sm" variant="secondary">Change avatar</Button>
          </div>
        </div>

        <div className="grid grid-cols-2 gap-4">
          <div className="flex flex-col gap-1.5">
            <Label htmlFor="name">Full name</Label>
            <Input id="name" defaultValue={user.name} />
          </div>
          <div className="flex flex-col gap-1.5">
            <Label htmlFor="role">Role</Label>
            <Input id="role" defaultValue={user.role} />
          </div>
        </div>

        <div className="flex flex-col gap-1.5">
          <Label htmlFor="email">Email address</Label>
          <Input id="email" type="email" defaultValue={user.email} />
        </div>

        <div className="flex justify-end pt-2">
          <Button>Save changes</Button>
        </div>
      </CardContent>
    </Card>
  );
}
