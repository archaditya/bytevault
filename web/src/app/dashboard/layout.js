"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { useAuth } from "@/contexts/AuthContext";
import { AuthGuard } from "@/components/auth/Guards";

const userLinks = [
  { href: "/dashboard", label: "Dashboard", icon: "📊" },
  { href: "/dashboard/profile", label: "Profile", icon: "👤" },
  { href: "/dashboard/devices", label: "Devices", icon: "📱" },
  { href: "/dashboard/sessions", label: "Sessions", icon: "🔑" },
];

export default function DashboardLayout({ children }) {
  const pathname = usePathname();
  const { user, logout } = useAuth();

  return (
    <AuthGuard>
      <div className="min-h-screen flex bg-background">
        {/* Sidebar */}
        <aside className="w-64 border-r border-border/50 bg-surface/50 flex flex-col shrink-0 sticky top-0 h-screen">
          <div className="p-5 border-b border-border/50">
            <Link href="/" className="flex items-center gap-3">
              <div className="w-9 h-9 rounded-lg bg-gradient-to-br from-primary to-accent flex items-center justify-center text-white font-bold text-sm">BV</div>
              <span className="font-bold text-sm">Byte<span className="text-primary">Vault</span></span>
            </Link>
          </div>

          <nav className="flex-1 p-4 space-y-1">
            {userLinks.map((link) => (
              <Link
                key={link.href}
                href={link.href}
                className={`sidebar-link ${pathname === link.href ? "active" : ""}`}
              >
                <span>{link.icon}</span>
                <span>{link.label}</span>
              </Link>
            ))}

            {user?.role === "super_admin" && (
              <>
                <div className="pt-4 pb-2 px-4">
                  <p className="text-xs text-muted uppercase tracking-wider font-semibold">Admin</p>
                </div>
                <Link href="/admin" className={`sidebar-link ${pathname.startsWith("/admin") ? "active" : ""}`}>
                  <span>🛡️</span><span>Admin Panel</span>
                </Link>
              </>
            )}
          </nav>

          <div className="p-4 border-t border-border/50">
            <div className="flex items-center gap-3 mb-3">
              <div className="w-9 h-9 rounded-full bg-gradient-to-br from-primary/20 to-accent/20 flex items-center justify-center text-sm font-semibold text-primary">
                {user?.first_name?.[0] || user?.email?.[0] || "U"}
              </div>
              <div className="min-w-0">
                <p className="text-sm font-medium truncate">{user?.first_name || "User"}</p>
                <p className="text-xs text-muted truncate">{user?.email}</p>
              </div>
            </div>
            <button
              onClick={logout}
              className="w-full text-left sidebar-link text-danger hover:bg-danger/5 hover:text-danger"
            >
              <span>🚪</span><span>Logout</span>
            </button>
          </div>
        </aside>

        {/* Main content */}
        <main className="flex-1 min-w-0 overflow-auto">
          <div className="p-6 md:p-8 max-w-6xl">{children}</div>
        </main>
      </div>
    </AuthGuard>
  );
}
