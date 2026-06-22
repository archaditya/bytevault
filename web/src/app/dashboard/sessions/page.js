"use client";

import { useState } from "react";
import { useAuth } from "@/contexts/AuthContext";

export default function SessionsPage() {
  const { user } = useAuth();

  return (
    <div style={{ animation: "fade-in 0.5s ease" }}>
      <h1 className="text-2xl font-bold mb-6">Active Sessions</h1>

      <div className="glass-card p-6">
        <div className="space-y-4">
          {/* Current session */}
          <div className="p-4 rounded-xl bg-surface-light/50 border border-primary/20">
            <div className="flex items-center justify-between mb-2">
              <div className="flex items-center gap-3">
                <span className="text-xl">🖥️</span>
                <div>
                  <p className="font-medium text-sm">Current Session</p>
                  <p className="text-muted text-xs">This device</p>
                </div>
              </div>
              <span className="badge badge-success">Active</span>
            </div>
            <div className="flex gap-4 mt-3 text-xs text-muted">
              <span>🕐 Started: {new Date().toLocaleDateString()}</span>
              <span>📍 Local</span>
            </div>
          </div>

          <p className="text-muted text-sm text-center py-4">
            Session management via API is available. Full session list coming with admin endpoints.
          </p>
        </div>
      </div>
    </div>
  );
}
