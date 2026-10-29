# Generated resource security (CopyTyGo v5 release candidate)

CopyTyGo v5 treats generated REST resources as **private by default**. Earlier generated routes were public until the developer added middleware. This change is intentional and can break old applications that relied on public CRUD endpoints.

## New resources

```bash
ctg make:resource Product name:string price:decimal stock:integer active:boolean
```

The generated controller includes `// copytygo:resource-access default`. The generated `routes/resources.go` enforces the following policy:

| Starter | Default GET, POST, PUT, DELETE |
| --- | --- |
| Multi-role browser auth (TypeScript + Go) | Signed-in **admin** only; all writes also require session CSRF token |
| Single-role browser auth | Signed-in user; writes also require session CSRF token |
| Multi-role API-only auth | Bearer token with **admin** role |
| Single-role API-only auth | Valid bearer token |
| No auth starter | HTTP 401, until auth is installed and resource routes are resynced |

The default deny rule applies to both collection routes and member routes. Unauthenticated users cannot read resources accidentally. A token/cookie does not bypass role checks.

## Explicit access options

```bash
# Explicitly public resource, including write methods. Use only intentionally.
ctg make:resource PublicNote body:text --public

# Browser auth uses session, API-only uses bearer token.
ctg make:resource Invoice amount:decimal --access=authenticated

# Multi-role auth; administrators only.
ctg make:resource AccountSettings label:string --access=admin

# Multi-role auth; only users with role user, not admins.
ctg make:resource MyTask title:string --access=user
```

`--access=default` is implicit. An explicit policy is saved in a `// copytygo:resource-access ...` comment in the generated controller. Flags are parsed before field definitions, so flags can appear anywhere after the resource name. Invalid or repeated access flags are rejected.

**Warning:** `--public` makes GET **and all writes** public. This is suitable only for resources intentionally exposed without login. Public write routes need separate rate limits, spam controls, input validation and authorization rules appropriate to their data. Never use `--public` to resolve a 401 without reviewing the data sensitivity.

Only the v5 generator is protected by default. Calls that developers explicitly write with `app.Resource(...)` keep the existing public semantics for compatibility. Use `app.ResourceWithMiddleware(...)` to apply your own security.

## Upgrade existing generated routes

Older generated resources lack the new policy marker. They are interpreted as `default` when regenerated and receive the current auth-starter policy.

1. Commit your project and review any custom route changes.
2. Build the newest v5 candidate CLI from the branch in the framework repository.
3. Inside your application, run `ctg resource:sync`.
4. Inspect `routes/resources.go`. It is **generated and overwritten**. Do not add custom route handlers there.
5. Rebuild and verify unauthenticated requests get 401, insufficient roles get 403, and CSRF-invalid browser writes get 403.
6. Explicitly public resources may opt into `// copytygo:resource-access public` in their controller, then rerun `ctg resource:sync`.

`ctg resource:sync` does not overwrite your controllers, database rows or migrations. It recreates the route registry from resource markers. If authentication is installed after resources are created, rerun `ctg resource:sync` to activate the appropriate guards.

## Browser authentication and CSRF

For generated resources with a browser auth frontend, `GET /api/session` supplies the CSRF token and session cookie. Browser JavaScript must use `credentials: "same-origin"` and send the `X-CSRF-Token` header on POST/PUT/DELETE. The cookie is HttpOnly and cannot be read by JavaScript. Users without the required role are refused by the server; UI visibility is not authorization.

**Example:**

```ts
const sessionResponse = await fetch("/api/session", { credentials: "same-origin" });
const session = await sessionResponse.json();
const response = await fetch("/products", {
  method: "POST",
  credentials: "same-origin",
  headers: {
    "Content-Type": "application/json",
    "X-CSRF-Token": session.csrf,
  },
  body: JSON.stringify({ name: "Example", price: 100, stock: 1, active: true }),
});
```

The default browser-generated resource does **not** accept bearer-token requests as an alternative to the cookie session. For external/mobile clients, use an API-only project with bearer auth or register a separate custom API route with `copyauth.Middleware()` and `copyauth.RequireRole(...)`.

## Custom resource middleware

`ResourceWithMiddleware` takes a list of middleware for every route, then an additional list for write routes. For example:

```go
app.ResourceWithMiddleware(
    "/orders",
    controllers.OrderController{},
    []core.Middleware{copyauth.Middleware(), copyauth.RequireCurrentRole(true, "admin")},
    nil,
)
```

Use this helper when custom policies are necessary. Authentication and authorization must be enforced on the server.

## Validation limits

Security regression tests cover route middleware dispatch and code generation for browser single/multi, bearer, missing auth and explicit public modes. Release testing must still verify real browser sessions, role changes, CSRF, starter compilation and deployment. This is not a completed external penetration test.
