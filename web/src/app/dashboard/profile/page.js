"use client";

import { useAuth } from "@/contexts/AuthContext";

export default function ProfilePage() {
  const { user } = useAuth();

  const fields = [
    { label: "Email", value: user?.email, icon: "📧" },
    { label: "First Name", value: user?.first_name || "—", icon: "👤" },
    { label: "Last Name", value: user?.last_name || "—", icon: "👤" },
    { label: "Role", value: user?.role || "user", icon: "🛡️" },
    { label: "Verified", value: user?.is_verified ? "Yes" : "No", icon: "✅" },
    { label: "Joined", value: user?.created_at ? new Date(user.created_at).toLocaleDateString() : "—", icon: "📅" },
  ];

  return (
    <div style={{ animation: "fade-in 0.5s ease" }}>
      <h1 className="text-2xl font-bold mb-6">Profile</h1>

      <div className="glass-card p-8 max-w-2xl">
        {/* Avatar */}
        <div className="flex items-center gap-5 mb-8 pb-6 border-b border-border/50">
          <div className="w-16 h-16 rounded-2xl bg-gradient-to-br from-primary to-accent flex items-center justify-center text-2xl font-bold text-white">
            {user?.first_name?.[0] || user?.email?.[0] || "U"}
          </div>
          <div>
            <h2 className="text-lg font-semibold">
              {user?.first_name} {user?.last_name}
            </h2>
            <p className="text-muted text-sm">{user?.email}</p>
            <span className="badge badge-info mt-1">{user?.role || "user"}</span>
          </div>
        </div>

        {/* Fields */}
        <div className="space-y-4">
          {fields.map((f, i) => (
            <div key={i} className="flex items-center justify-between py-2">
              <div className="flex items-center gap-3">
                <span>{f.icon}</span>
                <span className="text-muted text-sm">{f.label}</span>
              </div>
              <span className="text-sm font-medium">{f.value}</span>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}
