"use client";

export default function AdminSessionsPage() {
  return (
    <div style={{ animation: "fade-in 0.5s ease" }}>
      <div className="mb-6">
        <h1 className="text-2xl font-bold">Session Monitor</h1>
        <p className="text-muted text-sm">Monitor and manage all active user sessions</p>
      </div>

      <div className="glass-card p-8 text-center">
        <p className="text-4xl mb-4">🔑</p>
        <p className="text-lg font-medium mb-2">Session Monitoring</p>
        <p className="text-muted text-sm max-w-md mx-auto">
          Active sessions are tracked in the database. This view will show real-time session data
          once the admin sessions API endpoint is connected.
        </p>
        <div className="mt-6 p-4 rounded-xl bg-surface-light/50 max-w-sm mx-auto text-left">
          <p className="text-xs text-muted font-mono">GET /api/v1/admin/sessions</p>
          <p className="text-xs text-success mt-1">→ Returns all active sessions with user info</p>
        </div>
      </div>
    </div>
  );
}
