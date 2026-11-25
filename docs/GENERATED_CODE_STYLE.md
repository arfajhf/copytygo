# Generated code style (v5.0.1 candidate)

CopyTyGo should be readable without requiring developers to read a tutorial for every file. The generator uses brief purpose comments, clear names and Go formatting. This is a presentation-only patch, not a change to runtime APIs.

## Comment placement

| File type | Expected use | Comment policy |
| --- | --- | --- |
| Controllers, services, application routes, frontend entry/views | Frequently customized | A short header above the file, with an optional single-line explanation where logic is not obvious |
| Models, middleware, jobs, listeners, seeders, factories, mail files | Customized as needed | One purpose sentence before `package` |
| Application bootstrap (`cmd/app/main.go`), generated registries | Rarely customized | Brief file-level explanation; leave generated-only markers intact |
| `.env`, `.env.example` | Frequently configured | One compact title header, blank line between configuration groups; no comments beside individual settings |

## Example Go file

```go
// ==========================================
// Application Controller
// Customize request handling and validation here.
// ==========================================

package controllers
```

## Example dotenv

```dotenv
# ==========================================
# CopyTyGo Application Configuration
# ==========================================

APP_ENV=local
APP_DEBUG=true

DB_DRIVER=postgres
DB_DATABASE=myapp
```

## Rules

- Use `//` in Go and TypeScript, `#` in dotenv and `/* ... */` in CSS. Non-standard comment syntax will not compile.
- Reserve `=====` for the header of major, user-edited files. Use `-----` separators only in large files when navigation truly benefits.
- Prefer a descriptive function name to a redundant inline comment.
- Add extra comments for surprising behavior, security assumptions or complex logic.
- Preserve required `copytygo:resource`, `copytygo:resource-access` and `copytygo:job` markers. These are read by the CLI.
- Never rewrite files customized by application developers during an update.
- Use `gofmt` for Go and the existing `.editorconfig` conventions for frontend files.

The v5.0.1 candidate applies this policy to generated starters. Existing user project files are left unchanged.

## CLI messages in the v5.0.1 candidate

Project creation reports the selected database, frontend, authentication mode, Studio status and the two next commands. It does not print generated secrets.

Database connection errors from `ctg db:check`, database-backed migrations and `ctg doctor --db` provide a short explanation and a configuration hint. Underlying driver errors are not copied into terminal output, since some database drivers include credentials or full connection strings in their messages. Test user-facing errors with incorrect hosts and invalid credentials.

Unknown CLI commands point to `ctg help`. Trying to create a project in an existing directory suggests choosing a different name or location, without modifying that directory.

These changes improve diagnostics and presentation; they do not add commands or change the framework's public APIs.
