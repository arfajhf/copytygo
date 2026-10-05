package migrations

import (
	"github.com/arfajhf/copytygo/database/migration"
	"github.com/arfajhf/copytygo/database/schema"
)

func Register20261003074008() error {
	return migration.Register(
		"20261003_074008_create_orders_table",

		func() *schema.Blueprint {
			return schema.Create(
				"orders",
				func(table *schema.Table) {
					table.ID()

					// Add your columns here.

					table.Timestamps()
				},
			)
		},

		func() *schema.Blueprint {
			return schema.Drop(
				"orders",
			)
		},
	)
}
