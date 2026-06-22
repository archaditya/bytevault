"use client";

import { useEffect, useState } from "react";
import { adminApi } from "@/lib/api";

const actionColors = {
  "user.register": "badge-success",
  "user.login": "badge-info",
  "user.logout": "badge-warning",
  "admin.view_users": "badge-info",
};

export default function ActivityLogsPage() {
  const [logs, setLogs] = useState([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [loading, setLoading] = useState(true);
  const limit = 50;

  useEffect(() => {
    setLoading(true);
    adminApi
      .listActivity(page, limit)
      .then((data) => {
        setLogs(data.logs || []);
        setTotal(data.total || 0);
      })
      .catch(() => {})
      .finally(() => setLoading(false));
  }, [page]);

  return (
    <div style={{ animation: "fade-in 0.5s ease" }}>
      <div className="mb-6">
        <h1 className="text-2xl font-bold">Activity Logs</h1>
        <p className="text-muted text-sm">{total} total events</p>
      </div>

      <div className="glass-card overflow-hidden">
        {loading ? (
          <div className="p-8 text-center text-muted">
            <div className="w-6 h-6 border-2 border-primary border-t-transparent rounded-full animate-spin mx-auto mb-3" />
            Loading activity...
          </div>
        ) : logs.length === 0 ? (
          <div className="p-8 text-center text-muted">No activity recorded yet</div>
        ) : (
          <table className="data-table">
            <thead>
              <tr>
                <th>Action</th>
                <th>Resource</th>
                <th>IP Address</th>
                <th>Time</th>
              </tr>
            </thead>
            <tbody>
              {logs.map((log) => (
                <tr key={log.id}>
                  <td>
                    <span className={`badge ${actionColors[log.action] || "badge-info"}`}>
                      {log.action}
                    </span>
                  </td>
                  <td className="text-muted text-sm">{log.resource_type || "—"}</td>
                  <td className="text-muted text-sm font-mono">{log.ip_address || "—"}</td>
                  <td className="text-muted text-sm">
                    {new Date(log.created_at).toLocaleString()}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}

        {Math.ceil(total / limit) > 1 && (
          <div className="flex items-center justify-between p-4 border-t border-border/50">
            <p className="text-sm text-muted">Page {page} of {Math.ceil(total / limit)}</p>
            <div className="flex gap-2">
              <button onClick={() => setPage((p) => Math.max(1, p - 1))} disabled={page === 1} className="px-3 py-1.5 rounded-lg bg-surface-light text-sm disabled:opacity-40 hover:bg-surface-lighter transition">← Prev</button>
              <button onClick={() => setPage((p) => p + 1)} disabled={page >= Math.ceil(total / limit)} className="px-3 py-1.5 rounded-lg bg-surface-light text-sm disabled:opacity-40 hover:bg-surface-lighter transition">Next →</button>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
