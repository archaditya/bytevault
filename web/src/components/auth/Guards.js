"use client";

import { useAuth } from "@/contexts/AuthContext";
import { useRouter } from "next/navigation";
import { useEffect } from "react";

export function AuthGuard({ children }) {
  const { isAuthenticated, isLoading } = useAuth();
  const router = useRouter();

  useEffect(() => {
    if (!isLoading && !isAuthenticated) {
      router.replace("/login");
    }
  }, [isLoading, isAuthenticated, router]);

  if (isLoading) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-background">
        <div className="flex flex-col items-center gap-4">
          <div className="w-10 h-10 border-2 border-primary border-t-transparent rounded-full animate-spin" />
          <p className="text-muted text-sm">Verifying access...</p>
        </div>
      </div>
    );
  }

  if (!isAuthenticated) return null;
  return children;
}

export function RoleGuard({ children, requiredRole }) {
  const { user, isLoading } = useAuth();
  const router = useRouter();

  useEffect(() => {
    if (!isLoading && user && user.role !== requiredRole) {
      router.replace("/dashboard");
    }
  }, [isLoading, user, requiredRole, router]);

  if (isLoading) return null;
  if (!user || user.role !== requiredRole) return null;
  return children;
}
