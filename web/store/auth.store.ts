import { create } from "zustand";
import { User } from "@/types";
import { apiClient, setTokens, getAccessToken } from "@/lib/api-client";

interface AuthState {
  user: User | null;
  isAuthenticated: boolean;
  isLoading: boolean;
  error: string | null;
  login: (email: string, password: string) => Promise<void>;
  register: (email: string, password: string, firstName: string, lastName: string) => Promise<void>;
  logout: () => Promise<void>;
  checkSession: () => Promise<void>;
  setUser: (user: User | null) => void;
}

function mapBackendUserToFrontend(backendUser: any): User {
  const firstName = backendUser.first_name || "";
  const lastName = backendUser.last_name || "";
  const fullName = [firstName, lastName].filter(Boolean).join(" ") || backendUser.email;
  
  const initials = [firstName[0], lastName[0]].filter(Boolean).join("").toUpperCase() || 
                   backendUser.email.slice(0, 2).toUpperCase();

  return {
    id: backendUser.id,
    name: fullName,
    email: backendUser.email,
    avatar: initials,
    role: backendUser.role || "user",
    plan: "free",
    joinedAt: backendUser.created_at || new Date().toISOString(),
    apiKeysCount: 0,
    twoFactorEnabled: false,
  };
}

export const useAuthStore = create<AuthState>((set) => ({
  user: null,
  isAuthenticated: false,
  isLoading: true,
  error: null,

  setUser: (user: User) => set({ user, isAuthenticated: !!user }),

  login: async (email: string, password: string) => {
    set({ isLoading: true, error: null });
    try {
      const data = await apiClient("/api/v1/auth/login", {
        method: "POST",
        body: JSON.stringify({ email, password }),
      });
      setTokens(data.tokens);
      const frontendUser = mapBackendUserToFrontend(data.user);
      set({ user: frontendUser, isAuthenticated: true, isLoading: false });
    } catch (err: any) {
      set({ error: err.message || "Login failed", isLoading: false });
      throw err;
    }
  },

  register: async (email, password, firstName, lastName) => {
    set({ isLoading: true, error: null });
    try {
      const data = await apiClient("/api/v1/auth/register", {
        method: "POST",
        body: JSON.stringify({ email, password, first_name: firstName, last_name: lastName }),
      });
      setTokens(data.tokens);
      const frontendUser = mapBackendUserToFrontend(data.user);
      set({ user: frontendUser, isAuthenticated: true, isLoading: false });
    } catch (err: any) {
      set({ error: err.message || "Registration failed", isLoading: false });
      throw err;
    }
  },

  logout: async () => {
    set({ isLoading: true });
    try {
      const refreshToken = localStorage.getItem("refreshToken");
      if (refreshToken) {
        await apiClient("/api/v1/auth/logout", {
          method: "POST",
          body: JSON.stringify({ refresh_token: refreshToken }),
        });
      }
    } catch (err) {
      console.error("Logout request failed:", err);
    } finally {
      setTokens(null);
      set({ user: null, isAuthenticated: false, isLoading: false });
      if (typeof window !== "undefined") {
        window.location.href = "/login";
      }
    }
  },

  checkSession: async () => {
    const token = getAccessToken();
    if (!token) {
      set({ user: null, isAuthenticated: false, isLoading: false });
      return;
    }
    try {
      const data = await apiClient("/api/v1/me");
      const frontendUser = mapBackendUserToFrontend(data.user);
      set({ user: frontendUser, isAuthenticated: true, isLoading: false });
    } catch (err) {
      console.error("Session verification failed:", err);
      setTokens(null);
      set({ user: null, isAuthenticated: false, isLoading: false });
    }
  },
}));
