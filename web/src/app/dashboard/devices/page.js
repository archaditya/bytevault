"use client";

import { useEffect, useState } from "react";
import { useAuth } from "@/contexts/AuthContext";
import { request } from "@/lib/api";

export default function DevicesPage() {
  const { user } = useAuth();
  const [devices, setDevices] = useState([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    request("/api/v1/me/devices")
      .then((data) => setDevices(data.devices || []))
      .catch(() => {})
      .finally(() => setLoading(false));
  }, []);

  return (
    <div style={{ animation: "fade-in 0.5s ease" }}>
      <h1 className="text-2xl font-bold mb-6">My Devices</h1>

      {loading ? (
        <div className="text-muted flex items-center gap-3">
          <div className="w-5 h-5 border-2 border-primary border-t-transparent rounded-full animate-spin" />
          Loading devices...
        </div>
      ) : devices.length === 0 ? (
        <div className="glass-card p-8 text-center">
          <p className="text-4xl mb-4">📱</p>
          <p className="text-muted">No devices registered yet.</p>
          <p className="text-muted text-sm mt-2">Devices will appear here when push notifications are enabled.</p>
        </div>
      ) : (
        <div className="space-y-4">
          {devices.map((d) => (
            <div key={d.id} className="glass-card p-5 flex items-center justify-between">
              <div className="flex items-center gap-4">
                <span className="text-2xl">
                  {d.device_type === "web" ? "🖥️" : d.device_type === "android" ? "📱" : "📱"}
                </span>
                <div>
                  <p className="font-medium text-sm">{d.device_type}</p>
                  <p className="text-muted text-xs font-mono">{d.device_id || "Unknown device"}</p>
                </div>
              </div>
              <span className={`badge ${d.is_active ? "badge-success" : "badge-danger"}`}>
                {d.is_active ? "Active" : "Inactive"}
              </span>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
