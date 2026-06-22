"use client";

import { useEffect, useState } from "react";
import { adminApi } from "@/lib/api";

export default function AdminDashboard() {
  const [stats, setStats] = useState(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    adminApi
      .getStats()
      .then(setStats)
      .catch(() => {})
      .finally(() => setLoading(false));
  }, []);

  const cards = stats
    ? [
        { label: "Total Users", value: stats.total_users, icon: "👥", color: "text-primary" },
        { label: "Active Users", value: stats.active_users, icon: "✅", color: "text-success" },
        { label: "Verified Users", value: stats.verified_users, icon: "🛡️", color: "text-accent-light" },
        { label: "Active Sessions", value: stats.active_sessions, icon: "🔑", color: "text-warning" },
      ]
    : [];

  return (
    <div style={{ animation: "fade-in 0.5s ease" }}>
      <div className="mb-8">
        <h1 className="text-2xl font-bold mb-1">Admin Dashboard</h1>
        <p className="text-muted text-sm">System overview and monitoring</p>
      </div>

      {loading ? (
        <div className="flex items-center gap-3 text-muted">
          <div className="w-5 h-5 border-2 border-primary border-t-transparent rounded-full animate-spin" />
          Loading stats...
        </div>
      ) : (
        <>
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4 mb-8">
            {cards.map((card, i) => (
              <div key={i} className="stat-card" style={{ animation: `slide-up ${0.3 + i * 0.1}s ease` }}>
                <div className="flex items-center justify-between mb-3">
                  <span className="text-2xl">{card.icon}</span>
                  <span className={`text-3xl font-bold ${card.color}`}>{card.value}</span>
                </div>
                <p className="text-muted text-sm">{card.label}</p>
              </div>
            ))}
          </div>

          <div className="glass-card p-6">
            <h3 className="text-lg font-semibold mb-4">System Health</h3>
            <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
              {[
                { name: "Go API", version: "v1.0", uptime: "99.9%", ok: true },
                { name: "PostgreSQL", version: "18", uptime: "99.9%", ok: true },
                { name: "Firebase FCM", version: "Admin SDK", uptime: "—", ok: false },
              ].map((s, i) => (
                <div key={i} className="p-4 rounded-xl bg-surface-light/50 border border-border/50">
                  <div className="flex items-center gap-2 mb-2">
                    <span className={`w-2 h-2 rounded-full ${s.ok ? "bg-success animate-pulse" : "bg-muted"}`} />
                    <span className="font-medium text-sm">{s.name}</span>
                  </div>
                  <p className="text-xs text-muted">Version: {s.version}</p>
                  <p className="text-xs text-muted">Uptime: {s.uptime}</p>
                </div>
              ))}
            </div>
          </div>
        </>
      )}
    </div>
  );
}
