# CopyTyGo v4 Development Runtime

CopyTyGo uses Native Runtime as the primary development runtime and Lite Runtime as a Windows-friendly fallback.

## Native-first execution

```text
ctg dev
  -> go run ./cmd/app
       -> success: Native Runtime
       -> supported Windows Application Control block:
            -> remember the decision in the local user cache
            -> automatic Lite Runtime fallback
  -> remembered policy block on later runs: Lite Runtime directly
```

The user-facing command remains `ctg dev`. Installing auth or editing source does not reset a remembered Windows policy block. Runtime decisions are stored per project in the user cache, outside the project source.

Retry Native Runtime after a policy change:

```powershell
ctg dev --native
```

Force Lite for one run:

```powershell
ctg dev --lite
```

## Native Runtime

Native Runtime executes the generated application and therefore supports the complete framework:

- arbitrary controllers and services
- middleware
- database resources
- queue workers
- durable database queue
- scheduler and cron
- events/listeners
- mail
- storage
- service container
- Studio runtime inspectors
- custom project packages

Local startup output includes:

```text
Application  : http://127.0.0.1:8080
Studio       : http://127.0.0.1:8080/__copytygo
Documentation: ...
```

## Lite Runtime

Lite Runtime executes inside the installed `ctg` process and does not require a generated temporary application executable.

Supported development features include:

- GET / POST / PUT / PATCH / DELETE
- route params and query params
- request input/body
- validation
- supported controller handlers
- generated database resources
- pagination/search/filter/sort
- soft deletes
- generated authentication routes
- MySQL/PostgreSQL access
- migration management
- route hot reload
- automatic free-port fallback
- CopyTyGo welcome page
- CopyTyGo Studio

Lite Studio includes inspection for:

- routes
- models
- resources
- database
- migrations
- authentication
- configured services
- health
- doctor

It also supports code generation.

## Native-only behavior

Lite Runtime is not a general Go interpreter.

Arbitrary project Go code such as custom queue job handlers and scheduler callbacks is not executed inside Lite Runtime.

Studio explicitly marks Queue and Scheduler execution as Native-only in this mode.

## Windows Application Control

CopyTyGo does not require users to disable Windows security controls.

If the operating system blocks the temporary executable produced by `go run`, `ctg dev` detects supported policy errors and falls back to Lite Runtime.

Production does not rely on this fallback.

## Browser helpers

```powershell
ctg open
ctg studio
```

These open the current application or Studio using the configured host/port.

## Production

Production uses a native binary:

```powershell
ctg build
```

The generated server includes read-header, read, write, idle, and shutdown timeouts plus graceful shutdown.

Before deployment:

```powershell
ctg doctor --db --production
```
