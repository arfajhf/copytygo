package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/arfajhf/copytygo/version"
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
		"go.mod":                 "module " + name + "\n\ngo 1.27.1\n\nrequire github.com/arfajhf/copytygo " + version.StableModule + "\n",
		".env":                   "APP_NAME=" + name + "\nAPP_ENV=local\nAPP_DEBUG=true\nAPP_HOST=127.0.0.1\nAPP_PORT=8080\nAPP_KEY=" + key + "\nDB_DRIVER=mysql\nDB_HOST=127.0.0.1\nDB_PORT=3306\nDB_DATABASE=" + name + "\nDB_USERNAME=root\nDB_PASSWORD=\n",
		".env.example":           "APP_NAME=CopyTyGo\nAPP_ENV=local\nAPP_DEBUG=true\nAPP_HOST=127.0.0.1\nAPP_PORT=8080\nAPP_KEY=\nDB_DRIVER=mysql\nDB_HOST=127.0.0.1\nDB_PORT=3306\nDB_DATABASE=copytygo\nDB_USERNAME=root\nDB_PASSWORD=\n",
		"cmd/app/main.go":        strings.ReplaceAll(projectMain, "{{MODULE}}", name),
		"routes/web.go":          projectRoutes,
		"frontend/package.json":  frontendPackage,
		"frontend/tsconfig.json": frontendTSConfig,
		"frontend/index.html":    frontendHTML,
		"frontend/src/main.ts":   frontendMain,
		"README.md":              "# " + name + "\n\nGenerated with CopyTyGo " + version.StableModule + ".\n",
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
    "github.com/arfajhf/copytygo/config"
    "github.com/arfajhf/copytygo/core"
    "{{MODULE}}/routes"
)

func main() {
    if err := config.LoadEnv(".env"); err != nil { log.Fatal(err) }
    app := core.New()
    app.Use(core.SecurityHeaders(), core.BodyLimit(2<<20))
    routes.Register(app)
    log.Fatal(app.Run())
}
`
const projectRoutes = `package routes

import "github.com/arfajhf/copytygo/core"

func Register(app *core.Application) {
    app.Get("/api/health", func(ctx *core.Context) error {
        return ctx.JSON(core.Map{"framework":"CopyTyGo","status":"ok"})
    }).Name("health")
}
`
const frontendPackage = `{"name":"copytygo-frontend","private":true,"scripts":{"dev":"vite","build":"tsc && vite build"},"devDependencies":{"typescript":"^5.6.0","vite":"^6.0.0"}}`
const frontendTSConfig = `{"compilerOptions":{"target":"ES2022","module":"ESNext","moduleResolution":"Bundler","strict":true,"outDir":"dist"},"include":["src"]}`
const frontendHTML = `<div id="app"></div><script type="module" src="/src/main.ts"></script>`
const frontendMain = `type Health = { framework: string; status: string };
const app = document.querySelector<HTMLDivElement>("#app")!;
async function boot() { const res = await fetch("/api/health"); const data = await res.json() as Health; app.innerHTML = "<h1>" + data.framework + "</h1><p>" + data.status + "</p>"; }
void boot();
`
