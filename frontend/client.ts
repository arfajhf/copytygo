export type ApiError = { error?: { status?: number; message?: string }; message?: string };

export class CopyTyGoClient {
  constructor(private readonly baseURL = "") {}

  async request<T>(path: string, init: RequestInit = {}): Promise<T> {
    const headers = new Headers(init.headers);
    if (init.body && !headers.has("Content-Type")) headers.set("Content-Type", "application/json");
    const csrf = this.cookie("ctg_csrf");
    if (csrf) headers.set("X-CSRF-Token", csrf);
    const response = await fetch(this.baseURL + path, { ...init, headers });
    const text = await response.text();
    const body = text ? JSON.parse(text) : null;
    if (!response.ok) {
      const error = body as ApiError;
      throw new Error(error?.error?.message ?? error?.message ?? `HTTP ${response.status}`);
    }
    return body as T;
  }

  get<T>(path: string) { return this.request<T>(path); }
  post<T>(path: string, data: unknown) { return this.request<T>(path, { method: "POST", body: JSON.stringify(data) }); }
  put<T>(path: string, data: unknown) { return this.request<T>(path, { method: "PUT", body: JSON.stringify(data) }); }
  patch<T>(path: string, data: unknown) { return this.request<T>(path, { method: "PATCH", body: JSON.stringify(data) }); }
  delete<T>(path: string) { return this.request<T>(path, { method: "DELETE" }); }

  private cookie(name: string): string {
    const row = document.cookie.split("; ").find(value => value.startsWith(`${name}=`));
    return row ? decodeURIComponent(row.split("=").slice(1).join("=")) : "";
  }
}

export const api = new CopyTyGoClient();
