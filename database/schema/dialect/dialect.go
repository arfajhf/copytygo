package dialect

import "github.com/arfajhf/copytygo/v4/database/schema"

type Dialect interface {
	Name() string

	Compile(
		blueprint *schema.Blueprint,
	) (string, error)
}
