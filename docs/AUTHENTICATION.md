# Authentication starter

## Install and run

Choose one mode:

```powershell
ctg install:auth single
```

Or use multiple roles:

```powershell
ctg install:auth multi
```

Configure your MySQL or PostgreSQL connection in `.env`, then create the table and start Native development:

```powershell
ctg migrate
ctg dev
```

Open the application welcome page. **Log in** and **Register** appear after the browser auth routes are installed. Register a user, and the application signs them in and redirects to `/account`.

## Files to customize

| File | Purpose |
| --- | --- |
| `routes/auth.go` | Explicit browser and API routes and their middleware |
| `app/auth/web.go` | Default public registration role and embedded HTML views |
| `app/auth/views/layout.html` | Shared responsive layout, CSS, password visibility and submit feedback |
| `app/auth/views/login.html` | Email/password login form |
| `app/auth/views/register.html` | Registration form with password confirmation |
| `app/auth/views/account.html` | Signed-in account page and logout form |
| `app/auth/user.go` | Application user structure |
| `app/auth/middleware.go` | Bearer-token authentication and role helper wrappers |
| `database/migrations/*_create_users_table.go` | Users table schema |

Routes and templates belong to your application. They are editable. HTML views compile into the Go binary, so restart `ctg dev` after editing them. No Node.js installation or frontend build is required for these default pages.

## Browser and API flows

| Browser | JSON API |
| --- | --- |
| GET/POST `/login` | POST `/api/auth/login` |
| GET/POST `/register` | POST `/api/auth/register` |
| GET `/account`, protected by `pages.Middleware()` | GET `/api/auth/me`, protected by `copyauth.Middleware()` |
| POST `/logout` clears the browser cookie | Clients discard their bearer token when logging out |
| Encrypted HttpOnly cookie | `Authorization: Bearer <token>` header |
| Form errors return readable HTML | Validation errors return JSON |

The browser cookie does not authenticate API routes. Existing API consumers keep using bearer tokens. Browser pages require Native Runtime; Lite supports the generated API handlers.

## Form and session behavior

- Missing/invalid fields show errors under the relevant input.
- Registration requires the password confirmation to match.
- Name and email remain in the form after validation fails; passwords do not.
- Invalid credentials show a message without exposing whether an email exists.
- Successful registration or login rotates the encrypted session and CSRF token.
- Guests visiting `/account` redirect to `/login`.
- Signed-in visitors to `/login` or `/register` redirect to `/account`.
- Every browser POST requires the hidden `_token` field from its rendered form.
- Browser responses use `Cache-Control: no-store`.
- Production uses a Secure, HttpOnly, SameSite=Lax cookie with the `__Host-` prefix. Serve production over HTTPS.
- Logging out clears the browser session; bearer tokens remain valid until their expiry.

Keep the hidden `_token` input when changing forms. Public multi-role registration uses `DefaultRole` from `app/auth/web.go` and ignores a role submitted by the client. Assign administrator roles through trusted application code.

## Upgrade an existing API-only starter

Stop the running server. Update the CLI and project module:

```powershell
go install github.com/arfajhf/copytygo/v4/cmd/ctg@v4.0.3
go get github.com/arfajhf/copytygo/v4@v4.0.3
go mod tidy
```

Run `ctg install:auth` again with the same mode originally installed, for example:

```powershell
ctg install:auth multi
ctg dev
```

Existing models, middleware, configuration, templates and users-table migration files remain intact. The installer upgrades an unchanged older generated `routes/auth.go` and adds missing browser starter files. An existing migrated users table does not need another migration.

If the older route file was customized, the installer stops before writing files. Integrate the browser routes manually instead of overwriting your changes. The default starter registers them as follows:

```go
pages := appauth.Pages()
app.Get("/login", pages.LoginPage).Name("auth.login")
app.Post("/login", pages.Login)
app.Get("/register", pages.RegisterPage).Name("auth.register")
app.Post("/register", pages.Register)
app.Get("/account", pages.Account).Middleware(pages.Middleware()).Name("auth.account")
app.Post("/logout", pages.Logout).Middleware(pages.Middleware())
```

Copy the starter `app/auth/web.go` and `app/auth/views` from a fresh throwaway project created with the same auth mode, then add the above routes and the `appauth "your-module/app/auth"` import to your application. Keep your existing API routes.
