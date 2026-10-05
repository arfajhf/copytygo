package migrations

import (
	"github.com/arfajhf/copytygo/database/migration"
	"github.com/arfajhf/copytygo/database/schema"
)

func Register() error {
	if err := registerUsers(); err != nil {
		return err
	}

	if err := registerProducts(); err != nil {
		return err
	}

	if err := RegisterGenerated(); err != nil {
		return err
	}

	return nil
}

func registerUsers() error {
	return migration.Register(
		"20261002_001_create_users",

		func() *schema.Blueprint {
			return schema.Create(
				"users",
				func(table *schema.Table) {

					table.ID()

					table.String(
						"name",
						100,
					)

					table.String(
						"email",
						255,
					).Unique()

					table.String(
						"password",
						255,
					)

					table.Boolean(
						"active",
					).DefaultValue(true)

					table.Timestamps()
				},
			)
		},

		func() *schema.Blueprint {
			return schema.Drop(
				"users",
			)
		},
	)
}

func registerProducts() error {
	return migration.Register(
		"20261002_002_create_products",

		func() *schema.Blueprint {
			return schema.Create(
				"products",
				func(table *schema.Table) {

					table.ID()

					table.String(
						"name",
						150,
					)

					table.Text(
						"description",
					).Nullable()

					table.Decimal(
						"price",
						12,
						2,
					)

					table.Boolean(
						"active",
					).DefaultValue(true)

					table.Timestamps()
				},
			)
		},

		func() *schema.Blueprint {
			return schema.Drop(
				"products",
			)
		},
	)
}
