# Your authentication starter

Start here:

1. Configure your MySQL or PostgreSQL connection in .env.
2. Run ctg migrate to create the users table.
3. Run ctg dev, then open the welcome page, /login, or /register.
4. Registration signs you in automatically and redirects to /account.

Files you can edit:

- routes/auth.go: browser and API route definitions.
- app/auth/web.go: default registration role and HTML view binding.
- app/auth/views/layout.html: shared layout, responsive CSS, and form helpers.
- app/auth/views/login.html: login form.
- app/auth/views/register.html: registration form and password confirmation.
- app/auth/views/account.html: signed-in page and logout form.
- app/auth/user.go: user structure.
- app/auth/middleware.go: API authentication and role helpers.

Restart ctg dev after editing Go code or embedded HTML views. Production builds
include the views and do not require Node.js for these pages.

Browser pages use encrypted HttpOnly cookies. Every POST form includes a CSRF
field. Keep that field when changing templates. API endpoints still use bearer
tokens and do not accept the browser cookie as API authentication.

Public multi-role registration assigns DefaultRole on the server. It never reads
a role field submitted by the user. Assign privileged roles through trusted
administrative code.

Running install:auth again preserves existing starter files and custom routes.
An unchanged older CopyTyGo API-only routes file is upgraded automatically.
