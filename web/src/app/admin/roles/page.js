"use client";

import { useEffect, useState } from "react";
import { adminApi } from "@/lib/api";

export default function RolesPage() {
  const [roles, setRoles] = useState([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    adminApi
      .listRoles()
      .then((data) => setRoles(data.roles || []))
      .catch(() => {})
      .finally(() => setLoading(false));
  }, []);

  return (
    <div style={{ animation: "fade-in 0.5s ease" }}>
      <div className="mb-6">
        <h1 className="text-2xl font-bold">Role Management</h1>
        <p className="text-muted text-sm">Manage system roles and permissions</p>
      </div>

      {loading ? (
        <div className="text-muted flex items-center gap-3">
          <div className="w-5 h-5 border-2 border-primary border-t-transparent rounded-full animate-spin" />
          Loading roles...
        </div>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
          {roles.map((role) => (
            <div key={role.id} className="glass-card p-6">
              <div className="flex items-center justify-between mb-4">
                <div>
                  <h3 className="text-lg font-semibold capitalize">{role.name}</h3>
                  <p className="text-muted text-sm">{role.description}</p>
                </div>
                {role.is_system_role && <span className="badge badge-warning">System</span>}
              </div>

              <div className="space-y-2">
                <p className="text-xs text-muted uppercase tracking-wider font-semibold mb-2">Permissions</p>
                <div className="flex flex-wrap gap-2">
                  {role.permissions &&
                    Object.entries(role.permissions).map(([perm, allowed]) => (
                      <span
                        key={perm}
                        className={`badge text-xs ${allowed ? "badge-success" : "badge-danger"}`}
                      >
                        {perm}: {allowed ? "✓" : "✗"}
                      </span>
                    ))}
                </div>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
