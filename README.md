<p><img src="branding/logo.png" alt="CopyTyGo logo" width="100"></p>

# CopyTyGo v5 (release candidate)

CopyTyGo is a full-stack Go framework focused on productive application development without hiding the strengths of Go.

CopyTyGo v5 is planned as the first public stable release. The earlier v4.0.6 tag was an internal prelaunch milestone. v5 focuses on Go + TypeScript developer experience, secure backend tooling, documentation, production reliability and release quality.

> **Prelaunch release candidate.** v5 has not been published as a stable module yet, and internal v4 tags are not recommended as the public onboarding path. Go 1.27.1+ and Node.js 22+ are needed for full frontend development.

## Quick start

Until v5.0.0 is tagged and verified, this branch is for release-candidate testing. Build the CLI locally rather than installing an unpublished version:

```bash
git clone -b feature/v5-release-candidate https://github.com/arfajhf/copytygo.git
cd copytygo
go build -o ctg ./cmd/ctg
./ctg version
```

After the **v5.0.0 stable tag is actually published and tested**, the public installation command will be:

```bash
go install github.com/arfajhf/copytygo/v5/cmd/ctg@v5.0.0
ctg new
```

For testing generated v5 applications before publication, follow [the v5 release-candidate upgrade and local module instructions](docs/UPGRADE_V5.md). Do not use a local `replace` directive in published projects.

The interactive installer can configure:

- MySQL or PostgreSQL
- no auth, single-role auth, or multi-role auth
- TypeScript, React, Vue, or API-only frontend
- CopyTyGo Studio

Ensure the Go binary directory, normally `%USERPROFILE%\go\bin` on Windows, is on your PATH.

The classic non-interactive workflow remains available:

```powershell
ctg new toko-online
cd toko-online
ctg dev
```

When the application starts locally:

```text
Application  : http://127.0.0.1:8080
Studio       : http://127.0.0.1:8080/__copytygo
Documentation: ...
```

The default application route renders the CopyTyGo welcome page with links to Documentation, Studio, and GitHub.

## Source formatting and environment configuration

Generated `.env` and `.env.example` files now separate the application, Studio, server, database, queue, storage and mail settings with one blank line between groups. Existing projects can apply the same layout without changing their key values:

```bash
ctg env:format
```

The formatter preserves existing values, comments and file permissions, and refuses symbolic-link dotenv files. The starter also generates an `.editorconfig` for Go, TypeScript, JSON, Vue and CSS indentation. Go files are formatted before being written, and frontend package/TS config files are emitted as readable multiline JSON.

## CopyTyGo Studio

Studio is a local development workspace bundled with the framework.

```text
/__copytygo
├── Dashboard
├── Routes
├── Models
├── Resources
├── Database
├── Migrations
├── Auth
├── Services
├── Queue
├── Scheduler
├── Requests
├── Errors
├── Logs
├── Generator
├── Health
└── Doctor
```

Studio is disabled when `APP_ENV=production`.

You can open the application or Studio directly:

```powershell
ctg open
ctg studio
```

## Backend ecosystem

The existing backend ecosystem carried forward toward v5 includes:

- MySQL and PostgreSQL
- migrations and schema DSL
- query builder and database resources
- transactions
- model relationships
- seeders and factories
- API resources and serialization
- database-backed authentication
- authorization policies
- encrypted sessions and flash data
- cache abstraction
- events and listeners
- local storage and uploads
- SMTP mail and mail templates
- scheduler with interval and cron support
- memory, synchronous, and durable database queues
- retries and backoff
- runtime health checks
- structured logs and request/error inspection
- HTTP client with safe retry support
- testing client and framework fakes

## Queue drivers

Generated projects use:

```env
QUEUE_DRIVER=memory
QUEUE_WORKERS=1
QUEUE_POLL_SECONDS=1
QUEUE_BACKOFF_SECONDS=1
```

Available drivers:

```text
sync
memory
database
```

The database driver persists jobs in MySQL/PostgreSQL and uses dedicated CopyTyGo queue and failed-job tables.

Generate a named job:

```powershell
ctg make:job SendWelcomeEmail
```

Dispatch it through the configured driver:

```go
queue.DispatchNamed(ctx, "SendWelcomeEmail", queue.Payload{
    "user_id": 42,
})
```

Delayed jobs are supported through `queue.DispatchOptions`.

## Scheduler

Interval scheduling:

```go
scheduler.Default.Hourly("reports", func(ctx context.Context) error {
    return nil
})
```

Cron scheduling:

```go
scheduler.Default.Cron(
    "weekday-report",
    "0 8 * * 1-5",
    func(ctx context.Context) error {
        return nil
    },
)
```

## Generated resources

```powershell
ctg make:resource Product name:string price:decimal stock:integer active:boolean description:text?
# v5 defaults to private resource endpoints. Use --public only when explicitly intended.
```

This generates:

- model
- database-backed controller
- migration
- validation
- REST routes
- search/filter/sort allowlists
- timestamps
- soft deletes

Resource API (private by default; browser sessions require CSRF on writes, API-only uses a bearer token and fresh database role checks; see [resource security](docs/RESOURCE_SECURITY.md)):

```text
GET     /products
GET     /products/:id
POST    /products
PUT     /products/:id
DELETE  /products/:id
```

List options:

```text
/products?page=2&per_page=20
/products?q=keyboard
/products?sort=price&order=desc
/products?filter[stock]=10
```

## Authentication

Install from the CLI:

```powershell
ctg install:auth single
ctg install:auth multi
```

The installer creates editable login, registration, dashboard and account pages. After setting your database connection in `.env`, run:

```powershell
ctg migrate
ctg dev
```

The welcome navbar shows **Log in** and **Register** when those routes are installed. Login and registration redirect to `/dashboard`. Multi-role administrators get user statistics and a Users menu with search, pagination, create, edit and delete forms.

Browser routes:

```text
GET/POST /login
GET/POST /register
GET      /dashboard
GET      /account
POST     /logout
```

The TypeScript frontend provides login/register, role-aware dashboards and Users CRUD. Go handles JSON endpoints, encrypted HttpOnly sessions, CSRF, validation and permissions. `ctg dev` starts native Go + Vite automatically; `ctg build` compiles both. Production cookies require HTTPS.

JSON API routes remain available:

```text
POST /api/auth/register
POST /api/auth/login
GET  /api/auth/me
```

Edit `frontend/src/auth/*.ts` and `style.css` for the UI, `routes/auth.go` for JSON routes, and `app/auth/web.go` for the default registration role. TypeScript changes reload through Vite; restart `ctg dev` after Go changes. Read [the authentication guide](docs/AUTHENTICATION.md) for setup, customization and upgrades.

For multi-role auth, register your first account, then promote it from your own terminal inside the project:

```powershell
ctg auth:admin your-email@example.com
```

The dashboard reads the current role from the database on every browser request. Public registration creates a regular user. Administrator CRUD requires server authorization and CSRF, never displays password hashes, and prevents removing your own admin account or role.

Studio includes a safe user inspector. Password hashes are never selected by the Studio inspector.

## Sessions

Sessions are encrypted with AES-GCM using `APP_KEY`.

```go
sessions := session.New()

_ = sessions.Put(ctx, "user_id", "42")
userID := sessions.Get(ctx, "user_id")

_ = sessions.Flash(ctx, "status", "Saved")
message, ok, _ := sessions.PullFlash(ctx, "status")
```

## Route groups and API versions

```go
api := app.APIVersion("v1")

api.Group("/admin").
    Name("admin.").
    Routes(func(routes *core.RouteGroup) {
        routes.Get("/users/:id", handler).Name("users.show")
    })
```

Named URL:

```go
url, ok := app.URL(
    "api.v1.admin.users.show",
    map[string]string{"id": "42"},
)
```

## Runtime health

Generated applications expose:

```text
GET /api/health
```

The default health registry checks:

- database
- cache
- storage

Unhealthy applications return HTTP `503`.

Studio also exposes a visual Runtime Health dashboard.

## Development runtime

```powershell
ctg dev
```

On Windows, this compiles and runs `build/dev/app.exe` from the project instead of running a temporary `go run` executable. On Linux/macOS, the default remains `go run`. Use `ctg dev --build` for a project-local build on any OS. This does **not** bypass Windows Defender or App Control: the CLI preserves security-policy errors.

If Smart App Control blocks Windows `go.exe`, the full Go + TypeScript Native Runtime can run inside WSL2 instead. Use `ctg dev` inside WSL, or the optional `ctg dev --wsl` from an **allowed** Windows CLI with WSL configured. See [WSL Native development](docs/WSL_DEVELOPMENT.md), [Windows diagnostics](docs/WINDOWS_DEVELOPMENT.md) and the [v5 launch checklist](docs/V5_LAUNCH_CHECKLIST.md).


CopyTyGo runs the complete generated Go application in Native Runtime on every `ctg dev` startup. Runtime decisions cached by older CLI versions are ignored, including after installing auth or changing project source.

`ctg dev --native` remains a compatible alias for the default command. Compilation, application and operating-system execution errors remain visible; the CLI does not automatically switch runtimes.

Opt in to Lite explicitly:

```powershell
ctg dev --lite
```

Lite Runtime includes the welcome page, Studio inspection tools, and the same resource/scaffold generators as Native Studio. Request, error and log runtime inspection remain available in Native Studio. Execution of arbitrary project Go background jobs remains Native-only by design.

Production always uses the native application binary:

```powershell
ctg build
```

For auth projects, `ctg build` packages the compiled TypeScript frontend alongside Go. Deploy the complete `build/` directory and start the binary from that directory. Production does not require Node.js. See [authentication setup and deployment](docs/AUTHENTICATION.md).

## Public images and assets

Store public assets in `frontend/public/`. The starter includes `logo.png` and `images/`.
Use `/logo.png` or `/images/banner.png` in TypeScript, without the `frontend/public` prefix.
`ctg dev` serves new images immediately; `ctg build` includes them in the production bundle.
Missing default assets are restored during auth installation, development and builds while custom files are preserved.
Read [the public asset guide](docs/PUBLIC_ASSETS.md) for paths and examples.

## Visual and CLI generators

CLI:

```text
ctg make:controller <name>
ctg make:resource <name> [field:type ...] [--access=default|authenticated|admin|user|public]
ctg resource:sync
ctg make:model <name>
ctg make:migration <name>
ctg make:middleware <name>
ctg make:service <name>
ctg make:job <name>
ctg make:listener <name>
ctg make:seeder <name>
ctg make:factory <name>
ctg make:mail <name>
```

The same ecosystem can be generated from Studio's Visual Generator.

## Production checks

Before deployment:

```powershell
ctg doctor --db --production
ctg migrate:status
ctg build
```

Recommended production values:

```env
APP_ENV=production
APP_DEBUG=false
COPYTYGO_STUDIO=false
APP_KEY=<strong generated key>
QUEUE_DRIVER=database
```

Studio does not register in production even if the local development setting was left enabled.

## Go module path

CopyTyGo v5 uses Go semantic import versioning:

```text
github.com/arfajhf/copytygo/v5
```

Existing v4 applications may remain on:

```text
github.com/arfajhf/copytygo/v4
```

until intentionally upgraded. See the [v5 upgrade guide](docs/UPGRADE_V5.md) and [documentation index](docs/INDEX.md). The candidate module is not installable from a public final tag yet.
