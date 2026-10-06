# Upgrading to CopyTyGo v2

CopyTyGo v2 uses Go semantic import versioning. The module path changes from:

```text
github.com/arfajhf/copytygo
```

to:

```text
github.com/arfajhf/copytygo/v2
```

## Upgrade steps

Update the CLI:

```powershell
go install github.com/arfajhf/copytygo/v2/cmd/ctg@latest
```

Update framework imports in the application to include `/v2`, then update the dependency:

```powershell
go get github.com/arfajhf/copytygo/v2@latest
go mod tidy
```

After the v2 CLI is installed, future updates can use:

```powershell
ctg update
```

## Recommended verification

```powershell
ctg doctor
ctg doctor --db
ctg route:list
```

Then run the application:

```powershell
ctg dev
```

## Resource generation

v2 resource generation can create the model, controller, migration, validation and route registry together:

```powershell
ctg make:resource Product name:string price:decimal stock:integer description:text?
```

## Authentication

v2 auth installation also creates the users migration and HTTP routes:

```powershell
ctg install:auth single
ctg migrate
```

or:

```powershell
ctg install:auth multi
ctg migrate
```

Existing v1 projects can remain on the v1 module path until they are intentionally migrated.
