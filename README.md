# CopyTyGo v4

CopyTyGo is a full-stack Go framework focused on productive application development without hiding the strengths of Go.

v4 is the **Complete Framework + Studio** milestone. It combines the production foundation from v3 with a broader backend ecosystem, visual development tooling, richer project starters, and Native/Lite Runtime parity.

> Stable release: `v4.0.3`. Go 1.27.1 or newer is required. Node.js is needed only for frontend development.

## Quick start

Install the final v4 CLI, then create a project:

```powershell
go install github.com/arfajhf/copytygo/v4/cmd/ctg@v4.0.3
ctg new
```

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

CopyTyGo v4 includes:

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

Resource API:

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

The installer creates editable login, registration and account pages. After setting your database connection in `.env`, run:

```powershell
ctg migrate
ctg dev
```

The welcome page shows **Log in** and **Register** when those routes are installed. Registration signs the user in and redirects to `/account`.

Browser routes:

```text
GET/POST /login
GET/POST /register
GET      /account
POST     /logout
```

Browser forms use encrypted HttpOnly cookie sessions, CSRF protection, password confirmation, and inline validation messages. Production cookies require HTTPS. These pages run in Native Runtime and do not require a frontend build.

JSON API routes remain available:

```text
POST /api/auth/register
POST /api/auth/login
GET  /api/auth/me
```

Edit `routes/auth.go` for routes, `app/auth/web.go` for the default registration role, and `app/auth/views/*.html` for the UI. Restart `ctg dev` after changing embedded views. Read [the authentication guide](docs/AUTHENTICATION.md) for setup, customization and upgrades.

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

## Visual and CLI generators

CLI:

```text
ctg make:controller <name>
ctg make:resource <name> [field:type ...]
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

CopyTyGo v4 follows Go semantic import versioning:

```text
github.com/arfajhf/copytygo/v4
```

Existing v3 applications may remain on:

```text
github.com/arfajhf/copytygo/v3
```

until intentionally upgraded.
