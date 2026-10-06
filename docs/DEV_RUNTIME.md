# CopyTyGo Development Runtime

CopyTyGo v2 keeps native Go as the primary development and production runtime.

```text
ctg dev
  -> native Go runtime
       -> success: run the application normally
       -> supported Windows Application Control block: Lite Runtime
```

Force Lite Runtime with:

```powershell
ctg dev --lite
```

## Lite Runtime v2

Lite Runtime executes inside the installed `ctg` process so Windows does not need to execute a newly generated application binary.

Supported development flows include:

- GET, POST, PUT, PATCH and DELETE routes
- route parameters and query parameters
- `ctx.Body()` and `ctx.Input()`
- declarative validation
- controller methods
- `app.Resource(...)`
- generated resource CRUD
- MySQL/PostgreSQL resource CRUD
- generated auth register/login/me routes
- HTTP status responses
- route hot reload
- automatic free-port fallback

Generated database resources can therefore keep working when native development execution is blocked.

## Native-only Go behavior

Lite Runtime is intentionally not a general Go interpreter. Arbitrary application expressions, custom package execution, complex service orchestration and unsupported middleware behavior still belong to Native Runtime.

Production continues to use:

```powershell
ctg build
```

which produces a normal Go application.

## Diagnostics

```powershell
ctg doctor
ctg doctor --db
```

Use `--db` when the project is configured with a live MySQL or PostgreSQL database.
