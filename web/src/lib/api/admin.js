import { request } from "./http";

export const adminApi = {
  getStats: () => request("/api/v1/admin/stats"),

  listUsers: (page = 1, limit = 20) =>
    request(`/api/v1/admin/users?page=${page}&limit=${limit}`),

  listRoles: () => request("/api/v1/admin/roles"),

  listActivity: (page = 1, limit = 50) =>
    request(`/api/v1/admin/activity?page=${page}&limit=${limit}`),
};
