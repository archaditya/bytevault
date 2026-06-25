import { Metadata } from "next";
import { currentUser } from "@/lib/mock";
import { Card, CardContent } from "@/components/ui/card";
import { Avatar, AvatarFallback } from "@/components/ui/avatar";
import { Badge } from "@/components/ui/badge";
import { formatDate } from "@/lib/utils";
import { Mail, Shield, Calendar, KeyRound } from "lucide-react";

export const metadata: Metadata = {
  title: "Profile — ByteVault",
};

export default function ProfilePage() {
  const user = currentUser;

  return (
    <div className="flex max-w-2xl flex-col gap-4">
      <Card>
        <CardContent className="flex items-center gap-4">
          <Avatar className="h-16 w-16">
            <AvatarFallback className="text-lg">{user.avatar}</AvatarFallback>
          </Avatar>
          <div>
            <div className="flex items-center gap-2">
              <h2 className="text-lg font-semibold text-ink">{user.name}</h2>
              <Badge variant="default" className="capitalize">{user.plan}</Badge>
            </div>
            <p className="text-[13px] text-ink-muted">{user.role}</p>
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardContent className="grid grid-cols-1 gap-4 sm:grid-cols-2">
          <InfoRow icon={Mail} label="Email" value={user.email} />
          <InfoRow icon={Calendar} label="Member since" value={formatDate(user.joinedAt)} />
          <InfoRow icon={Shield} label="Two-factor auth" value={user.twoFactorEnabled ? "Enabled" : "Disabled"} />
          <InfoRow icon={KeyRound} label="API keys" value={`${user.apiKeysCount} active`} />
        </CardContent>
      </Card>
    </div>
  );
}

function InfoRow({ icon: Icon, label, value }: { icon: any; label: string; value: string }) {
  return (
    <div className="flex items-center gap-3">
      <div className="flex h-9 w-9 items-center justify-center rounded-sm bg-bg-overlay text-ink-muted">
        <Icon className="h-4 w-4" />
      </div>
      <div>
        <p className="text-[11px] uppercase tracking-wide text-ink-faint">{label}</p>
        <p className="text-[13px] font-medium text-ink">{value}</p>
      </div>
    </div>
  );
}
