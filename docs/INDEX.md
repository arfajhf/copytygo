# CopyTyGo documentation

CopyTyGo is a full-stack Go + TypeScript framework. These documents describe the **v5 release candidate**, intended to become CopyTyGo's first publicly recommended stable release. The v4.0.6 tag was an internal prelaunch version. Do not publish stable-install instructions for v5 until the final immutable tag is available.

## Start here

1. [Architecture](ARCHITECTURE.md): HTTP core, routing, service container, database, background jobs and frontend architecture.
2. [Authentication](AUTHENTICATION.md): install single/multi-role authentication, run Vite + native Go, configure sessions and edit the TypeScript views.
3. [Public assets](PUBLIC_ASSETS.md): publish logo files, images and fonts correctly.
3a. [Generated resource security](RESOURCE_SECURITY.md): private-by-default CRUD, role enforcement, CSRF and upgrading old routes.
4. [Development runtime](DEV_RUNTIME.md): understand Native vs Lite and the stable project-local Windows native build.
5. [WSL development](WSL_DEVELOPMENT.md): run complete native Go + TypeScript while Windows Smart App Control stays enabled.
6. [Windows troubleshooting](WINDOWS_DEVELOPMENT.md): inspect Defender Antivirus and CodeIntegrity logs safely.
7. [Production checklist](PRODUCTION.md): environment, database, queue workers, application health and deployment.
8. [Upgrade v4 to v5](UPGRADE_V5.md): module path, breaking changes and migration checks.
9. [V5 release gates](V5_LAUNCH_CHECKLIST.md): track outstanding launch validation.
10. [Recommended code reading order](LEARNING_PATH.md): navigate framework packages.

## Development commands

```powershell
ctg new example-app
cd example-app
ctg doctor
ctg dev
```

- `ctg dev`: full Native Runtime. On Windows, this builds `build/dev/app.exe` in a predictable project-local location.
- `ctg dev --build`: compile and run the same stable native binary on any OS.
- `ctg dev --wsl`: run full Linux Native Runtime from an allowed Windows CLI; configure WSL first. If Windows blocks that CLI, run `ctg dev` directly in WSL instead.
- `ctg dev --lite`: explicitly run the limited Lite Runtime.
- `ctg studio`: open the local development Studio, available at `/__copytygo`.
- `ctg build`: compile a deployable native application and frontend assets.
- `ctg doctor --db --production`: check the configuration of a production deployment.

## Security baseline

Never expose Studio on a production server. Set `APP_ENV=production` and `APP_DEBUG=false`, configure a strong and unique `APP_KEY`, and enforce HTTPS in production. Development execution blocked by Windows security should be diagnosed through policy/antivirus logs, not through global protection disabling.

This index and all attached docs are maintained as source-controlled documentation. The official public site is planned, not yet live. Its default installation and documentation links must resolve to the latest **stable** released version. Keep historical versions accessible through explicitly versioned URLs. See [website release plan](WEBSITE_LAUNCH_PLAN.md).
