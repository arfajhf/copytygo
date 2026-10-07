# CopyTyGo v4 Roadmap

CopyTyGo v4 is the feature-complete framework milestone. It combines a mature backend ecosystem with first-class developer experience.

## Developer Experience

- [x] Default welcome page
- [x] CopyTyGo Studio foundation
- [x] Route explorer
- [x] Resource browser and database preview
- [x] Database connection explorer
- [x] Migration manager
- [x] Request inspector
- [x] Error inspector
- [x] Structured log viewer
- [x] Visual resource and ecosystem generators
- [x] Doctor dashboard
- [x] Runtime Health dashboard
- [x] Queue dashboard
- [x] Scheduler dashboard
- [x] Application services dashboard
- [x] Interactive project installer
- [x] TypeScript, React, Vue and API-only starters
- [x] Documentation integration
- [x] Production-safe Studio controls
- [x] Lite Runtime Studio parity for Windows execution-policy fallback
- [ ] Dedicated model/auth inspector
- [ ] Final Studio navigation/UI polish

## Backend Ecosystem

- [x] Memory queue and background workers
- [x] Durable MySQL/PostgreSQL database queue
- [x] Named job registry and delayed jobs
- [x] Scheduler intervals and five-field cron expressions
- [x] Cache abstraction and TTL memory store
- [x] SMTP mail and mail templates
- [x] Storage abstraction, local disk and uploads
- [x] Events and listeners
- [x] Database transaction helper
- [x] Model relationships
- [x] Seeders and factories
- [x] API resources and serialization
- [x] Encrypted sessions, cookies and flash data
- [x] Authorization and policy gate
- [x] Configurable rate limiting
- [x] Service container and default providers
- [x] Testing helpers and framework fakes
- [x] Reusable retry/backoff
- [x] Runtime health registry
- [x] Structured observability
- [x] Nested route groups and API versioning
- [x] HTTP client retry safety
- [ ] Final database-queue end-to-end smoke test on MySQL and PostgreSQL
- [ ] Final Windows Native/Lite application smoke test

## v4 Release Gate

v4 becomes RC only when:

1. CI is green for tests, race tests, vet, all-package build, Linux CLI and Windows CLI.
2. A fresh project can be created with the interactive installer.
3. Welcome page and Studio work in Native Runtime and Lite Runtime.
4. MySQL and PostgreSQL migrations and durable queue are smoke-tested.
5. Studio production routes return 404.
6. Generated TypeScript, React, Vue and API-only project presets are validated.
7. Windows smoke testing passes on a fresh project.

v4 should already be usable for real application development. v5 is reserved for stabilization, official documentation/examples, distribution polish, security/performance review and public launch.
