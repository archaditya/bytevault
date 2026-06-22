"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { useAuth } from "@/contexts/AuthContext";
import { AuthGuard, RoleGuard } from "@/components/auth/Guards";

const adminLinks = [
  { href: "/admin", label: "Overview", icon: "📊" },
  { href: "/admin/users", label: "Users", icon: "👥" },
  { href: "/admin/roles", label: "Roles", icon: "🛡️" },
  { href: "/admin/activity", label: "Activity Logs", icon: "📋" },
  { href: "/admin/sessions", label: "Sessions", icon: "🔑" },
];

export default function AdminLayout({ children }) {
  const pathname = usePathname();
  const { user, logout } = useAuth();

  return (
    <AuthGuard>
      <RoleGuard requiredRole="super_admin">
        <div className="min-h-screen flex bg-background">
          {/* Sidebar */}
          <aside className="w-64 border-r border-danger/10 bg-surface/50 flex flex-col shrink-0 sticky top-0 h-screen">
            <div className="p-5 border-b border-border/50">
              <Link href="/" className="flex items-center gap-3">
                <div className="w-9 h-9 rounded-lg bg-gradient-to-br from-danger to-accent flex items-center justify-center text-white font-bold text-sm">BV</div>
                <div>
                  <span className="font-bold text-sm block">ByteVault</span>
                  <span className="text-xs text-danger font-medium">ADMIN</span>
                </div>
              </Link>
            </div>

            <nav className="flex-1 p-4 space-y-1">
              {adminLinks.map((link) => (
                <Link
                  key={link.href}
                  href={link.href}
                  className={`sidebar-link ${pathname === link.href ? "active" : ""}`}
                >
                  <span>{link.icon}</span>
                  <span>{link.label}</span>
                </Link>
              ))}

              <div className="pt-4 pb-2 px-4">
                <p className="text-xs text-muted uppercase tracking-wider font-semibold">User View</p>
              </div>
              <Link href="/dashboard" className="sidebar-link">
                <span>🏠</span><span>User Dashboard</span>
              </Link>
            </nav>

            <div className="p-4 border-t border-border/50">
              <div className="flex items-center gap-3 mb-3">
                <div className="w-9 h-9 rounded-full bg-gradient-to-br from-danger/20 to-accent/20 flex items-center justify-center text-sm font-semibold text-danger">
                  {user?.first_name?.[0] || "A"}
                </div>
                <div className="min-w-0">
                  <p className="text-sm font-medium truncate">{user?.first_name || "Admin"}</p>
                  <p className="text-xs text-danger truncate">Super Admin</p>
                </div>
              </div>
              <button onClick={logout} className="w-full text-left sidebar-link text-danger hover:bg-danger/5">
                <span>🚪</span><span>Logout</span>
              </button>
            </div>
          </aside>

          <main className="flex-1 min-w-0 overflow-auto">
            <div className="p-6 md:p-8 max-w-7xl">{children}</div>
          </main>
        </div>
      </RoleGuard>
    </AuthGuard>
  );
}
