# Upgrading CopyTyGo v4 to v5

> **Release-candidate instructions.** v5 is not a published stable Go module until the final tag exists. Do not run the v5 commands below against production yet.

## Breaking change: Go semantic import versioning

Go major versions 2+ require their major version in module imports. For v5, use:

```text
github.com/arfajhf/copytygo/v5
```

Existing applications that still require `github.com/arfajhf/copytygo/v4` can remain on the supported v4 release. There is no automatic migration of a project's Go source or database schema.

When v5 is published, **in a separate upgrade branch**:

1. Commit your application and back up the database.
2. Replace framework import references `github.com/arfajhf/copytygo/v4` with `github.com/arfajhf/copytygo/v5` in the application's Go source and in generated code, after reviewing differences.
3. Update the module dependency and install the new CLI using the exact final release tag rather than an unpinned `latest`.
4. Run `go mod tidy`, `go test ./...`, `ctg doctor`, and `ctg build` in your application.
5. Verify migrations, session/login behavior, admin permissions, APIs, frontend assets, queue workers and deployment before rolling out.

## Breaking change: generated resource routes are protected

Earlier `make:resource` routes could be publicly readable/writable. In v5, newly generated resources are **private by default** and middleware covers read and write requests. Multi-role projects default to admin, single-role projects require a signed-in account, and resource routes without auth now reject requests with HTTP 401.

After updating an existing project to v5, run:

```bash
ctg resource:sync
```

This rebuilds `routes/resources.go`, including security on controllers created with older versions. Check each endpoint before deployment. An endpoint that intentionally needs public CRUD must explicitly use `// copytygo:resource-access public` in the generated controller before syncing; public writes require careful abuse controls. Browser-authenticated writes require a valid `X-CSRF-Token` header. See [resource security guide](RESOURCE_SECURITY.md).

## Changes to review

### Windows native development

`ctg dev` on Windows now builds and runs the project-local `build/dev/app.exe`. This is **not** a bypass for Windows security policy. The new [Windows development guide](WINDOWS_DEVELOPMENT.md) explains Defender, Smart App Control and App Control for Business diagnostics.

### Authentication UI

Generated browser authentication UI is TypeScript under `frontend/src/auth/`. Go implements database-backed session APIs and access control. Earlier projects using unchanged HTML starters may use `ctg install:auth` to adopt the current starter, but projects with customized views or routes require a manual, reviewed migration. Never overwrite your custom auth files or existing migrations without review.

### Frontend and public assets

Keep publicly accessible images and fonts in `frontend/public/` and link them by root-relative URLs such as `/images/banner.png`. Deploy the full `build/` directory, not just the native executable.

### Production

Set `APP_ENV=production`, `APP_DEBUG=false`, and configure a unique strong `APP_KEY`. CopyTyGo Studio must respond with HTTP 404 in production. Complete production smoke testing using [production checklist](PRODUCTION.md).

## Release validation

The v5 major version requires a successful fresh-project smoke test from the **published immutable tag**. Local `replace` directives do not count as proof that external users can install the module. This is a tracked gate on the [v5 launch checklist](V5_LAUNCH_CHECKLIST.md).
