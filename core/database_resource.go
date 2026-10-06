package core

import (
	"net/http"

	"github.com/arfajhf/copytygo/config"
	"github.com/arfajhf/copytygo/database"
	"github.com/arfajhf/copytygo/database/drivers"
)

func databaseResource(table string) (*database.Resource, error) {
	drivers.Register()

	db, err := database.Connect()
	if err != nil {
		return nil, err
	}

	return database.NewResource(
		db,
		config.Get("DB_DRIVER", "mysql"),
		table,
	)
}

func (ctx *Context) DBIndex(table string) error {
	resource, err := databaseResource(table)
	if err != nil {
		return err
	}

	items, err := resource.Index()
	if err != nil {
		return err
	}

	return ctx.JSON(Map{"data": items})
}

func (ctx *Context) DBShow(table, id string) error {
	resource, err := databaseResource(table)
	if err != nil {
		return err
	}

	item, found, err := resource.Show(id)
	if err != nil {
		return err
	}
	if !found {
		return ctx.NotFound("Resource not found")
	}

	return ctx.JSON(item)
}

func (ctx *Context) DBStore(table string, values Map) error {
	resource, err := databaseResource(table)
	if err != nil {
		return err
	}

	item, err := resource.Store(map[string]any(values))
	if err != nil {
		return err
	}

	return ctx.Status(http.StatusCreated).JSON(item)
}

func (ctx *Context) DBUpdate(table, id string, values Map) error {
	resource, err := databaseResource(table)
	if err != nil {
		return err
	}

	item, found, err := resource.Update(id, map[string]any(values))
	if err != nil {
		return err
	}
	if !found {
		return ctx.NotFound("Resource not found")
	}

	return ctx.JSON(item)
}

func (ctx *Context) DBDestroy(table, id string) error {
	resource, err := databaseResource(table)
	if err != nil {
		return err
	}

	deleted, err := resource.Destroy(id)
	if err != nil {
		return err
	}
	if !deleted {
		return ctx.NotFound("Resource not found")
	}

	return ctx.NoContent(http.StatusNoContent)
}
