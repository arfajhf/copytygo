# CopyTyGo v4 Development Runtime

CopyTyGo runs Native Runtime by default. Lite Runtime requires explicit opt-in.

## Native execution

```text
ctg dev
  -> go run ./cmd/app
       -> success: Native Runtime
       -> failure: return the Native error

ctg dev --lite
  -> Lite Runtime
```

Every `ctg dev` startup runs the complete generated application, including after installing auth or editing source. Cached Lite decisions written by v4.0.1 are ignored. No cache cleanup is required when updating the CLI.

The explicit Native flag remains a compatible alias:

```powershell
ctg dev --native
```

Choose Lite for one run:

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

The browser authentication starter (HTML forms, encrypted web session, account and logout pages) requires Native Runtime. Lite continues to support the generated bearer-token authentication API.

Lite Runtime is not a general Go interpreter.

Arbitrary project Go code such as custom queue job handlers and scheduler callbacks is not executed inside Lite Runtime.

Studio explicitly marks Queue and Scheduler execution as Native-only in this mode.

## Windows Application Control

If Windows blocks the temporary executable produced by `go run`, `ctg dev` returns the execution error. The CLI does not override operating-system policy or switch to Lite automatically. Resolving an actual policy refusal requires identifying the applicable Windows policy and its trust requirements.

An earlier policy refusal does not prevent later Native startup when Windows permits execution.

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
