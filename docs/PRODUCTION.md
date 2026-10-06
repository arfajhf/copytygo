# CopyTyGo v3 production checklist

CopyTyGo v3 is the first production-oriented major release line.

## 1. Environment

Use:

```env
APP_ENV=production
APP_DEBUG=false
APP_KEY=<strong generated key>

SERVER_READ_HEADER_TIMEOUT=5
SERVER_READ_TIMEOUT=15
SERVER_WRITE_TIMEOUT=30
SERVER_IDLE_TIMEOUT=60
SERVER_SHUTDOWN_TIMEOUT=10
```

Generate a key with:

```powershell
ctg key:generate
```

Do not commit the production `.env` file.

## 2. Database

Configure MySQL or PostgreSQL credentials, then verify:

```powershell
ctg db:check
ctg migrate:status
ctg migrate
```

Destructive commands require explicit confirmation in production:

```powershell
ctg migrate:rollback --force
ctg migrate:reset --force
ctg migrate:fresh --force
```

## 3. Framework diagnostics

Before deploying:

```powershell
ctg doctor --db --production
```

All checks should pass.

## 4. Build

Build a native Go application:

```powershell
ctg build
```

Lite Runtime is a development fallback and is not the production runtime.

## 5. Health and logs

Generated projects expose:

```text
GET /api/health
```

Responses include framework version, environment, status and request ID.

Every request can receive an `X-Request-ID` response header. Generated applications also use structured request logs with method, path, status, duration and request ID.

## 6. Reverse proxy

Place the native application behind a production reverse proxy or load balancer when appropriate. Terminate TLS at a trusted proxy or the application infrastructure and forward the original request information according to your deployment environment.

## 7. Authentication

Use a strong `APP_KEY` of at least 32 characters. CopyTyGo signed tokens validate signature, subject, issued-at and expiration.

Passwords are stored using one-way PBKDF2 hashing.

## 8. Resource behavior

Generated v3 resources include:

- pagination
- search
- filter allowlists
- sort allowlists
- validation
- timestamps
- soft deletes

DELETE requests on generated resources set `deleted_at`; normal generated queries exclude deleted rows.

## 9. Release verification

Recommended final commands:

```powershell
ctg version
ctg doctor --db --production
ctg route:list
ctg migrate:status
ctg build
```

Then run the built application in the target environment and verify `/api/health`.
