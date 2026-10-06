package migrations

import (
	"github.com/arfajhf/copytygo/v2/database/migration"
	"github.com/arfajhf/copytygo/v2/database/schema"
)

func Register20261005195535() error {
	return migration.Register(
		"20261005_195535_create_categories_table",

		func() *schema.Blueprint {
			return schema.Create(
				"categories",
				func(table *schema.Table) {
					table.ID()

					// Add your columns here.

					table.Timestamps()
				},
			)
		},

		func() *schema.Blueprint {
			return schema.Drop(
				"categories",
			)
		},
	)
}
