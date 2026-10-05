# CopyTyGo Development Runtime

`ctg dev` prefers the native Go runtime because it provides complete Go compatibility.

On Windows systems where Application Control blocks the temporary application executable, CopyTyGo detects the policy error and automatically falls back to Lite Runtime.

```text
ctg dev
  -> native Go runtime
       -> success: application runs normally
       -> Windows execution policy block: Lite Runtime
```

You can test Lite Runtime directly with:

```powershell
ctg dev --lite
```

## Lite Runtime v1.1 scope

Lite Runtime runs inside the already-installed `ctg` process and does not create a project executable. It currently discovers route declarations from `routes/*.go` and supports simple inline handlers returning:

```go
ctx.Text("hello")
```

or static scalar JSON maps:

```go
ctx.JSON(core.Map{"framework": "CopyTyGo", "status": "ok"})
```

Controllers, arbitrary Go packages, dynamic expressions, database calls, middleware execution and other general Go behavior require Native Runtime. CopyTyGo intentionally reports unsupported Lite behavior instead of pretending it is equivalent to compiled Go.

Lite Runtime is a development fallback, not a production runtime. `ctg build` continues to produce a normal native Go application.


### Current v1.1 development support

Lite Runtime now understands:

- GET, POST, PUT, PATCH and DELETE inline routes
- route parameters using CopyTyGo syntax such as `/users/:id`
- `ctx.Param("id")` inside Text/JSON responses
- `ctx.Query("q")` inside Text/JSON responses
- chained status responses such as `ctx.Status(201).JSON(...)`
- automatic free-port selection when the configured port is busy
- route hot reload without restarting `ctg dev --lite`

Example:

```go
app.Get("/users/:id", func(ctx *core.Context) error {
    return ctx.JSON(core.Map{
        "id": ctx.Param("id"),
        "q":  ctx.Query("q"),
    })
})
```

Controllers, arbitrary application logic, middleware execution, request binding/validation, database calls and general Go expressions still require Native Runtime while Lite support is expanded.
