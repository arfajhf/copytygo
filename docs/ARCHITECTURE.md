# CopyTyGo v2 architecture

## Request flow

Client -> Router -> Global middleware -> Route middleware -> Handler -> Context response.

`core.Application` implements `http.Handler`, so the framework can run through its built-in server, `httptest`, or another Go HTTP server.

## Resource flow

Generated v2 resources use:

```text
ctg make:resource
  -> model
  -> database-backed controller
  -> migration
  -> generated route registry
```

At runtime:

```text
REST route
  -> controller
  -> Context DB helper
  -> database.Resource
  -> query.Builder
  -> MySQL/PostgreSQL
```

`app.Resource(...)` expands to the standard index/show/store/update/destroy REST routes.

## Database flow

Application -> shared database connection -> resource/model/query layer -> MySQL or PostgreSQL.

The shared connection is initialized safely for concurrent requests. Query builder identifiers and operators are validated before execution.

Migrations use:

```text
migration file
  -> generated registry
  -> schema blueprint
  -> database dialect
  -> SQL
  -> migration repository
```

## Authentication flow

```text
register/login
  -> database user lookup/write
  -> PBKDF2 password hashing
  -> signed token
  -> bearer middleware
  -> claims stored on Context
  -> optional role guard
```

`ctg install:auth single` and `multi` generate the users migration and register HTTP auth routes.

## Development runtime

Native Go is the primary runtime. Lite Runtime is a development fallback for environments that block generated executables.

Lite Runtime v2 can execute generated database resources and generated auth routes directly inside the `ctg` process.

## Frontend flow

Generated projects contain `frontend/` with TypeScript and Vite. The generated API client supports JSON requests and bearer tokens, while Vite proxies `/api` requests to the Go development server.

CopyTyGo intentionally does not introduce a Blade-like template language.

## Security model

- Passwords: PBKDF2-HMAC-SHA256.
- Reversible application values: AES-256-GCM.
- Auth tokens: HMAC signed.
- Public integer IDs: UUIDCos masking.
- HTTP protections: CSRF, CORS, security headers, body limits and rate limiting.
- Query builder: identifier/operator validation.
- Database errors are hidden in Lite Runtime unless `APP_DEBUG=true`.
