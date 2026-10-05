# Recommended code reading order

1. `core/application.go` and `core/router.go`
2. `core/context.go`, `core/request.go`, middleware files
3. `config/`
4. `database/database.go` and `database/drivers/`
5. `database/schema/` and `database/migration/`
6. `database/query/` and `database/model/`
7. `security/` and `validation/`
8. `httpclient/`
9. `cli/`
10. `frontend/`
11. `example/`

The framework is intentionally split this way so each package has one main responsibility.
