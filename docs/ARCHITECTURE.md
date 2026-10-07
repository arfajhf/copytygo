# CopyTyGo v3 architecture

## Request flow

```text
Client
  -> RequestID middleware
  -> Request logger
  -> Security middleware
  -> Route middleware
  -> Handler
  -> Context response
```

`core.Application` implements `http.Handler`. The built-in production server adds read/write/idle timeouts and graceful shutdown.

## Resource flow

```text
ctg make:resource
  -> typed model
  -> database-backed controller
  -> migration
  -> timestamps
  -> soft-delete column
  -> generated REST route registry
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

Generated list endpoints support pagination, search, filtering and sorting through generated allowlists.

Generated resources use `deleted_at`; normal list/show/update queries exclude soft-deleted rows.

## Database flow

The shared database connection is initialized safely for concurrent requests. Query builder table/column identifiers and operators are validated before SQL execution.

Migrations use:

```text
project migration source
  -> migration source parser
  -> registry
  -> schema blueprint
  -> MySQL/PostgreSQL dialect
  -> SQL
  -> migration repository
```

Destructive migration commands require `--force` in production.

## Authentication flow

```text
register/login
  -> database user lookup/write
  -> PBKDF2 password hashing
  -> HMAC signed token
  -> bearer middleware
  -> claims on Context
  -> optional role guard
```

Signed tokens require a strong application key and validate subject, issued-at and expiry.

## Development runtime

Native Go is primary. Lite Runtime remains a development fallback for environments that block generated executables.

Lite Runtime v3 supports generated DB resources, list options, soft deletes, validation and generated auth routes.

## Frontend

Generated projects use TypeScript + Vite and include a reusable API client with bearer-token support and an `/api` development proxy.

## Production observability

Generated applications use:

- `X-Request-ID` request correlation.
- structured request logs.
- JSON framework errors.
- health responses with framework version and environment.
- graceful shutdown on interrupt/SIGTERM.

## Security model

- Passwords: PBKDF2-HMAC-SHA256.
- Reversible application values: AES-256-GCM.
- Auth tokens: HMAC-SHA256 signed claims.
- Public integer IDs: UUIDCos masking.
- HTTP protections: security headers, body limits, CSRF, CORS and rate limiting helpers.
- Query builder: identifier/operator allowlisting.
- Production diagnostics: `ctg doctor --db --production`.
