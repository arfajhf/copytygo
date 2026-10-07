# Your authentication starter

Start here:

1. Configure your MySQL or PostgreSQL connection in .env.
2. Run ctg migrate to create the users table.
3. Run ctg dev, then open the welcome page, /login, or /register.
4. Registration signs you in automatically and redirects to /dashboard.

Files you can edit:

- routes/auth.go: browser and API route definitions.
- app/auth/web.go: default registration role and HTML view binding.
- app/auth/views/layout.html: shared layout, responsive CSS, and form helpers.
- app/auth/views/login.html: login form.
- app/auth/views/register.html: registration form and password confirmation.
- app/auth/views/app-layout.html: dashboard navbar, role menu, CSS and logout form.
- app/auth/views/dashboard.html: member/admin dashboard.
- app/auth/views/users.html: searchable, paginated admin user list.
- app/auth/views/user-form.html: admin create/edit form.
- app/auth/views/account.html: your account details.
- app/auth/user.go: user structure.
- app/auth/middleware.go: API authentication and role helpers.

Restart ctg dev after editing Go code or embedded HTML views. Production builds
include the views and do not require Node.js for these pages.

Browser pages use encrypted HttpOnly cookies. Every POST form includes a CSRF
field. Keep that field when changing templates. API endpoints still use bearer
tokens and do not accept the browser cookie as API authentication.

Public multi-role registration assigns DefaultRole on the server. It never reads
a role field submitted by the user. Create the first administrator by registering normally, then running
ctg auth:admin your-email@example.com on your own terminal inside this project.
The account's role is read from the database on each browser request. Only admins
can manage /users. Passwords are hashed and never displayed. Your own admin role
and account cannot be removed from this menu.

Running install:auth again preserves customized files and routes. Unchanged
v4.0.3 layouts and routes are upgraded automatically and missing files are added.
Use the same auth mode as the original installation. Custom routes may need the
dashboard and user routes from docs/AUTHENTICATION.md added manually.
