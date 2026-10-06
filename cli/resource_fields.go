package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

type ResourceField struct {
	Name     string
	Type     string
	Nullable bool
}

var resourceFieldPattern = regexp.MustCompile("^[A-Za-z][A-Za-z0-9_]*$")

func ParseResourceFields(specs []string) ([]ResourceField, error) {
	if len(specs) == 0 {
		return []ResourceField{{Name: "name", Type: "string"}}, nil
	}

	fields := make([]ResourceField, 0, len(specs))
	seen := map[string]bool{}

	for _, spec := range specs {
		parts := strings.SplitN(strings.TrimSpace(spec), ":", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("copytygo: invalid field %q, expected name:type", spec)
		}

		name := strings.TrimSpace(parts[0])
		typ := strings.ToLower(strings.TrimSpace(parts[1]))
		nullable := strings.HasSuffix(typ, "?")
		typ = strings.TrimSuffix(typ, "?")

		if !resourceFieldPattern.MatchString(name) {
			return nil, fmt.Errorf("copytygo: invalid field name %q", name)
		}
		if seen[name] {
			return nil, fmt.Errorf("copytygo: duplicate field %q", name)
		}
		seen[name] = true

		switch typ {
		case "string", "text", "integer", "bigint", "boolean", "decimal", "json", "uuid", "datetime", "timestamp":
		default:
			return nil, fmt.Errorf("copytygo: unsupported field type %q", typ)
		}

		fields = append(fields, ResourceField{Name: name, Type: typ, Nullable: nullable})
	}

	return fields, nil
}

func MakeResourceWithFields(name string, specs []string) error {
	if !safeName.MatchString(name) {
		return fmt.Errorf("invalid resource name")
	}

	fields, err := ParseResourceFields(specs)
	if err != nil {
		return err
	}

	if err := writeResourceModel(name, fields); err != nil {
		return err
	}
	if err := writeResourceController(name, fields); err != nil {
		return err
	}
	if err := writeResourceMigration(name, fields); err != nil {
		return err
	}
	if err := GenerateResourceRoutes("."); err != nil {
		return fmt.Errorf("copytygo: resource files created, but route generation failed: %w", err)
	}

	table := strings.ToLower(name) + "s"
	fmt.Println()
	fmt.Printf("Resource %s created.\n", name)
	fmt.Printf("Route: /%s\n", table)
	fmt.Printf("Fields: %s\n", resourceFieldSummary(fields))
	fmt.Println()
	return nil
}

func writeResourceModel(name string, fields []ResourceField) error {
	var body strings.Builder
	body.WriteString("package models\n\n")
	body.WriteString("type " + name + " struct {\n")
	body.WriteString("\tID int64 `json:\"id\"`\n")
	for _, field := range fields {
		body.WriteString(fmt.Sprintf(
			"\t%s %s `json:\"%s\"`\n",
			exportedName(field.Name),
			goTypeForResourceField(field),
			field.Name,
		))
	}
	body.WriteString("\tCreatedAt string `json:\"created_at\"`\n")
	body.WriteString("\tUpdatedAt string `json:\"updated_at\"`\n")
	body.WriteString("}\n")

	return writeGenerated(
		filepath.Join("app", "models", strings.ToLower(name)+".go"),
		body.String(),
	)
}

func writeResourceController(name string, fields []ResourceField) error {
	controller := name
	if !strings.HasSuffix(controller, "Controller") {
		controller += "Controller"
	}
	resource := strings.ToLower(strings.TrimSuffix(controller, "Controller")) + "s"

	var rules strings.Builder
	var values strings.Builder
	for _, field := range fields {
		rule := validationRuleForResourceField(field)
		if rule != "" {
			rules.WriteString(fmt.Sprintf("\t\t\"%s\": %q,\n", field.Name, rule))
		}
		values.WriteString(fmt.Sprintf("\t\t\"%s\": ctx.Input(\"%s\"),\n", field.Name, field.Name))
	}

	body := fmt.Sprintf(`package controllers

import "github.com/arfajhf/copytygo/v2/core"

// copytygo:resource %s

type %s struct{}

func (%s) Index(ctx *core.Context) error {
	return ctx.DBIndex("%s")
}

func (%s) Show(ctx *core.Context) error {
	return ctx.DBShow("%s", ctx.Param("id"))
}

func (%s) Store(ctx *core.Context) error {
	if err := ctx.Validate(map[string]string{
%s	}); err != nil {
		return err
	}

	return ctx.DBStore("%s", core.Map{
%s	})
}

func (%s) Update(ctx *core.Context) error {
	if err := ctx.Validate(map[string]string{
%s	}); err != nil {
		return err
	}

	return ctx.DBUpdate("%s", ctx.Param("id"), core.Map{
%s	})
}

func (%s) Destroy(ctx *core.Context) error {
	return ctx.DBDestroy("%s", ctx.Param("id"))
}
`,
		resource,
		controller,
		controller, resource,
		controller, resource,
		controller, rules.String(), resource, values.String(),
		controller, rules.String(), resource, values.String(),
		controller, resource,
	)

	return writeGenerated(
		filepath.Join("app", "controllers", strings.ToLower(strings.TrimSuffix(controller, "Controller"))+"_controller.go"),
		body,
	)
}

func writeResourceMigration(name string, fields []ResourceField) error {
	directory := filepath.Join("database", "migrations")
	if err := os.MkdirAll(directory, 0755); err != nil {
		return err
	}

	table := strings.ToLower(name) + "s"
	now := time.Now()
	timestamp := now.Format("20060102_150405")
	functionID := now.Format("20060102150405")
	migrationName := timestamp + "_create_" + table + "_table"
	path := filepath.Join(directory, migrationName+".go")

	var columns strings.Builder
	for _, field := range fields {
		expr := migrationColumnForResourceField(field)
		columns.WriteString("\t\t\t\t\t" + expr + "\n")
	}

	content := fmt.Sprintf(`package migrations

import (
	"github.com/arfajhf/copytygo/v2/database/migration"
	"github.com/arfajhf/copytygo/v2/database/schema"
)

func Register%s() error {
	return migration.Register(
		%q,
		func() *schema.Blueprint {
			return schema.Create(%q, func(table *schema.Table) {
				table.ID()
%s				table.Timestamps()
			})
		},
		func() *schema.Blueprint {
			return schema.Drop(%q)
		},
	)
}
`, functionID, migrationName, table, columns.String(), table)

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return err
	}
	return GenerateMigrationRegistry(directory)
}

func migrationColumnForResourceField(field ResourceField) string {
	var call string
	switch field.Type {
	case "string":
		call = fmt.Sprintf("table.String(%q)", field.Name)
	case "text":
		call = fmt.Sprintf("table.Text(%q)", field.Name)
	case "integer":
		call = fmt.Sprintf("table.Integer(%q)", field.Name)
	case "bigint":
		call = fmt.Sprintf("table.BigInteger(%q)", field.Name)
	case "boolean":
		call = fmt.Sprintf("table.Boolean(%q)", field.Name)
	case "decimal":
		call = fmt.Sprintf("table.Decimal(%q, 12, 2)", field.Name)
	case "json":
		call = fmt.Sprintf("table.JSON(%q)", field.Name)
	case "uuid":
		call = fmt.Sprintf("table.UUID(%q)", field.Name)
	case "datetime":
		call = fmt.Sprintf("table.DateTime(%q)", field.Name)
	case "timestamp":
		call = fmt.Sprintf("table.Timestamp(%q)", field.Name)
	}
	if field.Nullable {
		call += ".Nullable()"
	}
	return call
}

func validationRuleForResourceField(field ResourceField) string {
	rules := make([]string, 0, 2)
	if !field.Nullable {
		rules = append(rules, "required")
	}
	switch field.Type {
	case "integer", "bigint":
		rules = append(rules, "integer")
	}
	return strings.Join(rules, "|")
}

func goTypeForResourceField(field ResourceField) string {
	switch field.Type {
	case "integer", "bigint":
		return "int64"
	case "boolean":
		return "bool"
	case "decimal":
		return "float64"
	case "json":
		return "any"
	default:
		return "string"
	}
}

func exportedName(name string) string {
	parts := strings.Split(name, "_")
	for i, part := range parts {
		if part == "" {
			continue
		}
		parts[i] = strings.ToUpper(part[:1]) + part[1:]
	}
	return strings.Join(parts, "")
}

func resourceFieldSummary(fields []ResourceField) string {
	values := make([]string, 0, len(fields))
	for _, field := range fields {
		suffix := ""
		if field.Nullable {
			suffix = "?"
		}
		values = append(values, field.Name+":"+field.Type+suffix)
	}
	return strings.Join(values, ", ")
}
