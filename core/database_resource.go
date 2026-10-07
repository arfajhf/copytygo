package core

import (
	"net/http"
	"strconv"

	"github.com/arfajhf/copytygo/v3/config"
	"github.com/arfajhf/copytygo/v3/database"
	"github.com/arfajhf/copytygo/v3/database/drivers"
)

func databaseResource(table string, softDeletes bool) (*database.Resource, error) {
	drivers.Register()

	db, err := database.Connect()
	if err != nil {
		return nil, err
	}

	resource, err := database.NewResource(
		db,
		config.Get("DB_DRIVER", "mysql"),
		table,
	)
	if err != nil {
		return nil, err
	}
	if softDeletes {
		resource.WithSoftDeletes()
	}
	return resource, nil
}

type DBResourceOptions struct {
	SoftDeletes bool
}

type DBIndexOptions struct {
	Searchable  []string
	Filterable  []string
	Sortable    []string
	SoftDeletes bool
}

func (ctx *Context) DBIndex(table string, configs ...DBIndexOptions) error {
	var config DBIndexOptions
	if len(configs) > 0 {
		config = configs[0]
	}

	resource, err := databaseResource(table, config.SoftDeletes)
	if err != nil {
		return err
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

func (ctx *Context) DBShow(table, id string, configs ...DBResourceOptions) error {
	var config DBResourceOptions
	if len(configs) > 0 {
		config = configs[0]
	}
	resource, err := databaseResource(table, config.SoftDeletes)
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

func (ctx *Context) DBStore(table string, values Map, configs ...DBResourceOptions) error {
	var config DBResourceOptions
	if len(configs) > 0 {
		config = configs[0]
	}
	resource, err := databaseResource(table, config.SoftDeletes)
	if err != nil {
		return err
	}

	item, err := resource.Store(map[string]any(values))
	if err != nil {
		return err
	}

	return ctx.Status(http.StatusCreated).JSON(item)
}

func (ctx *Context) DBUpdate(table, id string, values Map, configs ...DBResourceOptions) error {
	var config DBResourceOptions
	if len(configs) > 0 {
		config = configs[0]
	}
	resource, err := databaseResource(table, config.SoftDeletes)
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

func (ctx *Context) DBDestroy(table, id string, configs ...DBResourceOptions) error {
	var config DBResourceOptions
	if len(configs) > 0 {
		config = configs[0]
	}
	resource, err := databaseResource(table, config.SoftDeletes)
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
