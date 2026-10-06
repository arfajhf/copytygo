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

## v1.1.0

- Reworked `ctg dev` into a runtime orchestrator with native-first execution.
- Added automatic Windows Application Control detection and Lite Runtime fallback.
- Added `ctg dev --lite` for direct lightweight development.
- Added route hot reload and automatic free-port fallback.
- Added Lite support for GET, POST, PUT, PATCH and DELETE.
- Added route params, query params, raw body access and `ctx.Input()`.
- Added declarative validation with structured 422 responses.
- Added controller method resolution in Lite Runtime.
- Added `ctg make:controller <Name> --resource`.
- Added in-memory resource CRUD for Native and Lite Runtime.
- Added centralized version reporting across CLI and application banners.
- Added `ctg doctor` for one-command project and Lite Runtime health checks.
- Added CI configuration for test, vet and CLI build checks.
- Docker, WSL and VM software remain optional and are not required by Lite Runtime.

## v2.0.0-dev

- Started the database-first resource layer for MySQL and PostgreSQL.
- Added map-based query/model CRUD helpers and portable insert ID handling.
- Added database-backed HTTP resource helpers: `DBIndex`, `DBShow`, `DBStore`, `DBUpdate`, and `DBDestroy`.
- Added database resource execution inside Lite Runtime.
- Added one-line REST resource routing with `app.Resource(...)`.
- Added `ctg make:resource <Name>` for model + DB controller + migration generation.
- Kept explicit `--memory-resource` mode for temporary development CRUD.
- Added `ctg db:check` for database health diagnostics.
- Added reusable auth middleware, claims context, user ID helpers, and role guards.
- Began v2 context value support for middleware and application state.
- Added `ctg update [version]` to update both the CLI and project dependency.
- Added generated resource route registry so `ctg make:resource` auto-registers REST resources.
- Added database-backed auth registration/login services and auth user migration generation.
- Upgraded the TypeScript starter with a reusable API client, bearer-token support, and Vite API proxying.
- Expanded `ctg route:list` to show routes created by `app.Resource(...)`.
- Expanded `ctg doctor` with resource-route, migration-registry, security, and optional live database checks via `--db`.

