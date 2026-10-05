package drivers

import (
	"fmt"
	"net/url"

	"github.com/arfajhf/copytygo/database"

	_ "github.com/go-sql-driver/mysql"
)

type MySQL struct{}

func (MySQL) Name() string {
	return "mysql"
}

func (MySQL) SQLDriver() string {
	return "mysql"
}

func (MySQL) DSN(config database.Config) string {
	query := url.Values{}

	query.Set("parseTime", "true")
	query.Set("charset", "utf8mb4")
	query.Set("loc", "Local")

	return fmt.Sprintf(
		"%s:%s@tcp(%s:%s)/%s?%s",
		config.Username,
		config.Password,
		config.Host,
		config.Port,
		config.Database,
		query.Encode(),
	)
}
