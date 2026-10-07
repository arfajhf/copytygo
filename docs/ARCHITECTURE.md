# CopyTyGo v4 architecture

CopyTyGo v4 is organized around a production HTTP core, application services, background workers, and a local developer workspace called CopyTyGo Studio.

## Application flow

```text
Client
  -> Request ID
  -> Request logger
  -> Security middleware
  -> Application middleware
  -> Route middleware
  -> Handler
  -> Context response
```

`core.Application` implements `http.Handler`, owns the router, service container, and background-service lifecycle.

## Application services

Every generated application owns a service container:

```text
Application
  -> Services
       -> Cache
       -> Events
       -> Mail
       -> Storage
       -> Queue
       -> Scheduler
       -> Health
       -> Custom services
```

Default services are registered through `foundation.RegisterDefaults(app.Services)`.

The container supports normal bindings, singletons, explicit instances, typed resolution, and replacement in tests.

## Routing

CopyTyGo supports:

- normal routes
- middleware
- named routes
- nested route groups
- name prefixes
- grouped resource routes
- API version helpers

Example:

```go
api := app.APIVersion("v1")
admin := api.Group("/admin").Name("admin.")

admin.Get("/users/:id", handler).
    Name("users.show")
```

## Database

```text
Handler
  -> Context database helper
  -> database.Resource / model.Model
  -> query.Builder
  -> database/sql
  -> MySQL or PostgreSQL
```

The query builder validates table/column identifiers and supported operators.

Generated resources support pagination, search, filtering, sorting, timestamps, and soft deletes.

Transactions are available through the database transaction helper.

Model relationship helpers include:

- HasOne
- HasMany
- BelongsTo

## Migrations

```text
Project migration source
  -> AST migration loader
  -> migration registry
  -> schema blueprint
  -> MySQL/PostgreSQL dialect
  -> SQL
  -> migration repository
```

Studio and CLI use the same migration engine.

Destructive production migration commands require `--force`.

## Authentication and authorization

```text
Register/Login
  -> database user service
  -> PBKDF2 password verification
  -> signed authentication token
  -> authentication middleware
  -> claims on Context
  -> optional role middleware / policy gate
```

Browser authentication adds rendered login/register/account pages and stores the signed token inside an encrypted HttpOnly web-session cookie. Form submissions verify the session's CSRF value; sign-in rotates both the session and CSRF value. Bearer-token API middleware remains independent. Generated templates are embedded in the application binary and editable in app/auth/views.

Studio's Auth inspector only queries safe user fields and never selects password hashes.

## Sessions

Session cookies use AES-GCM authenticated encryption derived from `APP_KEY`.

Sessions support:

- Get / Put / Remove
- full Write
- flash data
- regeneration
- forget/logout
- HttpOnly
- SameSite
- Secure in production

Request-scoped state keeps repeated session mutations consistent before the final encrypted cookie is written.

## Queue architecture

CopyTyGo supports three queue drivers:

```text
sync
  -> execute named job immediately

memory
  -> in-process worker
  -> retry/backoff
  -> delayed dispatch

database
  -> MySQL/PostgreSQL jobs table
  -> worker reservation
  -> retry/release
  -> failed jobs table
```

Named jobs are registered in the project job registry.

The durable database queue persists jobs across application restarts.

## Scheduler

Scheduler entries may use:

- fixed intervals
- EveryMinute
- Hourly
- Daily
- five-field cron expressions

The runtime tracks next run, last run, run count, and last error.

## Background lifecycle

`Application.Background` registers long-running services against an application context.

On shutdown:

```text
SIGINT / SIGTERM
  -> cancel background context
  -> stop queue/scheduler workers
  -> HTTP graceful shutdown
```

## Cache, events, mail and storage

Cache:
- Store abstraction
- in-memory TTL driver
- Remember helper

Events:
- named event bus
- multiple listeners
- listener inspection

Mail:
- SMTP
- HTML/text template rendering
- fake mailer for tests

Storage:
- Disk abstraction
- local driver
- traversal protection
- uploads with maximum-size enforcement

## Health and observability

Runtime health is separate from project diagnostics.

```text
Runtime Health
  -> Database
  -> Cache
  -> Storage
```

Generated `/api/health` returns `503` when the overall state is unhealthy.

Observability includes:

- request IDs
- structured logs
- request ring buffer
- error/panic inspector
- health latency
- queue/scheduler status

## Studio

Studio is a development-only surface on:

```text
/__copytygo
```

It uses the same engines as the CLI rather than duplicating framework behavior.

Studio can inspect:

- routes
- models
- resources
- database
- migrations
- auth
- services
- queue
- scheduler
- requests
- errors
- logs
- health
- doctor results

It also exposes visual code generation.

Studio is not registered when `APP_ENV=production`.

## Native and Lite Runtime

Native Runtime executes the generated Go application and supports the complete framework.

Lite Runtime executes inside the installed `ctg` process for development environments where generated executables are blocked.

Lite Runtime provides:

- supported route execution
- resource/auth compatibility
- welcome page
- Studio inspectors
- migrations
- visual generators
- database tools
- health checks

Arbitrary project Go background code is intentionally Native-only.

## Frontend

The interactive project installer supports:

- TypeScript + Vite
- React + Vite
- Vue + Vite
- API-only

Frontend code communicates with CopyTyGo through the generated API client and local Vite proxy.

## Security

- Passwords: PBKDF2-HMAC-SHA256
- Reversible values: AES-256-GCM
- Sessions: AES-GCM encrypted cookies
- Authentication tokens: HMAC-SHA256 signed claims
- Public integer IDs: UUIDCos
- HTTP: security headers, body limits, CSRF/CORS helpers, configurable rate limiting
- SQL: identifier/operator validation
- Studio: development-only registration
- Production migrations: explicit destructive-operation safeguard
