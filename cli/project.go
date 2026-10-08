package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/arfajhf/copytygo/v4/version"
)

type ProjectOptions struct {
	Database string
	Auth     string
	Studio   bool
	Frontend string
}

func DefaultProjectOptions() ProjectOptions {
	return ProjectOptions{
		Database: "mysql",
		Auth:     "none",
		Studio:   true,
		Frontend: "typescript",
	}
}

func NewProject(name string) error {
	return NewProjectWithOptions(name, DefaultProjectOptions())
}

func NewProjectWithOptions(name string, options ProjectOptions) error {
	if name == "" || strings.ContainsAny(name, "/\\") {
		return fmt.Errorf("copytygo: invalid project name")
	}
	if _, err := os.Stat(name); err == nil {
		return fmt.Errorf("copytygo: directory %q already exists", name)
	}

	options.Database = strings.ToLower(strings.TrimSpace(options.Database))
	switch options.Database {
	case "mysql", "postgres":
	default:
		return fmt.Errorf("copytygo: database must be mysql or postgres")
	}

	options.Auth = strings.ToLower(strings.TrimSpace(options.Auth))
	switch options.Auth {
	case "", "none":
		options.Auth = "none"
	case "single", "multi":
	default:
		return fmt.Errorf("copytygo: auth must be none, single or multi")
	}

	options.Frontend = strings.ToLower(strings.TrimSpace(options.Frontend))
	switch options.Frontend {
	case "", "typescript":
		options.Frontend = "typescript"
	case "react", "vue", "api":
	default:
		return fmt.Errorf("copytygo: frontend must be typescript, react, vue or api")
	}

	dirs := []string{
		"cmd/app",
		"app/controllers",
		"app/models",
		"app/middleware",
		"app/services",
		"app/jobs",
		"app/listeners",
		"app/mails",
		"database/migrations",
		"database/seeders",
		"database/factories",
		"routes",
	}
	if options.Frontend != "api" {
		dirs = append(dirs, "frontend/src", "frontend/public")
	}
	for _, d := range dirs {
		if err := os.MkdirAll(filepath.Join(name, d), 0755); err != nil {
			return err
		}
	}

	key, err := GenerateKey()
	if err != nil {
		return err
	}

	dbPort := "3306"
	dbUser := "root"
	if options.Database == "postgres" {
		dbPort = "5432"
		dbUser = "postgres"
	}

	studioValue := "false"
	if options.Studio {
		studioValue = "true"
	}

	env := "APP_NAME=" + name +
		"\nAPP_ENV=local" +
		"\nAPP_DEBUG=true" +
		"\nAPP_HOST=127.0.0.1" +
		"\nAPP_PORT=8080" +
		"\nAPP_KEY=" + key +
		"\nCOPYTYGO_STUDIO=" + studioValue +
		"\nCOPYTYGO_DOCS_URL=" + version.DocsURL +
		"\nSERVER_READ_HEADER_TIMEOUT=5" +
		"\nSERVER_READ_TIMEOUT=15" +
		"\nSERVER_WRITE_TIMEOUT=30" +
		"\nSERVER_IDLE_TIMEOUT=60" +
		"\nSERVER_SHUTDOWN_TIMEOUT=10" +
		"\nDB_DRIVER=" + options.Database +
		"\nDB_HOST=127.0.0.1" +
		"\nDB_PORT=" + dbPort +
		"\nDB_DATABASE=" + name +
		"\nDB_USERNAME=" + dbUser +
		"\nDB_PASSWORD=" +
		"\nQUEUE_DRIVER=memory" +
		"\nQUEUE_WORKERS=1" +
		"\nQUEUE_POLL_SECONDS=1" +
		"\nQUEUE_BACKOFF_SECONDS=1" +
		"\nSTORAGE_PATH=storage/app" +
		"\nMAIL_HOST=127.0.0.1" +
		"\nMAIL_PORT=1025" +
		"\nMAIL_USERNAME=" +
		"\nMAIL_PASSWORD=" +
		"\nMAIL_FROM=noreply@copytygo.local\n"

	envExample := "APP_NAME=CopyTyGo" +
		"\nAPP_ENV=local" +
		"\nAPP_DEBUG=true" +
		"\nAPP_HOST=127.0.0.1" +
		"\nAPP_PORT=8080" +
		"\nAPP_KEY=" +
		"\nCOPYTYGO_STUDIO=true" +
		"\nCOPYTYGO_DOCS_URL=" + version.DocsURL +
		"\nSERVER_READ_HEADER_TIMEOUT=5" +
		"\nSERVER_READ_TIMEOUT=15" +
		"\nSERVER_WRITE_TIMEOUT=30" +
		"\nSERVER_IDLE_TIMEOUT=60" +
		"\nSERVER_SHUTDOWN_TIMEOUT=10" +
		"\nDB_DRIVER=" + options.Database +
		"\nDB_HOST=127.0.0.1" +
		"\nDB_PORT=" + dbPort +
		"\nDB_DATABASE=copytygo" +
		"\nDB_USERNAME=" + dbUser +
		"\nDB_PASSWORD=" +
		"\nQUEUE_DRIVER=memory" +
		"\nQUEUE_WORKERS=1" +
		"\nQUEUE_POLL_SECONDS=1" +
		"\nQUEUE_BACKOFF_SECONDS=1" +
		"\nSTORAGE_PATH=storage/app" +
		"\nMAIL_HOST=127.0.0.1" +
		"\nMAIL_PORT=1025" +
		"\nMAIL_USERNAME=" +
		"\nMAIL_PASSWORD=" +
		"\nMAIL_FROM=noreply@copytygo.local\n"

	files := map[string]string{
		"go.mod":          "module " + name + "\n\ngo 1.27.1\n\nrequire github.com/arfajhf/copytygo/v4 " + version.Module() + "\n",
		".env":            env,
		".env.example":    envExample,
		"cmd/app/main.go": strings.ReplaceAll(projectMain, "{{MODULE}}", name),
		"routes/web.go":   projectRoutes,
		"README.md": "# " + name + "\n\nGenerated with CopyTyGo " + version.Framework +
			".\n\nDatabase: " + options.Database +
			"\nFrontend: " + options.Frontend +
			"\nAuth: " + options.Auth + "\n",
	}

	if options.Frontend != "api" {
		files["README.md"] += "\nPublic assets: frontend/public/ (logo.png and images/).\nUse /images/banner.png in TypeScript. See frontend/ASSETS.md.\n"
		files["routes/web.go"] = strings.Replace(projectRoutes, `"github.com/arfajhf/copytygo/v4/core"`, `"github.com/arfajhf/copytygo/v4/core"`+"\n"+`"github.com/arfajhf/copytygo/v4/frontend"`, 1)
		files["routes/web.go"] = strings.Replace(files["routes/web.go"], "func Register(app *core.Application) {", "func Register(app *core.Application) {\n frontend.Assets(app)", 1)
		if err := ensurePublicAssets(name); err != nil {
			return err
		}
	}
	for path, content := range frontendFiles(options.Frontend) {
		files[path] = content
	}

	for path, content := range files {
		if err := os.WriteFile(filepath.Join(name, path), []byte(content), 0644); err != nil {
			return err
		}
	}

	if err := GenerateJobRegistry(filepath.Join(name, "app", "jobs")); err != nil {
		return err
	}

	if options.Auth != "none" {
		if err := InstallAuth(options.Auth, name); err != nil {
			return err
		}
	}

	fmt.Println()
	fmt.Printf("Created CopyTyGo project %s\n", name)
	fmt.Printf("Database : %s\n", options.Database)
	fmt.Printf("Frontend : %s\n", options.Frontend)
	fmt.Printf("Auth     : %s\n", options.Auth)
	fmt.Printf("Studio   : %t\n", options.Studio)
	fmt.Println()
	fmt.Printf("Next: cd %s && ctg dev\n", name)
	return nil
}

func frontendFiles(kind string) map[string]string {
	switch kind {
	case "api":
		return map[string]string{}

	case "react":
		return map[string]string{
			"frontend/package.json":  `{"name":"copytygo-react","private":true,"scripts":{"dev":"vite","build":"tsc && vite build"},"dependencies":{"react":"^19.0.0","react-dom":"^19.0.0"},"devDependencies":{"@types/node":"^24.0.0","@types/react":"^19.0.0","@types/react-dom":"^19.0.0","@vitejs/plugin-react":"^5.0.0","typescript":"^5.9.0","vite":"^6.0.0"}}`,
			"frontend/tsconfig.json": `{"compilerOptions":{"target":"ES2022","lib":["ES2022","DOM","DOM.Iterable","ESNext.Disposable"],"module":"ESNext","moduleResolution":"Bundler","strict":true,"jsx":"react-jsx","types":["vite/client","node"],"noEmit":true},"include":["src/**/*.ts","src/**/*.tsx","vite.config.ts"]}`,
			"frontend/vite.config.ts": `import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
export default defineConfig({ plugins:[react()], server:{ proxy:{ "/api":"http://127.0.0.1:8080" } } });`,
			"frontend/index.html": `<link rel="icon" href="/logo.png"><div id="root"></div><script type="module" src="/src/main.tsx"></script>`,
			"frontend/src/main.tsx": `import React from "react";
import { createRoot } from "react-dom/client";
createRoot(document.getElementById("root")!).render(<React.StrictMode><main><img src="/logo.png" alt="CopyTyGo logo" width="72" height="72" style={{objectFit:"contain"}} /><h1>CopyTyGo + React</h1><p>Your frontend is ready.</p></main></React.StrictMode>);`,
			"frontend/src/api.ts": frontendAPI,
		}

	case "vue":
		return map[string]string{
			"frontend/package.json":  `{"name":"copytygo-vue","private":true,"scripts":{"dev":"vite","build":"vue-tsc --noEmit && vite build"},"dependencies":{"vue":"^3.5.0"},"devDependencies":{"@types/node":"^24.0.0","@vitejs/plugin-vue":"^6.0.0","typescript":"^5.9.0","vite":"^6.0.0","vue-tsc":"^3.0.0"}}`,
			"frontend/tsconfig.json": `{"compilerOptions":{"target":"ES2022","lib":["ES2022","DOM","DOM.Iterable","ESNext.Disposable"],"module":"ESNext","moduleResolution":"Bundler","strict":true,"types":["vite/client","node"],"noEmit":true},"include":["src/**/*.ts","src/**/*.vue","vite.config.ts"]}`,
			"frontend/vite.config.ts": `import { defineConfig } from "vite";
import vue from "@vitejs/plugin-vue";
export default defineConfig({ plugins:[vue()], server:{ proxy:{ "/api":"http://127.0.0.1:8080" } } });`,
			"frontend/index.html": `<link rel="icon" href="/logo.png"><div id="app"></div><script type="module" src="/src/main.ts"></script>`,
			"frontend/src/main.ts": `import { createApp } from "vue";
import App from "./App.vue";
createApp(App).mount("#app");`,
			"frontend/src/App.vue": `<template><main><img src="/logo.png" alt="CopyTyGo logo" width="72" height="72" style="object-fit:contain"><h1>CopyTyGo + Vue</h1><p>Your frontend is ready.</p></main></template>`,
			"frontend/src/api.ts":  frontendAPI,
			"frontend/src/vite-env.d.ts": `/// <reference types="vite/client" />
declare module "*.vue" {
  import type { DefineComponent } from "vue";
  const component: DefineComponent<Record<string, never>, Record<string, never>, any>;
  export default component;
}`,
		}

	default:
		return map[string]string{
			"frontend/package.json":   frontendPackage,
			"frontend/tsconfig.json":  frontendTSConfig,
			"frontend/vite.config.ts": frontendViteConfig,
			"frontend/index.html":     frontendHTML,
			"frontend/src/api.ts":     frontendAPI,
			"frontend/src/forms.ts":   frontendForms,
			"frontend/src/main.ts":    frontendMain,
		}
	}
}

const projectMain = `package main

import (
    "log"
    "github.com/arfajhf/copytygo/v4/config"
    "github.com/arfajhf/copytygo/v4/core"
    "github.com/arfajhf/copytygo/v4/foundation"
    "github.com/arfajhf/copytygo/v4/studio"
    "github.com/arfajhf/copytygo/v4/queue"
    "github.com/arfajhf/copytygo/v4/scheduler"
    "{{MODULE}}/app/jobs"
    "{{MODULE}}/routes"
)

func main() {
    if err := config.LoadEnv(".env"); err != nil { log.Fatal(err) }
    app := core.New()
    if err := foundation.RegisterDefaults(app.Services); err != nil { log.Fatal(err) }
    if err := jobs.Register(queue.DefaultRegistry); err != nil { log.Fatal(err) }
    app.Use(core.RequestID(), core.RequestLogger(), core.SecurityHeaders(), core.BodyLimit(2<<20))
    if err := queue.AttachConfigured(app); err != nil { log.Fatal(err) }
    scheduler.Attach(app, scheduler.Default)
    routes.Register(app)
    studio.Register(app)
    log.Fatal(app.Run())
}
`
const projectRoutes = `package routes

import (
    "net/http"

    "github.com/arfajhf/copytygo/v4/config"
    "github.com/arfajhf/copytygo/v4/core"
    "github.com/arfajhf/copytygo/v4/foundation"
    "github.com/arfajhf/copytygo/v4/version"
)

func Register(app *core.Application) {
    app.Get("/", core.Welcome()).Name("welcome")

    app.Get("/api/health", func(ctx *core.Context) error {
        status, checks, err := foundation.RunHealth(ctx.Request.Context(), app.Services)
        if err != nil {
            return err
        }
        if string(status) != "healthy" {
            ctx.Status(http.StatusServiceUnavailable)
        }
        return ctx.JSON(core.Map{
            "framework": "CopyTyGo",
            "version": version.Framework,
            "environment": config.Get("APP_ENV", "local"),
            "status": status,
            "checks": checks,
            "request_id": core.RequestIDValue(ctx),
        })
    }).Name("health")
}
`
const frontendPackage = `{"name":"copytygo-frontend","private":true,"scripts":{"dev":"vite","build":"tsc && vite build"},"devDependencies":{"@types/node":"^24.0.0","typescript":"^5.9.0","vite":"^6.0.0"}}`
const frontendTSConfig = `{"compilerOptions":{"target":"ES2022","lib":["ES2022","DOM","DOM.Iterable","ESNext.Disposable"],"module":"ESNext","moduleResolution":"Bundler","strict":true,"types":["vite/client","node"],"noEmit":true},"include":["src","vite.config.ts"]}`
const frontendViteConfig = `import { defineConfig } from "vite";

export default defineConfig({
  server: {
    proxy: {
      "/api": "http://127.0.0.1:8080",
    },
  },
});
`
const frontendHTML = `<link rel="icon" href="/logo.png"><div id="app"></div><script type="module" src="/src/main.ts"></script>`
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
const frontendMain = `import { api } from "./api";

type Health = { framework: string; status: string };
const app = document.querySelector<HTMLDivElement>("#app")!;

async function boot() {
  const data = await api.get<Health>("/api/health");
  app.innerHTML = '<img src="/logo.png" alt="CopyTyGo logo" width="72" height="72" style="object-fit:contain">' + "<h1>" + data.framework + "</h1><p>" + data.status + "</p>";
}

void boot();
`
