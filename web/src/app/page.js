"use client";

import Link from "next/link";
import { useAuth } from "@/contexts/AuthContext";

const features = [
  {
    icon: "🔐",
    title: "Military-Grade Encryption",
    desc: "AES-256 encryption at rest and in transit. Your files are untouchable.",
  },
  {
    icon: "⚡",
    title: "Lightning Fast",
    desc: "CDN-backed delivery with intelligent caching. Global access in milliseconds.",
  },
  {
    icon: "🛡️",
    title: "Role-Based Access",
    desc: "Granular permissions system. Control exactly who sees what.",
  },
  {
    icon: "📊",
    title: "Real-Time Analytics",
    desc: "Monitor uploads, downloads, and user activity in real-time.",
  },
  {
    icon: "🔄",
    title: "Version Control",
    desc: "Track every change. Roll back to any previous version instantly.",
  },
  {
    icon: "🌐",
    title: "API-First Design",
    desc: "RESTful APIs with JWT auth. Integrate with any system.",
  },
];

export default function LandingPage() {
  const { isAuthenticated, user } = useAuth();

  return (
    <div className="min-h-screen circuit-bg relative overflow-hidden">
      {/* Floating particles */}
      <div className="fixed inset-0 pointer-events-none overflow-hidden">
        {[...Array(20)].map((_, i) => (
          <div
            key={i}
            className="absolute rounded-full bg-primary/20"
            style={{
              width: Math.random() * 4 + 2 + "px",
              height: Math.random() * 4 + 2 + "px",
              left: Math.random() * 100 + "%",
              top: Math.random() * 100 + "%",
              animation: `float ${Math.random() * 6 + 4}s ease-in-out infinite`,
              animationDelay: Math.random() * 5 + "s",
            }}
          />
        ))}
      </div>

      {/* Nav */}
      <nav className="relative z-10 flex items-center justify-between px-6 md:px-12 py-5 border-b border-border/50">
        <div className="flex items-center gap-3">
          <div className="w-9 h-9 rounded-lg bg-gradient-to-br from-primary to-accent flex items-center justify-center text-white font-bold text-sm">
            BV
          </div>
          <span className="text-lg font-bold tracking-tight">
            Byte<span className="text-primary">Vault</span>
          </span>
        </div>
        <div className="flex items-center gap-4">
          {isAuthenticated ? (
            <Link
              href={user?.role === "super_admin" ? "/admin" : "/dashboard"}
              className="btn-primary text-sm px-5 py-2.5"
            >
              Dashboard →
            </Link>
          ) : (
            <>
              <Link
                href="/login"
                className="text-muted hover:text-foreground transition text-sm"
              >
                Sign In
              </Link>
              <Link
                href="/register"
                className="btn-primary text-sm px-5 py-2.5"
              >
                Get Started
              </Link>
            </>
          )}
        </div>
      </nav>

      {/* Hero */}
      <section className="relative z-10 flex flex-col items-center text-center px-6 pt-20 pb-16 md:pt-32 md:pb-24">
        <div
          className="inline-flex items-center gap-2 px-4 py-1.5 rounded-full border border-primary/20 bg-primary/5 mb-8"
          style={{ animation: "fade-in 0.8s ease" }}
        >
          <span className="w-2 h-2 rounded-full bg-success animate-pulse" />
          <span className="text-xs text-primary font-medium">
            System Online — All Services Operational
          </span>
        </div>

        <h1
          className="text-4xl md:text-6xl lg:text-7xl font-bold leading-tight mb-6 max-w-4xl"
          style={{ animation: "slide-up 0.8s ease" }}
        >
          Secure File Storage
          <br />
          <span className="bg-gradient-to-r from-primary to-accent bg-clip-text text-transparent">
            Built for Scale
          </span>
        </h1>

        <p
          className="text-muted text-lg md:text-xl max-w-2xl mb-10"
          style={{ animation: "slide-up 1s ease" }}
        >
          Enterprise-grade file storage with role-based access control,
          real-time monitoring, and military-grade encryption.
        </p>

        <div
          className="flex flex-col sm:flex-row gap-4"
          style={{ animation: "slide-up 1.2s ease" }}
        >
          <Link href="/register" className="btn-primary text-base px-8 py-3.5">
            Start Building Free →
          </Link>
          <a
            href="https://github.com/archaditya/bytevault"
            target="_blank"
            className="px-8 py-3.5 rounded-xl border border-border-light text-foreground hover:bg-surface-light transition text-base font-medium text-center"
          >
            View on GitHub
          </a>
        </div>

        {/* Terminal preview */}
        <div
          className="mt-16 w-full max-w-2xl glass-card p-0 overflow-hidden text-left"
          style={{ animation: "slide-up 1.4s ease" }}
        >
          <div className="flex items-center gap-2 px-4 py-3 border-b border-border/50">
            <div className="w-3 h-3 rounded-full bg-danger/60" />
            <div className="w-3 h-3 rounded-full bg-warning/60" />
            <div className="w-3 h-3 rounded-full bg-success/60" />
            <span className="text-xs text-muted ml-2 font-mono">
              terminal — bytevault
            </span>
          </div>
          <div className="p-5 font-mono text-sm leading-relaxed">
            <p className="text-muted">$ curl -X POST /api/v1/auth/register</p>
            <p className="text-success mt-1">✓ User created successfully</p>
            <p className="text-muted mt-3">
              $ curl -H &quot;Authorization: Bearer eyJhbG...&quot; /api/v1/me
            </p>
            <p className="text-primary mt-1">
              {`{ "user": { "email": "dev@bytevault.io", "role": "user" } }`}
            </p>
            <p className="text-muted mt-3">
              <span className="text-accent">▌</span>
            </p>
          </div>
        </div>
      </section>

      {/* Features */}
      <section className="relative z-10 px-6 md:px-12 pb-24">
        <div className="max-w-6xl mx-auto">
          <h2 className="text-3xl font-bold text-center mb-12">
            Built for <span className="text-primary">Production</span>
          </h2>
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
            {features.map((f, i) => (
              <div
                key={i}
                className="glass-card p-6 cursor-default"
                style={{ animation: `slide-up ${0.8 + i * 0.1}s ease` }}
              >
                <div className="text-3xl mb-4">{f.icon}</div>
                <h3 className="text-lg font-semibold mb-2">{f.title}</h3>
                <p className="text-muted text-sm leading-relaxed">{f.desc}</p>
              </div>
            ))}
          </div>
        </div>
      </section>

      {/* Footer */}
      <footer className="relative z-10 border-t border-border/50 py-8 text-center text-muted text-sm">
        <p>© 2026 ByteVault. Built with Go + Next.js.</p>
      </footer>
    </div>
  );
}
