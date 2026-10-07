# TypeScript + Go authentication

## Install and run

Choose `ctg install:auth single` or `ctg install:auth multi`. Configure your database in `.env`, then run:

```powershell
ctg migrate
ctg dev
```

Use Node.js 22+ and Go 1.27.1+. `ctg dev` installs frontend dependencies when needed and starts Vite alongside the native Go application. Open the **application URL** printed by Go. TypeScript pages and Go JSON APIs share that origin. Login/Register appear in a full-width navbar at the top. Successful registration/login opens `/dashboard`.

## Source layout

| File | Responsibility |
| --- | --- |
| `frontend/src/auth/views.ts` | Welcome/navbar, login/register, dashboard, account, Users and create/edit form |
| `frontend/src/auth/main.ts` | Page selection and form interactions |
| `frontend/src/auth/api.ts` | Typed JSON client, session and user response types |
| `frontend/src/auth/style.css` | Responsive styling |
| `frontend/auth.html` | Minimal Vite entry, loads the TypeScript module |
| `frontend/vite.auth.config.ts` | Auth entry combined with the existing frontend preset |
| `routes/auth.go` | Go JSON endpoints, middleware and frontend page routes |
| `app/auth/web.go` | Default registration role and JSON session controller |
| `app/auth/user.go` | Application user shape |
| `app/auth/middleware.go` | Bearer authentication/role helpers |
| `database/migrations/*_create_users_table.go` | Users schema |

Auth UI uses TypeScript under the existing frontend folder, including React/Vue presets. Your existing frontend index and preset are preserved; the auth starter has a separate entry you can customize or integrate into your chosen framework. No Go HTML view files are generated. TypeScript edits reload through Vite; restart after Go edits. API-only projects generate bearer routes without a frontend or Node dependency.

## JSON APIs and sessions

| Method | Endpoint | Access |
| --- | --- | --- |
| GET | `/api/session` | Guest or member; returns current user and CSRF token |
| POST | `/api/session/register`, `/api/session/login` | CSRF; creates encrypted browser session |
| POST | `/api/session/logout` | Signed in + CSRF; clears cookie |
| GET | `/api/dashboard` | Signed in; admin also receives user statistics |
| GET | `/api/users`, `/api/users/:id` | Admin in multi-role mode |
| POST | `/api/users` | Admin + CSRF; create |
| PUT | `/api/users/:id` | Admin + CSRF; update |
| DELETE | `/api/users/:id` | Admin + CSRF; delete |
| POST | `/api/auth/register`, `/api/auth/login` | Existing bearer client APIs |
| GET | `/api/auth/me` | Authorization: Bearer token |

The browser holds authentication in an encrypted HttpOnly, SameSite=Lax cookie. JavaScript stores only the CSRF token in memory and sends it as `X-CSRF-Token` on writes. Login/registration rotate the session and CSRF token. Browser responses use `Cache-Control: no-store`; passwords and session tokens never appear in browser JSON responses. Production cookies are Secure with the `__Host-` prefix; serve production over HTTPS.

Frontend redirects guests to login and signed-in visitors away from login/register. Backend APIs independently return 401/403 and enforce current database roles. Hiding a menu never grants permission. Bearer routes keep their own authentication and do not accept browser cookies.

## Multi-role dashboard and Users

Public registration always assigns the server's `DefaultRole` (`user` in multi mode). Register the first account, then promote it locally:

```powershell
ctg auth:admin your-email@example.com
```

Refresh `/dashboard`. Members see their own workspace; administrators see user statistics and the Users menu. Administrators can search and paginate accounts, create, update, change roles/passwords, and delete after confirmation. Leave both password fields blank during editing to keep the existing password. Passwords are hashed. Go checks administrator writes within a database transaction and prevents deleting/demoting your own admin account. Role revocation and deleted accounts take effect on the next API request.

## Build and deploy

```powershell
ctg build
```

This type-checks/builds TypeScript and compiles Go. Deploy the complete `build/` folder: `app` (`app.exe` on Windows) plus `frontend/dist/`. Start the application from that folder, with production environment variables and the database configured. Node.js is required for development/building and is not required on the production server. Set `COPYTYGO_FRONTEND_DIST` if serving compiled assets from another directory. Production ignores the dev Vite proxy.

## Upgrade from the old HTML starter

Stop the server, then update the CLI and module inside your project:

```powershell
go install github.com/arfajhf/copytygo/v4/cmd/ctg@v4.0.5
go get github.com/arfajhf/copytygo/v4@v4.0.5
go mod tidy
ctg install:auth multi
ctg dev
```

Use `single` if that was your original mode. Reinstallation migrates unchanged v4.0.3/v4.0.4 routes/controllers to the JSON + TypeScript starter, adds frontend sources, and removes unchanged generated Go HTML views. It preserves the existing users migration and database. An already migrated users table needs no new migration.

Customized routes/controllers stop automatic migration before any files are written. Use a fresh throwaway project with the same mode as a reference: copy its `frontend/src/auth/`, `frontend/auth.html`, `frontend/vite.auth.config.ts` and adapt its `routes/auth.go` and `app/auth/web.go` into your custom application. Keep your existing API behavior and custom frontend configuration. Customized old HTML files are retained on disk so you can move their UI to TypeScript; they are no longer used by the new default starter. Customized TypeScript source is preserved on reinstall.
