package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/arfajhf/copytygo/v4/version"
)

func NewProject(name string) error {
	if name == "" || strings.ContainsAny(name, "/\\") {
		return fmt.Errorf("copytygo: invalid project name")
	}
	if _, err := os.Stat(name); err == nil {
		return fmt.Errorf("copytygo: directory %q already exists", name)
	}
	dirs := []string{"cmd/app", "app/controllers", "app/models", "app/middleware", "app/services", "database/migrations", "routes", "frontend/src"}
	for _, d := range dirs {
		if err := os.MkdirAll(filepath.Join(name, d), 0755); err != nil {
			return err
		}
	}
	key, _ := GenerateKey()
	files := map[string]string{
		"go.mod":                 "module " + name + "\n\ngo 1.27.1\n\nrequire github.com/arfajhf/copytygo/v4 " + version.StableModule + "\n",
		".env":                   "APP_NAME=" + name + "\nAPP_ENV=local\nAPP_DEBUG=true\nAPP_HOST=127.0.0.1\nAPP_PORT=8080\nAPP_KEY=" + key + "\nCOPYTYGO_STUDIO=true\nCOPYTYGO_DOCS_URL=" + version.DocsURL + "\nSERVER_READ_HEADER_TIMEOUT=5\nSERVER_READ_TIMEOUT=15\nSERVER_WRITE_TIMEOUT=30\nSERVER_IDLE_TIMEOUT=60\nSERVER_SHUTDOWN_TIMEOUT=10\nDB_DRIVER=mysql\nDB_HOST=127.0.0.1\nDB_PORT=3306\nDB_DATABASE=" + name + "\nDB_USERNAME=root\nDB_PASSWORD=\n",
		".env.example":           "APP_NAME=CopyTyGo\nAPP_ENV=local\nAPP_DEBUG=true\nAPP_HOST=127.0.0.1\nAPP_PORT=8080\nAPP_KEY=\nCOPYTYGO_STUDIO=true\nCOPYTYGO_DOCS_URL=" + version.DocsURL + "\nSERVER_READ_HEADER_TIMEOUT=5\nSERVER_READ_TIMEOUT=15\nSERVER_WRITE_TIMEOUT=30\nSERVER_IDLE_TIMEOUT=60\nSERVER_SHUTDOWN_TIMEOUT=10\nDB_DRIVER=mysql\nDB_HOST=127.0.0.1\nDB_PORT=3306\nDB_DATABASE=copytygo\nDB_USERNAME=root\nDB_PASSWORD=\n",
		"cmd/app/main.go":        strings.ReplaceAll(projectMain, "{{MODULE}}", name),
		"routes/web.go":          projectRoutes,
		"frontend/package.json":  frontendPackage,
		"frontend/tsconfig.json":  frontendTSConfig,
		"frontend/vite.config.ts": frontendViteConfig,
		"frontend/index.html":     frontendHTML,
		"frontend/src/api.ts":     frontendAPI,
		"frontend/src/forms.ts":   frontendForms,
		"frontend/src/main.ts":    frontendMain,
		"README.md":              "# " + name + "\n\nGenerated with CopyTyGo " + version.Framework + ".\n",
	}
	for p, c := range files {
		if err := os.WriteFile(filepath.Join(name, p), []byte(c), 0644); err != nil {
			return err
		}
	}
	fmt.Printf("Created CopyTyGo project %s\n", name)
	return nil
}

const projectMain = `package main

import (
    "log"
    "github.com/arfajhf/copytygo/v4/config"
    "github.com/arfajhf/copytygo/v4/core"
    "{{MODULE}}/routes"
)

func main() {
    if err := config.LoadEnv(".env"); err != nil { log.Fatal(err) }
    app := core.New()
    app.Use(core.RequestID(), core.RequestLogger(), core.SecurityHeaders(), core.BodyLimit(2<<20))
    routes.Register(app)
    log.Fatal(app.Run())
}
`
const projectRoutes = `package routes

import (
    "github.com/arfajhf/copytygo/v4/config"
    "github.com/arfajhf/copytygo/v4/core"
    "github.com/arfajhf/copytygo/v4/version"
)

func Register(app *core.Application) {
    app.Get("/api/health", func(ctx *core.Context) error {
        return ctx.JSON(core.Map{
            "framework": "CopyTyGo",
            "version": version.Framework,
            "environment": config.Get("APP_ENV", "local"),
            "status": "ok",
            "request_id": core.RequestIDValue(ctx),
        })
    }).Name("health")
}
`
const frontendPackage = `{"name":"copytygo-frontend","private":true,"scripts":{"dev":"vite","build":"tsc && vite build"},"devDependencies":{"typescript":"^5.6.0","vite":"^6.0.0"}}`
const frontendTSConfig = `{"compilerOptions":{"target":"ES2022","module":"ESNext","moduleResolution":"Bundler","strict":true,"outDir":"dist"},"include":["src","vite.config.ts"]}`
const frontendViteConfig = `import { defineConfig } from "vite";

export default defineConfig({
  server: {
    proxy: {
      "/api": "http://127.0.0.1:8080",
    },
  },
});
`
const frontendHTML = `<div id="app"></div><script type="module" src="/src/main.ts"></script>`
const frontendAPI = `export type ApiError = {
  error?: {
    status?: number;
    message?: string;
    fields?: Record<string, string[]>;
  };
  message?: string;
};

export class CopyTyGoClient {
  private token = "";

  constructor(private readonly baseURL = "") {}

  setToken(token: string) {
    this.token = token;
  }

  clearToken() {
    this.token = "";
  }

  async request<T>(path: string, init: RequestInit = {}): Promise<T> {
    const headers = new Headers(init.headers);

    if (init.body && !headers.has("Content-Type")) {
      headers.set("Content-Type", "application/json");
    }

    if (this.token) {
      headers.set("Authorization", "Bearer " + this.token);
    }

    const response = await fetch(this.baseURL + path, {
      ...init,
      headers,
    });

    const text = await response.text();
    const body = text ? JSON.parse(text) : null;

    if (!response.ok) {
      const error = body as ApiError;
      throw new Error(
        error?.error?.message ??
        error?.message ??
        "HTTP " + response.status
      );
    }

    return body as T;
  }

  get<T>(path: string) {
    return this.request<T>(path);
  }

  post<T>(path: string, data: unknown) {
    return this.request<T>(path, {
      method: "POST",
      body: JSON.stringify(data),
    });
  }

  put<T>(path: string, data: unknown) {
    return this.request<T>(path, {
      method: "PUT",
      body: JSON.stringify(data),
    });
  }

  patch<T>(path: string, data: unknown) {
    return this.request<T>(path, {
      method: "PATCH",
      body: JSON.stringify(data),
    });
  }

  delete<T>(path: string) {
    return this.request<T>(path, { method: "DELETE" });
  }
}

export const api = new CopyTyGoClient();
`
const frontendForms = `export function formJSON(form: HTMLFormElement): Record<string, string> {
  return Object.fromEntries(new FormData(form).entries()) as Record<string, string>;
}

export function fieldErrors(errors: Record<string, string[]>): string[] {
  return Object.entries(errors).flatMap(([field, messages]) =>
    messages.map(message => field + ": " + message)
  );
}
`
const frontendMain = `import { api } from "./api";\n\ntype Health = { framework: string; status: string };
const app = document.querySelector<HTMLDivElement>("#app")!;
async function boot() { const data = await api.get<Health>("/api/health"); app.innerHTML = "<h1>" + data.framework + "</h1><p>" + data.status + "</p>"; }
void boot();
`
