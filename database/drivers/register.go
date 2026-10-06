package drivers

import "github.com/arfajhf/copytygo/v3/database"

func Register() {
	database.RegisterDriver(
		MySQL{},
	)

	database.RegisterDriver(
		PostgreSQL{},
	)
}
