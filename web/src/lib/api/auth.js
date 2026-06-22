import { request } from "./http";

export const authApi = {
  register: (email, password, first_name, last_name) =>
    request("/api/v1/auth/register", {
      method: "POST",
      body: JSON.stringify({ email, password, first_name, last_name }),
    }),

  login: (email, password) =>
    request("/api/v1/auth/login", {
      method: "POST",
      body: JSON.stringify({ email, password }),
    }),

  refresh: (refresh_token) =>
    request("/api/v1/auth/refresh", {
      method: "POST",
      body: JSON.stringify({ refresh_token }),
    }),

  logout: (refresh_token) =>
    request("/api/v1/auth/logout", {
      method: "POST",
      body: JSON.stringify({ refresh_token }),
    }),

  me: () => request("/api/v1/me"),
};
