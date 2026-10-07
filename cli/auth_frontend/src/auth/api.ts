export interface User { id: string; name: string; email: string; role: string }
export interface Session { appName: string; version: string; environment: string; showStudio: boolean; multi: boolean; csrf: string; user: User | null }
export interface Dashboard { user: User; stats?: { total: number; admins: number; members: number } }
export interface UserList { users: User[]; total: number; page: number; pageSize: number }
export class ApiError extends Error {
  constructor(public status: number, message: string, public fields: Record<string, string[]> = {}) { super(message); }
}

// Cookies remain HttpOnly. JavaScript holds only the CSRF token, in memory.
let csrf = "";
export async function api<T>(path: string, method = "GET", data?: Record<string, string>): Promise<T> {
  const response = await fetch(path, {
    method,
    credentials: "same-origin",
    headers: { "Accept": "application/json", ...(data ? { "Content-Type": "application/json" } : {}), ...(method !== "GET" ? { "X-CSRF-Token": csrf } : {}) },
    body: data ? JSON.stringify(data) : undefined,
  });
  const result = await response.json();
  if (!response.ok) throw new ApiError(response.status, result.error?.message || "Request could not be completed.", result.error?.fields || {});
  if (typeof result.csrf === "string") csrf = result.csrf;
  return result as T;
}
export const getSession = () => api<Session>("/api/session");
