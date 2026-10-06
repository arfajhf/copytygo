# CopyTyGo v3 Development Runtime

CopyTyGo uses native Go as the primary runtime.

```text
ctg dev
  -> native Go
       -> success: normal development server
       -> supported Windows Application Control block: Lite Runtime
```

Force Lite Runtime:

```powershell
ctg dev --lite
```

## Lite Runtime v3

Lite Runtime executes inside the installed `ctg` process.

Supported generated flows include:

- GET, POST, PUT, PATCH and DELETE
- route params/query params
- `ctx.Body()` and `ctx.Input()`
- validation
- controller methods
- resource CRUD
- pagination/search/filter/sort
- soft-delete resource options
- MySQL/PostgreSQL resources
- auth register/login/me
- hot route reload
- automatic free-port fallback

Lite Runtime is not a general Go interpreter. Arbitrary service/package logic still belongs to Native Runtime.

## Production

Production uses a normal Go executable:

```powershell
ctg build
```

Generated servers include timeouts and graceful shutdown.

Before launch:

```powershell
ctg doctor --db --production
```

For production, use `APP_ENV=production`, `APP_DEBUG=false`, a generated strong `APP_KEY`, and real database credentials.
