const BASE_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

const TOKEN_KEY = "bytevault_access_token";
const REFRESH_KEY = "bytevault_refresh_token";

export const tokenStore = {
  getAccess: () => (typeof window !== "undefined" ? localStorage.getItem(TOKEN_KEY) : null),
  getRefresh: () => (typeof window !== "undefined" ? localStorage.getItem(REFRESH_KEY) : null),
  set: (access, refresh) => {
    localStorage.setItem(TOKEN_KEY, access);
    if (refresh) localStorage.setItem(REFRESH_KEY, refresh);
  },
  clear: () => {
    localStorage.removeItem(TOKEN_KEY);
    localStorage.removeItem(REFRESH_KEY);
  },
};

function extractError(payload, fallback) {
  if (typeof payload === "string") return payload;
  if (!payload || typeof payload !== "object") return fallback;
  return payload.message || payload.error || payload.detail || fallback;
}

async function refreshAccessToken() {
  const refresh = tokenStore.getRefresh();
  if (!refresh) return null;
  try {
    const res = await fetch(`${BASE_URL}/api/v1/auth/refresh`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ refresh_token: refresh }),
    });
    if (!res.ok) { tokenStore.clear(); return null; }
    const data = await res.json();
    tokenStore.set(data.tokens.access_token, data.tokens.refresh_token);
    return data.tokens.access_token;
  } catch {
    tokenStore.clear();
    return null;
  }
}

export async function request(path, options = {}, retry = true) {
  const token = tokenStore.getAccess();
  const headers = { ...(options.headers || {}) };

  if (!(options.body instanceof FormData) && !headers["Content-Type"]) {
    headers["Content-Type"] = "application/json";
  }
  if (token && !headers["Authorization"]) {
    headers["Authorization"] = `Bearer ${token}`;
  }

  const res = await fetch(`${BASE_URL}${path}`, { ...options, headers });

  if (res.status === 401 && retry) {
    const newToken = await refreshAccessToken();
    if (newToken) return request(path, options, false);
    tokenStore.clear();
    if (typeof window !== "undefined") window.location.href = "/login";
    throw new Error("Session expired");
  }

  if (!res.ok) {
    const err = await res.json().catch(() => ({ error: res.statusText }));
    throw new Error(extractError(err, "Request failed"));
  }

  if (res.status === 204) return undefined;
  return res.json();
}
