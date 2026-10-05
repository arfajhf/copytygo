# Changelog

## v1.0.0

- Added HTTP core, routing, middleware and request binding.
- Added MySQL/PostgreSQL database layer, schema dialects and migration engine.
- Added query builder and model base.
- Added migration, project and code generators.
- Added TypeScript frontend starter and API client.
- Added validation, HTTP client and security utilities.
- Added PBKDF2 password hashing, AES-GCM encryption, signed tokens and UUIDCos public IDs.
- Added single-role/multi-role auth scaffold.
- Added security middleware, migration reset/fresh and route listing.

## v1.1.0-dev

- Reworked `ctg dev` into a runtime orchestrator.
- Added automatic Windows Application Control detection and Lite Runtime fallback.
- Added `ctg dev --lite` to force the lightweight in-process development runtime.
- Lite Runtime is dependency-free and currently supports simple inline `ctx.JSON(core.Map{...})` and `ctx.Text(...)` routes.
- Native runtime remains the default for full Go compatibility.
- Docker, WSL, and VM software are not required by Lite Runtime.
