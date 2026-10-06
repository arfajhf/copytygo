package migration

import "github.com/arfajhf/copytygo/v2/database/schema"

type BlueprintFactory func() *schema.Blueprint

type Migration struct {
	Name string
	Up   BlueprintFactory
	Down BlueprintFactory
}
