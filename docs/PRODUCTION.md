# CopyTyGo v4 production checklist

CopyTyGo v4 expands the v3 production foundation with application services, durable queues, runtime health, encrypted sessions, and a development-only Studio.

## 1. Environment

Recommended baseline:

```env
APP_ENV=production
APP_DEBUG=false
COPYTYGO_STUDIO=false
APP_KEY=<strong generated key>

SERVER_READ_HEADER_TIMEOUT=5
SERVER_READ_TIMEOUT=15
SERVER_WRITE_TIMEOUT=30
SERVER_IDLE_TIMEOUT=60
SERVER_SHUTDOWN_TIMEOUT=10
```

Generate an application key with:

```powershell
ctg key:generate
```

Do not commit production `.env` files.

## 2. Database

Configure either MySQL or PostgreSQL:

```env
DB_DRIVER=mysql
DB_HOST=127.0.0.1
DB_PORT=3306
DB_DATABASE=app
DB_USERNAME=app
DB_PASSWORD=<secret>
```

Verify before deployment:

```powershell
ctg db:check
ctg migrate:status
ctg migrate
```

Destructive production migration commands require:

```powershell
ctg migrate:rollback --force
ctg migrate:reset --force
ctg migrate:fresh --force
```

## 3. Queue

For production workloads that must survive restarts, use:

```env
QUEUE_DRIVER=database
QUEUE_WORKERS=2
QUEUE_POLL_SECONDS=1
QUEUE_BACKOFF_SECONDS=1
```

The database queue maintains CopyTyGo jobs and failed-job tables in the configured database.

Use `sync` only when immediate execution is explicitly desired.

Use `memory` for development or workloads where in-memory durability is acceptable.

## 4. Mail and storage

Example:

```env
STORAGE_PATH=storage/app

MAIL_HOST=smtp.example.com
MAIL_PORT=587
MAIL_USERNAME=<username>
MAIL_PASSWORD=<secret>
MAIL_FROM=noreply@example.com
```

Restrict filesystem permissions for the configured storage path.

Use real production SMTP credentials and never expose `MAIL_PASSWORD` through application responses.

## 5. Sessions and authentication

`APP_KEY` must contain at least 32 characters.

CopyTyGo uses:

- PBKDF2 for password hashes
- HMAC-signed authentication claims
- AES-GCM for encrypted session cookies
- AES-GCM for reversible application values

Session cookies are marked Secure automatically in production.

## 6. Studio

Studio is a development surface and is not registered in production.

Expected behavior:

```text
GET /__copytygo
-> 404
```

Do not expose a development Studio through reverse-proxy exceptions.

## 7. Health

Generated applications expose:

```text
GET /api/health
```

Default health checks include:

- database
- cache
- storage

The endpoint returns HTTP `503` when the overall health state is unhealthy.

Use this endpoint for load balancer or uptime checks when appropriate.

## 8. Logs and request IDs

Responses can include:

```text
X-Request-ID
```

Structured request logs include:

- request ID
- method
- path
- status
- latency
- remote address

Route application logs through the production log collection system used by your infrastructure.

## 9. Server lifecycle

Generated applications include configurable HTTP timeouts and graceful shutdown.

On SIGINT/SIGTERM:

- background context is cancelled
- queue/scheduler workers stop
- the HTTP server shuts down within the configured timeout

## 10. Framework diagnostics

Run:

```powershell
ctg doctor --db --production
```

All required production checks should pass.

## 11. Build

Build the native application:

```powershell
ctg build
```

Windows output:

```text
build\app.exe
```

Linux/macOS output:

```text
build/app
```

Lite Runtime is development-only.

## 12. Reverse proxy

Place the application behind a trusted reverse proxy/load balancer where appropriate.

Terminate TLS through trusted infrastructure and forward only the proxy headers your deployment explicitly trusts.

## 13. Final verification

Recommended sequence:

```powershell
ctg version
ctg doctor --db --production
ctg route:list
ctg migrate:status
ctg build
```

Then verify:

- native binary starts
- `/api/health` is healthy
- `/__copytygo` returns 404
- database queue processes a named job when enabled
- graceful shutdown completes
