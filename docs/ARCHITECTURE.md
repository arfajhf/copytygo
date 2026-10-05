# CopyTyGo architecture

## Request flow

Client -> Router -> Global middleware -> Route middleware -> Handler -> Context response.

## Database flow

Application -> database connection -> query/model layer -> MySQL or PostgreSQL.

Migrations use: migration file -> generated registry -> blueprint -> dialect -> SQL -> migration repository.

## Frontend flow

The generated project contains `frontend/` with TypeScript and Vite. The frontend talks to Go through JSON APIs. CopyTyGo intentionally does not invent a Blade-like language.

## Security model

Passwords use PBKDF2-HMAC-SHA256. Reversible values use AES-256-GCM. Auth tokens are HMAC signed. Public integer IDs can be encrypted with `UUIDCos`. HTTP middleware provides CSRF, CORS, security headers, body limits and rate limiting.

## Auth

`ctg install:auth single` generates the base user and auth middleware. `multi` also adds a role field. Projects can extend this scaffold with database-backed login handlers and permission rules.
