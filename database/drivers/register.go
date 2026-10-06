package drivers

import "github.com/arfajhf/copytygo/v2/database"

func Register() {
	database.RegisterDriver(
		MySQL{},
	)

	database.RegisterDriver(
		PostgreSQL{},
	)
}
