package drivers

import "github.com/arfajhf/copytygo/v4/database"

func Register() {
	database.RegisterDriver(
		MySQL{},
	)

	database.RegisterDriver(
		PostgreSQL{},
	)
}
