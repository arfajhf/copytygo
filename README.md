# CopyTyGo v3

CopyTyGo is an opinionated full-stack Go framework with a TypeScript-first frontend workflow. v3 is the first production-oriented major release line: stable REST resources, database-backed auth, production diagnostics, request tracing, graceful shutdown, and Native/Lite Runtime parity.

## Quick start

```powershell
go install github.com/arfajhf/copytygo/v3/cmd/ctg@latest
ctg new toko-online
cd toko-online

ctg make:resource Product name:string price:decimal stock:integer description:text?
ctg install:auth multi
ctg migrate
ctg dev
```

Generated resources include the model, database-backed controller, migration, validation, REST route registration, pagination/search/filter/sort configuration, timestamps, and soft deletes.

## Generated resource API

```text
GET     /products
GET     /products/:id
POST    /products
PUT     /products/:id
DELETE  /products/:id
```

List resources support:

```text
/products?page=2&per_page=20
/products?q=keyboard
/products?sort=price&order=desc
/products?filter[stock]=10
```

Generated DELETE operations use soft deletes through `deleted_at`.

## v3 production highlights

- Native Go runtime with automatic Lite Runtime fallback on supported Windows Application Control failures.
- MySQL and PostgreSQL.
- Pagination, search, filtering and sorting with generated field allowlists.
- Generated soft deletes and automatic `updated_at`.
- JSON responses for 404, 405, validation and framework errors.
- Request IDs through `X-Request-ID`.
- Structured request logging with method, path, status and latency.
- HTTP server read/write/idle timeouts and graceful shutdown.
- Hardened signed tokens with minimum key length, issued-at and expiration validation.
- Production safeguards for destructive migrations.
- `ctg doctor --production` for deployment configuration checks.
- TypeScript/Vite frontend starter with reusable API client.
- Automated Git tags and GitHub Releases when a final version is merged to `main`.

## Resource field types

```text
string
text
integer
bigint
boolean
decimal
json
uuid
datetime
timestamp
```

Append `?` for nullable columns:

```powershell
ctg make:resource Product name:string price:decimal description:text?
```

Generated validation includes integer, numeric, boolean and UUID rules where appropriate.

## Authentication

Single role:

```powershell
ctg install:auth single
ctg migrate
```

Multi role:

```powershell
ctg install:auth multi
ctg migrate
```

Generated endpoints:

```text
POST /api/auth/register
POST /api/auth/login
GET  /api/auth/me
```

Protect routes:

```go
app.Get("/admin", handler).
    Middleware(auth.Middleware(), auth.RequireRole("admin"))
```

## Development runtime

`ctg dev` tries native Go first.

```powershell
ctg dev
ctg dev --lite
```

Lite Runtime v3 supports generated database resources, pagination/search/filter/sort, validation, soft-delete options and generated authentication routes. It remains a development fallback. Production uses a normal Go binary built through:

```powershell
ctg build
```

## Production verification

Before deployment:

```powershell
ctg doctor --db --production
```

Production expectations include:

```text
APP_ENV=production
APP_DEBUG=false
APP_KEY=<strong generated key>
```

Generated applications also support:

```text
SERVER_READ_HEADER_TIMEOUT=5
SERVER_READ_TIMEOUT=15
SERVER_WRITE_TIMEOUT=30
SERVER_IDLE_TIMEOUT=60
SERVER_SHUTDOWN_TIMEOUT=10
```

## Migration safety

Normal migration:

```powershell
ctg migrate
```

Destructive migration commands require `--force` when `APP_ENV=production`:

```powershell
ctg migrate:rollback --force
ctg migrate:reset --force
ctg migrate:fresh --force
```

## Core CLI

```text
ctg new <name>
ctg dev [--lite]
ctg doctor [--db] [--production]
ctg build
ctg update [version]
ctg route:list

ctg make:resource <name> [field:type ...]
ctg make:controller <name> [--resource|--memory-resource]
ctg make:model <name>
ctg make:migration <name>
ctg make:middleware <name>
ctg make:service <name>

ctg migrate
ctg migrate:status
ctg migrate:rollback [--force]
ctg migrate:reset [--force]
ctg migrate:fresh [--force]
ctg db:check

ctg key:generate
ctg install:auth [single|multi]
```

## Public IDs

```go
crypt, _ := security.NewCrypt(config.Get("APP_KEY"))
publicID, _ := crypt.UUIDCos(42)
id, _ := crypt.ResolveUUIDCos(publicID)
```

## Go module path

CopyTyGo v3 follows Go semantic import versioning:

```text
github.com/arfajhf/copytygo/v3
```

v2 applications can remain on `github.com/arfajhf/copytygo/v2` until intentionally upgraded.
