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

Open the application welcome page. **Log in** and **Register** appear after the browser auth routes are installed. Register a user, and the application signs them in and redirects to `/dashboard`.

## Files to customize

| File | Purpose |
| --- | --- |
| `routes/auth.go` | Explicit browser and API routes and their middleware |
| `app/auth/web.go` | Default public registration role and embedded HTML views |
| `app/auth/views/layout.html` | Shared responsive layout, CSS, password visibility and submit feedback |
| `app/auth/views/login.html` | Email/password login form |
| `app/auth/views/register.html` | Registration form with password confirmation |
| `app/auth/views/app-layout.html` | Dashboard navbar, role menu and logout form |
| `app/auth/views/dashboard.html` | Member/admin dashboard |
| `app/auth/views/users.html` | Admin searchable user list and pagination |
| `app/auth/views/user-form.html` | Admin create/edit form |
| `app/auth/views/account.html` | Account details |
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
- Guests visiting `/dashboard`, `/account`, or `/users` redirect to `/login`.
- Signed-in visitors to `/login` or `/register` redirect to `/dashboard`.
- Every browser POST requires the hidden `_token` field from its rendered form.
- Browser responses use `Cache-Control: no-store`.
- Production uses a Secure, HttpOnly, SameSite=Lax cookie with the `__Host-` prefix. Serve production over HTTPS.
- Logging out clears the browser session; bearer tokens remain valid until their expiry.

Keep the hidden `_token` input when changing forms. Public multi-role registration uses `DefaultRole` from `app/auth/web.go` and ignores a role submitted by the client. Create the first administrator through the local CLI described below.

## Multi-role dashboard and user management

With `single`, the dashboard shows your profile and account menu. With `multi`,
the dashboard uses the current database role. Regular users see their own profile;
users with role `admin` also see user statistics and the **Users** menu.

Register your first account normally. In your own terminal, inside the project, run:

```powershell
ctg auth:admin your-email@example.com
```

Refresh `/dashboard`. No default admin password is created. This command uses the
project database configured in `.env` and requires an existing multi-role account.

Administrators can search users by name/email, browse 10 accounts per page, add
an account, edit name/email/role, change a password, and delete an account after
confirmation. Leave both password fields blank on edit to keep the current hash.
All writes require CSRF and server authorization. Password hashes are never
selected for display. Invalid/duplicate emails and confirmation errors return
readable form errors. Administrators cannot delete their own account or remove
their own admin role, preventing removal of the final administrator through the UI.

Every protected browser request loads the current user and role from the database.
Role revocation immediately blocks administration, profile edits appear on refresh,
and deleted users are redirected to login. Bearer APIs remain a separate flow.

| Method | Path | Access |
| --- | --- | --- |
| GET | `/dashboard`, `/account` | Signed-in user |
| GET | `/users` | Admin, multi-role only |
| GET | `/users/create`, `/users/:id/edit` | Admin, multi-role only |
| POST | `/users` | Create user, admin + CSRF |
| POST | `/users/:id` | Update user, admin + CSRF |
| POST | `/users/:id/delete` | Delete user, admin + CSRF |

## Upgrade an existing starter

Stop the running server. Update the CLI and project module:

```powershell
go install github.com/arfajhf/copytygo/v4/cmd/ctg@v4.0.4
go get github.com/arfajhf/copytygo/v4@v4.0.4
go mod tidy
```

Run `ctg install:auth` again with the same mode originally installed, for example:

```powershell
ctg install:auth multi
ctg dev
```

Customized files and the existing users-table migration remain intact. The installer upgrades unchanged v4.0.3 layouts, account templates and routes, and adds missing dashboard templates. Unchanged older API-only routes are also upgraded. An existing migrated users table does not need another migration.

If the older route file was customized, the installer stops before writing files. Integrate the browser routes manually instead of overwriting your changes. The default starter registers them as follows:

```go
pages := appauth.Pages()
app.Get("/login", pages.LoginPage).Name("auth.login")
app.Post("/login", pages.Login)
app.Get("/register", pages.RegisterPage).Name("auth.register")
app.Post("/register", pages.Register)
app.Get("/dashboard", pages.Dashboard).Middleware(pages.Middleware()).Name("auth.dashboard")
app.Get("/account", pages.Account).Middleware(pages.Middleware()).Name("auth.account")
app.Post("/logout", pages.Logout).Middleware(pages.Middleware())
if appauth.DefaultRole != "" {
    app.Get("/users", pages.Users).Middleware(pages.Middleware(), pages.RequireAdmin())
    app.Get("/users/create", pages.UserCreatePage).Middleware(pages.Middleware(), pages.RequireAdmin())
    app.Post("/users", pages.UserCreate).Middleware(pages.Middleware(), pages.RequireAdmin())
    app.Get("/users/:id/edit", pages.UserEditPage).Middleware(pages.Middleware(), pages.RequireAdmin())
    app.Post("/users/:id", pages.UserUpdate).Middleware(pages.Middleware(), pages.RequireAdmin())
    app.Post("/users/:id/delete", pages.UserDelete).Middleware(pages.Middleware(), pages.RequireAdmin())
}
```

Copy the starter `app/auth/web.go` and `app/auth/views` from a fresh throwaway project created with the same auth mode, then add the above routes and the `appauth "your-module/app/auth"` import to your application. Keep your existing API routes.
