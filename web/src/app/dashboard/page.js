"use client";

import { useAuth } from "@/contexts/AuthContext";

export default function DashboardPage() {
  const { user } = useAuth();

  const stats = [
    { label: "Files Uploaded", value: "0", icon: "📁", color: "text-primary" },
    { label: "Storage Used", value: "0 MB", icon: "💾", color: "text-accent-light" },
    { label: "Shared Files", value: "0", icon: "🔗", color: "text-success" },
    { label: "Active Sessions", value: "1", icon: "🔑", color: "text-warning" },
  ];

  return (
    <div style={{ animation: "fade-in 0.5s ease" }}>
      {/* Header */}
      <div className="mb-8">
        <h1 className="text-2xl font-bold mb-1">
          Welcome back, <span className="text-primary">{user?.first_name || "User"}</span>
        </h1>
        <p className="text-muted text-sm">
          Your vault is secure. Role:{" "}
          <span className="badge badge-info">{user?.role || "user"}</span>
        </p>
      </div>

      {/* Stats grid */}
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4 mb-8">
        {stats.map((stat, i) => (
          <div key={i} className="stat-card" style={{ animation: `slide-up ${0.3 + i * 0.1}s ease` }}>
            <div className="flex items-center justify-between mb-3">
              <span className="text-2xl">{stat.icon}</span>
              <span className={`text-2xl font-bold ${stat.color}`}>{stat.value}</span>
            </div>
            <p className="text-muted text-sm">{stat.label}</p>
          </div>
        ))}
      </div>

      {/* Quick actions + Recent activity */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Quick actions */}
        <div className="glass-card p-6">
          <h3 className="text-lg font-semibold mb-4">Quick Actions</h3>
          <div className="space-y-3">
            <button className="w-full flex items-center gap-3 p-3 rounded-xl bg-surface-light/50 hover:bg-surface-light transition text-left text-sm">
              <span className="text-xl">📤</span>
              <div>
                <p className="font-medium">Upload Files</p>
                <p className="text-muted text-xs">Drag & drop or browse files</p>
              </div>
            </button>
            <button className="w-full flex items-center gap-3 p-3 rounded-xl bg-surface-light/50 hover:bg-surface-light transition text-left text-sm">
              <span className="text-xl">📂</span>
              <div>
                <p className="font-medium">Create Folder</p>
                <p className="text-muted text-xs">Organize your vault</p>
              </div>
            </button>
            <button className="w-full flex items-center gap-3 p-3 rounded-xl bg-surface-light/50 hover:bg-surface-light transition text-left text-sm">
              <span className="text-xl">🔗</span>
              <div>
                <p className="font-medium">Share a File</p>
                <p className="text-muted text-xs">Generate secure share links</p>
              </div>
            </button>
          </div>
        </div>

        {/* System status */}
        <div className="glass-card p-6">
          <h3 className="text-lg font-semibold mb-4">System Status</h3>
          <div className="space-y-4">
            {[
              { name: "API Server", status: "Operational", ok: true },
              { name: "Database", status: "Connected", ok: true },
              { name: "Storage CDN", status: "Coming Soon", ok: false },
              { name: "Push Notifications", status: "Coming Soon", ok: false },
            ].map((s, i) => (
              <div key={i} className="flex items-center justify-between">
                <div className="flex items-center gap-3">
                  <span className={`w-2.5 h-2.5 rounded-full ${s.ok ? "bg-success animate-pulse" : "bg-muted"}`} />
                  <span className="text-sm">{s.name}</span>
                </div>
                <span className={`text-xs ${s.ok ? "text-success" : "text-muted"}`}>{s.status}</span>
              </div>
            ))}
          </div>
        </div>
      </div>
    </div>
  );
}
