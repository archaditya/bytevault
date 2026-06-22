"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import Link from "next/link";
import { useAuth } from "@/contexts/AuthContext";

export default function RegisterPage() {
  const [form, setForm] = useState({ email: "", password: "", firstName: "", lastName: "" });
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);
  const { register } = useAuth();
  const router = useRouter();

  const update = (key) => (e) => setForm({ ...form, [key]: e.target.value });

  const handleSubmit = async (e) => {
    e.preventDefault();
    setError("");
    if (form.password.length < 8) { setError("Password must be at least 8 characters"); return; }
    setLoading(true);
    try {
      await register(form.email, form.password, form.firstName, form.lastName);
      router.push("/dashboard");
    } catch (err) {
      setError(err.message || "Registration failed");
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="min-h-screen circuit-bg flex items-center justify-center px-4 relative overflow-hidden">
      <div className="absolute top-1/3 right-1/4 w-96 h-96 bg-accent/5 rounded-full blur-3xl" />
      <div className="absolute bottom-1/3 left-1/4 w-96 h-96 bg-primary/5 rounded-full blur-3xl" />

      <div className="w-full max-w-md relative z-10" style={{ animation: "slide-up 0.6s ease" }}>
        <div className="text-center mb-8">
          <Link href="/" className="inline-flex items-center gap-3">
            <div className="w-11 h-11 rounded-xl bg-gradient-to-br from-primary to-accent flex items-center justify-center text-white font-bold">BV</div>
            <span className="text-2xl font-bold">Byte<span className="text-primary">Vault</span></span>
          </Link>
          <p className="text-muted text-sm mt-3">Create your secure vault</p>
        </div>

        <div className="glass-card p-8">
          <h2 className="text-xl font-semibold mb-6">Create Account</h2>

          {error && (
            <div className="mb-4 p-3 rounded-lg bg-danger/10 border border-danger/20 text-danger text-sm">{error}</div>
          )}

          <form onSubmit={handleSubmit} className="space-y-4">
            <div className="grid grid-cols-2 gap-3">
              <div>
                <label className="block text-sm text-muted mb-1.5">First Name</label>
                <input type="text" value={form.firstName} onChange={update("firstName")} className="w-full glow-input rounded-xl px-4 py-3 text-sm text-foreground" placeholder="Aditya" />
              </div>
              <div>
                <label className="block text-sm text-muted mb-1.5">Last Name</label>
                <input type="text" value={form.lastName} onChange={update("lastName")} className="w-full glow-input rounded-xl px-4 py-3 text-sm text-foreground" placeholder="Kumar" />
              </div>
            </div>

            <div>
              <label className="block text-sm text-muted mb-1.5">Email</label>
              <input type="email" value={form.email} onChange={update("email")} className="w-full glow-input rounded-xl px-4 py-3 text-sm text-foreground" placeholder="you@example.com" required />
            </div>

            <div>
              <label className="block text-sm text-muted mb-1.5">Password</label>
              <input type="password" value={form.password} onChange={update("password")} className="w-full glow-input rounded-xl px-4 py-3 text-sm text-foreground" placeholder="Min. 8 characters" required />
            </div>

            <button type="submit" disabled={loading} className="w-full btn-primary py-3 text-sm mt-2">
              {loading ? (
                <span className="flex items-center justify-center gap-2">
                  <span className="w-4 h-4 border-2 border-white/30 border-t-white rounded-full animate-spin" />
                  Creating vault...
                </span>
              ) : (
                "Create Account →"
              )}
            </button>
          </form>

          <p className="text-center text-muted text-sm mt-6">
            Already have an account?{" "}
            <Link href="/login" className="text-primary hover:underline">Sign in</Link>
          </p>
        </div>
      </div>
    </div>
  );
}
