# Changelog

## v4.0.1

- Made plain `ctg dev` remember Windows policy fallback per project in the local user cache.
- Removed raw native execution-policy errors when automatic Lite startup is available.
- Added `ctg dev --native` to retry native execution after a remembered fallback.
- Preserved compilation and source-file permission errors rather than hiding them behind Lite.
- Added regression tests and a Windows process smoke test covering policy fallback followed by `install:auth`.

## v4.0.0

- Migrated the framework module to `github.com/arfajhf/copytygo/v4`.
- Added the CopyTyGo welcome page with Documentation, Studio and GitHub links.
- Added CopyTyGo Studio as a production-disabled local development workspace.
- Added Studio dashboards for routes, models, resources, database, migrations, authentication, services, queue, scheduler, requests, errors, logs, runtime health and Doctor diagnostics.
- Added a visual generator that uses the same generator engine as the CLI.
- Added Studio and welcome-page parity to Lite Runtime for supported Windows Application Control fallback scenarios.
- Added interactive project creation with MySQL/PostgreSQL, auth mode, Studio, and TypeScript/React/Vue/API-only starter choices.
- Added `ctg open` and `ctg studio`.
- Added cache abstraction with TTL memory storage and `Remember`.
- Added events/listeners and event registration inspection.
- Added SMTP mail, mail fakes and HTML/text mail templates.
- Added storage abstraction, local disk protection and uploaded-file helpers.
- Added service container bindings, singletons, instances and default application services.
- Added in-memory, synchronous and durable MySQL/PostgreSQL queue drivers.
- Added named job registry, delayed jobs, retries/backoff, failed jobs and configurable queue workers.
- Added scheduler runtime status, interval helpers and five-field cron expressions.
- Added application background-service lifecycle with graceful shutdown.
- Added database transaction helpers and model HasOne/HasMany/BelongsTo relationships.
- Added seeders, generic factories and API resource serialization.
- Added encrypted AES-GCM sessions with flash data and request-scoped mutation consistency.
- Added policy-based authorization gates.
- Finalized configurable rate limiting with remaining/reset/retry headers and bucket cleanup.
- Added nested route groups, route-name prefixes, grouped resources and API version helpers.
- Added a reusable retry/backoff package and safe HTTP-client retries for idempotent requests.
- Added runtime health registry with default database/cache/storage checks and generated `/api/health` responses.
- Added structured framework logging, request inspection and error inspection.
- Added HTTP testing helpers and framework fakes.
- Added CLI and Studio generators for jobs, listeners, seeders, factories and mail classes.
- Added production documentation and v4 release-gate tracking.
- Completed Native/Lite visual scaffold generator parity.
- Gated final releases on successful CI and verified installation from the published tag.

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

## v2.0.0

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
- Added typed resource fields to `ctg make:resource`, including nullable `?` syntax.
- Added ready-to-use auth HTTP routes for register, login and current-user lookup.
- Added Lite Runtime support for generated auth routes.
- Added typed auth errors and single-role/multi-role schema alignment.
- Added `core.Application` support for the standard `http.Handler` interface.
- Added automatic model connection with `model.Auto(...)`.
- Hardened the query builder with identifier/operator validation and empty-write protection.
- Made shared database connection initialization safe for concurrent requests.
- Hid Lite Runtime database internals unless `APP_DEBUG=true`.
- Migrated the v2 Go module path to `github.com/arfajhf/copytygo/v2`.
- Added v2 unit coverage for resource routing, resource fields, auth middleware/roles, query safety and schema timestamps.
- Made installed `ctg migrate` load migrations from the current project instead of the framework example package.
- Added source-based project migration discovery for the CopyTyGo schema DSL.
- Made generated migration function IDs collision-resistant while keeping legacy registry compatibility.

## v3.0.0

- Added paginated database resource listings with page/per_page metadata.
- Added safe resource search across generated string/text fields.
- Added generated filter and sort allowlists for resource endpoints.
- Added query parameters for q, sort, order and filter[field].
- Added numeric, boolean and UUID validation rules.
- Added type-aware resource validation for decimal, boolean and UUID fields.
- Added Lite Runtime support for v3 resource list options.
- Added tests for extended validation and multi-column search.
- Added production HTTP server timeouts and graceful shutdown.
- Added consistent JSON 404 and 405 responses.
- Added request IDs and structured request logging.
- Added production diagnostics through `ctg doctor --production`.
- Added automatic `updated_at` values for resource updates.
- Added generated soft deletes with `deleted_at`.
- Added Lite Runtime parsing/execution support for generated soft-delete resources.
- Added soft-delete support to the project migration source loader.
- Hardened signed auth tokens with minimum secret length, subject checks, issued-at and expiration validation.
- Protected destructive migration commands in production behind `--force`.
- Added generated production server timeout settings and richer health responses.
- Added production-focused tests for JSON errors, request IDs and signed tokens.

