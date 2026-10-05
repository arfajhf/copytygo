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
