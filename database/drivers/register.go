package drivers

import "github.com/arfajhf/copytygo/database"

func Register() {
	database.RegisterDriver(
		MySQL{},
	)

	database.RegisterDriver(
		PostgreSQL{},
	)
}
