# CopyTyGo v2

CopyTyGo is an opinionated full-stack Go framework with a TypeScript-first frontend workflow. v2 focuses on Laravel-like developer experience while keeping the runtime native Go.

## Quick start

```powershell
go install github.com/arfajhf/copytygo/v2/cmd/ctg@latest
ctg new toko-online
cd toko-online
ctg make:resource Product name:string price:decimal stock:integer description:text?
ctg migrate
ctg dev
```

A generated resource includes its model, database-backed controller, migration, validation rules, and automatic REST route registration.

Generated endpoints:

```text
GET     /products
GET     /products/:id
POST    /products
PUT     /products/:id
DELETE  /products/:id
```

## v2 highlights

- Native Go runtime with automatic Lite Runtime fallback on supported Windows Application Control failures.
- MySQL and PostgreSQL database drivers.
- Database-backed resource CRUD in Native and Lite Runtime.
- One-line REST resources through `app.Resource(...)`.
- `ctg make:resource` with typed fields and nullable `?` syntax.
- Model and query CRUD helpers.
- Schema builder and migration engine with migrate, rollback, reset, fresh and status.
- Ready-to-use database auth with register, login, bearer tokens and role guards.
- Generated TypeScript/Vite frontend with API client and development proxy.
- `ctg update`, `ctg db:check`, and expanded `ctg doctor --db`.
- PBKDF2 password hashing, AES-GCM encryption, signed tokens and public ID masking.

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

Append `?` to make a generated column nullable:

```powershell
ctg make:resource Product name:string price:decimal description:text?
```

With no fields, CopyTyGo defaults to:

```text
name:string
```

## Authentication

Install single-role auth:

```powershell
ctg install:auth single
ctg migrate
```

Or multi-role auth:

```powershell
ctg install:auth multi
ctg migrate
```

The installer generates the users migration and registers:

```text
POST /api/auth/register
POST /api/auth/login
GET  /api/auth/me
```

Protect custom routes with:

```go
app.Get("/admin", handler).
    Middleware(auth.Middleware(), auth.RequireRole("admin"))
```

Passwords are one-way hashed. Do not encrypt passwords.

## Development runtime

`ctg dev` uses native Go first. If supported Windows Application Control blocks the temporary application executable, CopyTyGo can fall back to Lite Runtime.

```powershell
ctg dev
ctg dev --lite
```

Lite Runtime v2 supports generated database resources and generated authentication routes. It remains a development fallback; `ctg build` produces a normal native Go application.

## Health checks

```powershell
ctg doctor
ctg doctor --db
ctg db:check
```

`doctor --db` checks the project structure, Lite Runtime core, validation, generated resources, migration registry, security helpers and the live database connection.

## Core CLI

```text
ctg new <name>
ctg dev [--lite]
ctg doctor [--db]
ctg build
ctg update [version]

ctg make:resource <name> [field:type ...]
ctg make:controller <name> [--resource|--memory-resource]
ctg make:model <name>
ctg make:migration <name>
ctg make:middleware <name>
ctg make:service <name>

ctg migrate
ctg migrate:status
ctg migrate:rollback
ctg migrate:reset
ctg migrate:fresh
ctg db:check

ctg key:generate
ctg install:auth [single|multi]
ctg route:list
```

## Public IDs

```go
crypt, _ := security.NewCrypt(config.Get("APP_KEY"))
publicID, _ := crypt.UUIDCos(42)
id, _ := crypt.ResolveUUIDCos(publicID)
```

The database can keep efficient integer IDs while public URLs avoid exposing sequential IDs.

## Go module path

CopyTyGo v2 follows Go semantic import versioning:

```text
github.com/arfajhf/copytygo/v2
```

v1 projects can remain on the original module path until intentionally upgraded.
