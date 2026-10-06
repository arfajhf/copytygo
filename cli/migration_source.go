package cli

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/arfajhf/copytygo/v2/database/migration"
	"github.com/arfajhf/copytygo/v2/database/schema"
)

type migrationColumnSpec struct {
	Method     string
	Name       string
	IntArgs    []int
	Nullable   bool
	Unique     bool
	HasDefault bool
	Default    any
}

func RegisterProjectMigrations(directory string) error {
	migration.ResetRegistry()

	entries, err := os.ReadDir(directory)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("copytygo: migration directory %s not found", directory)
		}
		return err
	}

	names := make([]string, 0)
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".go" {
			continue
		}
		if entry.Name() == "registry.go" || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		names = append(names, entry.Name())
	}
	sort.Strings(names)

	registered := 0
	for _, name := range names {
		count, err := registerMigrationSourceFile(filepath.Join(directory, name))
		if err != nil {
			return fmt.Errorf("copytygo: unable to load migration %s: %w", name, err)
		}
		registered += count
	}

	if registered == 0 {
		return nil
	}
	return nil
}

func registerMigrationSourceFile(path string) (int, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, nil, 0)
	if err != nil {
		return 0, err
	}

	count := 0
	var parseErr error

	ast.Inspect(file, func(node ast.Node) bool {
		if parseErr != nil {
			return false
		}

		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}

		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || sel.Sel.Name != "Register" || len(call.Args) < 2 {
			return true
		}

		pkg, ok := sel.X.(*ast.Ident)
		if !ok || pkg.Name != "migration" {
			return true
		}

		name, ok := sourceString(call.Args[0])
		if !ok {
			parseErr = fmt.Errorf("migration name must be a string literal")
			return false
		}

		up, err := parseMigrationFactory(call.Args[1])
		if err != nil {
			parseErr = fmt.Errorf("%s Up: %w", name, err)
			return false
		}

		var down *schema.Blueprint
		if len(call.Args) >= 3 {
			down, err = parseMigrationFactory(call.Args[2])
			if err != nil {
				parseErr = fmt.Errorf("%s Down: %w", name, err)
				return false
			}
		}

		upBlueprint := up
		downBlueprint := down

		var downFactory migration.BlueprintFactory
		if downBlueprint != nil {
			downFactory = func() *schema.Blueprint {
				return downBlueprint
			}
		}

		if err := migration.Register(
			name,
			func() *schema.Blueprint {
				return upBlueprint
			},
			downFactory,
		); err != nil {
			parseErr = err
			return false
		}

		count++
		return true
	})

	return count, parseErr
}

func parseMigrationFactory(expr ast.Expr) (*schema.Blueprint, error) {
	fn, ok := expr.(*ast.FuncLit)
	if !ok {
		return nil, fmt.Errorf("expected migration factory function")
	}

	for _, stmt := range fn.Body.List {
		ret, ok := stmt.(*ast.ReturnStmt)
		if !ok || len(ret.Results) != 1 {
			continue
		}
		return parseMigrationBlueprint(ret.Results[0])
	}

	return nil, fmt.Errorf("migration factory must return a schema blueprint")
}

func parseMigrationBlueprint(expr ast.Expr) (*schema.Blueprint, error) {
	call, ok := expr.(*ast.CallExpr)
	if !ok {
		return nil, fmt.Errorf("expected schema.Create or schema.Drop")
	}

	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return nil, fmt.Errorf("expected schema operation")
	}

	pkg, ok := sel.X.(*ast.Ident)
	if !ok || pkg.Name != "schema" {
		return nil, fmt.Errorf("expected schema.Create or schema.Drop")
	}

	switch sel.Sel.Name {
	case "Drop":
		if len(call.Args) != 1 {
			return nil, fmt.Errorf("schema.Drop requires a table name")
		}
		table, ok := sourceString(call.Args[0])
		if !ok {
			return nil, fmt.Errorf("schema.Drop table must be a string literal")
		}
		return schema.Drop(table), nil

	case "Create":
		if len(call.Args) != 2 {
			return nil, fmt.Errorf("schema.Create requires table name and callback")
		}

		tableName, ok := sourceString(call.Args[0])
		if !ok {
			return nil, fmt.Errorf("schema.Create table must be a string literal")
		}

		callback, ok := call.Args[1].(*ast.FuncLit)
		if !ok {
			return nil, fmt.Errorf("schema.Create callback must be a function")
		}

		operations := make([]func(*schema.Table), 0, len(callback.Body.List))
		for _, stmt := range callback.Body.List {
			exprStmt, ok := stmt.(*ast.ExprStmt)
			if !ok {
				return nil, fmt.Errorf("only direct table schema calls are supported in project migrations")
			}

			op, err := parseMigrationTableOperation(exprStmt.X)
			if err != nil {
				return nil, err
			}
			operations = append(operations, op)
		}

		return schema.Create(tableName, func(table *schema.Table) {
			for _, op := range operations {
				op(table)
			}
		}), nil
	}

	return nil, fmt.Errorf("unsupported schema operation %s", sel.Sel.Name)
}

func parseMigrationTableOperation(expr ast.Expr) (func(*schema.Table), error) {
	call, ok := expr.(*ast.CallExpr)
	if !ok {
		return nil, fmt.Errorf("expected table schema call")
	}

	spec, timestamps, err := parseMigrationColumnChain(call)
	if err != nil {
		return nil, err
	}

	if timestamps {
		return func(table *schema.Table) {
			table.Timestamps()
		}, nil
	}

	return func(table *schema.Table) {
		column := applyMigrationColumn(table, spec)

		if spec.Nullable {
			column.Nullable()
		}
		if spec.Unique {
			column.Unique()
		}
		if spec.HasDefault {
			column.DefaultValue(spec.Default)
		}
	}, nil
}

func parseMigrationColumnChain(call *ast.CallExpr) (migrationColumnSpec, bool, error) {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return migrationColumnSpec{}, false, fmt.Errorf("invalid table schema call")
	}

	if tableIdent, ok := sel.X.(*ast.Ident); ok && tableIdent.Name == "table" {
		return parseMigrationBaseColumn(sel.Sel.Name, call.Args)
	}

	inner, ok := sel.X.(*ast.CallExpr)
	if !ok {
		return migrationColumnSpec{}, false, fmt.Errorf("unsupported schema chain")
	}

	spec, timestamps, err := parseMigrationColumnChain(inner)
	if err != nil {
		return migrationColumnSpec{}, false, err
	}
	if timestamps {
		return migrationColumnSpec{}, false, fmt.Errorf("Timestamps cannot be chained")
	}

	switch sel.Sel.Name {
	case "Nullable":
		if len(call.Args) != 0 {
			return migrationColumnSpec{}, false, fmt.Errorf("Nullable does not accept arguments")
		}
		spec.Nullable = true

	case "Unique":
		if len(call.Args) != 0 {
			return migrationColumnSpec{}, false, fmt.Errorf("Unique does not accept arguments")
		}
		spec.Unique = true

	case "DefaultValue":
		if len(call.Args) != 1 {
			return migrationColumnSpec{}, false, fmt.Errorf("DefaultValue requires one argument")
		}
		value, err := sourceDefaultValue(call.Args[0])
		if err != nil {
			return migrationColumnSpec{}, false, err
		}
		spec.HasDefault = true
		spec.Default = value

	default:
		return migrationColumnSpec{}, false, fmt.Errorf("unsupported column modifier %s", sel.Sel.Name)
	}

	return spec, false, nil
}

func parseMigrationBaseColumn(method string, args []ast.Expr) (migrationColumnSpec, bool, error) {
	if method == "Timestamps" {
		if len(args) != 0 {
			return migrationColumnSpec{}, false, fmt.Errorf("Timestamps does not accept arguments")
		}
		return migrationColumnSpec{}, true, nil
	}

	spec := migrationColumnSpec{Method: method}

	if method == "ID" {
		if len(args) != 0 {
			return migrationColumnSpec{}, false, fmt.Errorf("ID does not accept arguments")
		}
		return spec, false, nil
	}

	if len(args) < 1 {
		return migrationColumnSpec{}, false, fmt.Errorf("%s requires a column name", method)
	}

	name, ok := sourceString(args[0])
	if !ok {
		return migrationColumnSpec{}, false, fmt.Errorf("%s column name must be a string literal", method)
	}
	spec.Name = name

	switch method {
	case "String":
		if len(args) > 1 {
			length, ok := sourceInt(args[1])
			if !ok {
				return migrationColumnSpec{}, false, fmt.Errorf("String length must be an integer")
			}
			spec.IntArgs = []int{length}
		}

	case "Decimal":
		if len(args) != 3 {
			return migrationColumnSpec{}, false, fmt.Errorf("Decimal requires precision and scale")
		}
		precision, ok := sourceInt(args[1])
		if !ok {
			return migrationColumnSpec{}, false, fmt.Errorf("Decimal precision must be an integer")
		}
		scale, ok := sourceInt(args[2])
		if !ok {
			return migrationColumnSpec{}, false, fmt.Errorf("Decimal scale must be an integer")
		}
		spec.IntArgs = []int{precision, scale}

	case "Text", "Integer", "BigInteger", "Boolean", "Timestamp", "DateTime", "JSON", "UUID", "ForeignID":
		if len(args) != 1 {
			return migrationColumnSpec{}, false, fmt.Errorf("%s only accepts a column name", method)
		}

	default:
		return migrationColumnSpec{}, false, fmt.Errorf("unsupported table method %s", method)
	}

	return spec, false, nil
}

func applyMigrationColumn(table *schema.Table, spec migrationColumnSpec) *schema.Column {
	switch spec.Method {
	case "ID":
		return table.ID()
	case "String":
		if len(spec.IntArgs) == 1 {
			return table.String(spec.Name, spec.IntArgs[0])
		}
		return table.String(spec.Name)
	case "Text":
		return table.Text(spec.Name)
	case "Integer":
		return table.Integer(spec.Name)
	case "BigInteger":
		return table.BigInteger(spec.Name)
	case "Boolean":
		return table.Boolean(spec.Name)
	case "Decimal":
		return table.Decimal(spec.Name, spec.IntArgs[0], spec.IntArgs[1])
	case "Timestamp":
		return table.Timestamp(spec.Name)
	case "DateTime":
		return table.DateTime(spec.Name)
	case "JSON":
		return table.JSON(spec.Name)
	case "UUID":
		return table.UUID(spec.Name)
	case "ForeignID":
		return table.ForeignID(spec.Name)
	default:
		panic("copytygo: unsupported parsed migration column")
	}
}

func sourceString(expr ast.Expr) (string, bool) {
	lit, ok := expr.(*ast.BasicLit)
	if !ok || lit.Kind != token.STRING {
		return "", false
	}
	value, err := strconv.Unquote(lit.Value)
	return value, err == nil
}

func sourceInt(expr ast.Expr) (int, bool) {
	lit, ok := expr.(*ast.BasicLit)
	if !ok || lit.Kind != token.INT {
		return 0, false
	}
	value, err := strconv.Atoi(lit.Value)
	return value, err == nil
}

func sourceDefaultValue(expr ast.Expr) (any, error) {
	if lit, ok := expr.(*ast.BasicLit); ok {
		switch lit.Kind {
		case token.STRING:
			value, err := strconv.Unquote(lit.Value)
			if err != nil {
				return nil, err
			}
			return value, nil
		case token.INT:
			value, err := strconv.ParseInt(lit.Value, 10, 64)
			if err != nil {
				return nil, err
			}
			return value, nil
		case token.FLOAT:
			value, err := strconv.ParseFloat(lit.Value, 64)
			if err != nil {
				return nil, err
			}
			return value, nil
		}
	}

	if ident, ok := expr.(*ast.Ident); ok {
		switch ident.Name {
		case "true":
			return true, nil
		case "false":
			return false, nil
		case "nil":
			return nil, nil
		}
	}

	if call, ok := expr.(*ast.CallExpr); ok {
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if ok && sel.Sel.Name == "Raw" && len(call.Args) == 1 {
			pkg, ok := sel.X.(*ast.Ident)
			if ok && pkg.Name == "schema" {
				value, ok := sourceString(call.Args[0])
				if !ok {
					return nil, fmt.Errorf("schema.Raw requires a string literal")
				}
				return schema.Raw(value), nil
			}
		}
	}

	return nil, fmt.Errorf("unsupported default value")
}
