"use client";

import { useState } from "react";
import { ProfileSection } from "./profile-section";
import { SecuritySection } from "./security-section";
import { ApiKeysSection } from "./api-keys-section";
import { StoragePreferencesSection } from "./storage-preferences-section";
import { NotificationsSection } from "./notifications-section";
import { Tabs, TabsList, TabsTrigger, TabsContent } from "@/components/ui/tabs";

const sections = [
  { value: "profile", label: "Profile" },
  { value: "security", label: "Security" },
  { value: "api-keys", label: "API keys" },
  { value: "storage", label: "Storage preferences" },
  { value: "notifications", label: "Notifications" },
];

export function SettingsTabs() {
  const [active, setActive] = useState("profile");

  return (
    <Tabs value={active} onValueChange={setActive}>
      <TabsList className="mb-6 flex-wrap">
        {sections.map((s) => (
          <TabsTrigger key={s.value} value={s.value}>
            {s.label}
          </TabsTrigger>
        ))}
      </TabsList>
      <TabsContent value="profile"><ProfileSection /></TabsContent>
      <TabsContent value="security"><SecuritySection /></TabsContent>
      <TabsContent value="api-keys"><ApiKeysSection /></TabsContent>
      <TabsContent value="storage"><StoragePreferencesSection /></TabsContent>
      <TabsContent value="notifications"><NotificationsSection /></TabsContent>
    </Tabs>
  );
}
