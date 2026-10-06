package core

import (
	"net/http"
	"strconv"

	"github.com/arfajhf/copytygo/v3/config"
	"github.com/arfajhf/copytygo/v3/database"
	"github.com/arfajhf/copytygo/v3/database/drivers"
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

type DBIndexOptions struct {
	Searchable []string
	Filterable []string
	Sortable   []string
}

func (ctx *Context) DBIndex(table string, configs ...DBIndexOptions) error {
	resource, err := databaseResource(table)
	if err != nil {
		return err
	}

	var config DBIndexOptions
	if len(configs) > 0 {
		config = configs[0]
	}

	page, _ := strconv.Atoi(ctx.Query("page"))
	perPage, _ := strconv.Atoi(ctx.Query("per_page"))

	filters := make(map[string]string)
	for _, field := range config.Filterable {
		filters[field] = ctx.Query("filter[" + field + "]")
	}

	result, err := resource.List(database.ResourceListOptions{
		Page:          page,
		PerPage:       perPage,
		Search:        ctx.Query("q"),
		SearchColumns: config.Searchable,
		Sort:          ctx.Query("sort"),
		Order:         ctx.Query("order"),
		Filters:       filters,
		Filterable:    config.Filterable,
		Sortable:      config.Sortable,
	})
	if err != nil {
		return err
	}

	return ctx.JSON(result)
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
