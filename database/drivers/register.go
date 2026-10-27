package drivers

import "github.com/arfajhf/copytygo/v5/database"

func Register() {
	database.RegisterDriver(
		MySQL{},
	)

	database.RegisterDriver(
		PostgreSQL{},
	)
}
