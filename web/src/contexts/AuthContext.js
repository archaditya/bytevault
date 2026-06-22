"use client";

import { createContext, useContext, useState, useEffect } from "react";
import { authApi, tokenStore } from "@/lib/api";

const AuthContext = createContext(null);

export function AuthProvider({ children }) {
  const [user, setUser] = useState(null);
  const [isLoading, setIsLoading] = useState(true);

  useEffect(() => {
    const token = tokenStore.getAccess();
    if (token) {
      authApi
        .me()
        .then((data) => setUser(data?.user ?? data))
        .catch(() => tokenStore.clear())
        .finally(() => setIsLoading(false));
    } else {
      setIsLoading(false);
    }
  }, []);

  const applySession = (data) => {
    tokenStore.set(data.tokens.access_token, data.tokens.refresh_token);
    setUser(data.user);
  };

  const login = async (email, password) => {
    const data = await authApi.login(email, password);
    applySession(data);
    return data;
  };

  const register = async (email, password, firstName, lastName) => {
    const data = await authApi.register(email, password, firstName, lastName);
    applySession(data);
    return data;
  };

  const logout = async () => {
    const refreshToken = tokenStore.getRefresh();
    tokenStore.clear();
    setUser(null);
    try {
      if (refreshToken) await authApi.logout(refreshToken);
    } catch { /* ignore */ }
  };

  return (
    <AuthContext.Provider
      value={{
        user,
        isAuthenticated: !!user,
        isLoading,
        login,
        register,
        logout,
        setUser,
      }}
    >
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth() {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error("useAuth must be used within AuthProvider");
  return ctx;
}
