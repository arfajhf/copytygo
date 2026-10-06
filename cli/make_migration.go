package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

var migrationNamePattern = regexp.MustCompile(
	`^[a-z][a-z0-9_]*$`,
)

func MakeMigration(
	name string,
	directory string,
) error {
	return makeMigration(name, directory, false)
}

func MakeResourceMigration(
	name string,
	directory string,
) error {
	return makeMigration(name, directory, true)
}

func makeMigration(
	name string,
	directory string,
	resource bool,
) error {

	name = strings.TrimSpace(
		strings.ToLower(name),
	)

	if name == "" {
		return fmt.Errorf(
			"migration name is required",
		)
	}

	if !migrationNamePattern.MatchString(name) {
		return fmt.Errorf(
			"migration name may only contain lowercase letters, numbers, and underscores",
		)
	}

	if err := os.MkdirAll(
		directory,
		0755,
	); err != nil {

		return fmt.Errorf(
			"unable to create migration directory: %w",
			err,
		)
	}

	now := time.Now()
	timestamp := now.Format(
		"20060102_150405",
	)

	filename := fmt.Sprintf(
		"%s_%s.go",
		timestamp,
		name,
	)

	path := filepath.Join(
		directory,
		filename,
	)

	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf(
			"migration %q already exists",
			filename,
		)
	}

	functionID := now.Format(
		"20060102150405",
	)

	tableName := detectTableName(name)

	content := migrationTemplate(
		functionID,
		timestamp+"_"+name,
		tableName,
	)
	if resource {
		content = resourceMigrationTemplate(
			functionID,
			timestamp+"_"+name,
			tableName,
		)
	}

	if err := os.WriteFile(
		path,
		[]byte(content),
		0644,
	); err != nil {

		return fmt.Errorf(
			"unable to create migration: %w",
			err,
		)
	}

	if err := GenerateMigrationRegistry(
		directory,
	); err != nil {

		_ = os.Remove(path)

		return fmt.Errorf(
			"migration created but registry generation failed: %w",
			err,
		)
	}

	fmt.Println()
	fmt.Println("Migration created:")
	fmt.Println(path)
	fmt.Println()

	return nil
}

func detectTableName(
	name string,
) string {

	const createPrefix = "create_"
	const tableSuffix = "_table"

	if strings.HasPrefix(
		name,
		createPrefix,
	) && strings.HasSuffix(
		name,
		tableSuffix,
	) {

		table := strings.TrimPrefix(
			name,
			createPrefix,
		)

		table = strings.TrimSuffix(
			table,
			tableSuffix,
		)

		if table != "" {
			return table
		}
	}

	return "change_me"
}

func migrationTemplate(
	functionID string,
	migrationName string,
	tableName string,
) string {

	return fmt.Sprintf(`package migrations

import (
	"github.com/arfajhf/copytygo/v2/database/migration"
	"github.com/arfajhf/copytygo/v2/database/schema"
)

func Register%s() error {
	return migration.Register(
		%q,

		func() *schema.Blueprint {
			return schema.Create(
				%q,
				func(table *schema.Table) {
					table.ID()

					// Add your columns here.

					table.Timestamps()
				},
			)
		},

		func() *schema.Blueprint {
			return schema.Drop(
				%q,
			)
		},
	)
}
`,
		functionID,
		migrationName,
		tableName,
		tableName,
	)
}


func resourceMigrationTemplate(
	functionID string,
	migrationName string,
	tableName string,
) string {
	return fmt.Sprintf(`package migrations

import (
	"github.com/arfajhf/copytygo/v2/database/migration"
	"github.com/arfajhf/copytygo/v2/database/schema"
)

func Register%s() error {
	return migration.Register(
		%q,

		func() *schema.Blueprint {
			return schema.Create(
				%q,
				func(table *schema.Table) {
					table.ID()
					table.String("name")
					table.Timestamps()
				},
			)
		},

		func() *schema.Blueprint {
			return schema.Drop(
				%q,
			)
		},
	)
}
`,
		functionID,
		migrationName,
		tableName,
		tableName,
	)
}
