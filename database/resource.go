package database

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/arfajhf/copytygo/v2/database/query"
)

var safeIdentifier = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

type Resource struct {
	DB     *sql.DB
	Driver string
	Table  string
}

type ResourceListOptions struct {
	Page          int
	PerPage       int
	Search        string
	SearchColumns []string
	Sort          string
	Order         string
	Filters       map[string]string
	Filterable    []string
	Sortable      []string
}

type ResourcePage struct {
	Data       []map[string]any `json:"data"`
	Page       int              `json:"page"`
	PerPage    int              `json:"per_page"`
	Total      int64            `json:"total"`
	TotalPages int              `json:"total_pages"`
}

func NewResource(db *sql.DB, driver, table string) (*Resource, error) {
	if db == nil {
		return nil, fmt.Errorf("copytygo: database connection is nil")
	}
	if !safeIdentifier.MatchString(table) {
		return nil, fmt.Errorf("copytygo: invalid table name %q", table)
	}
	return &Resource{DB: db, Driver: driver, Table: table}, nil
}

func (r *Resource) Index() ([]map[string]any, error) {
	page, err := r.List(ResourceListOptions{Page: 1, PerPage: 100})
	if err != nil {
		return nil, err
	}
	return page.Data, nil
}

func (r *Resource) List(options ResourceListOptions) (ResourcePage, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	page := options.Page
	if page < 1 {
		page = 1
	}
	perPage := options.PerPage
	if perPage < 1 {
		perPage = 20
	}
	if perPage > 100 {
		perPage = 100
	}

	builder := query.Table(r.DB, r.Driver, r.Table)

	if options.Search != "" && len(options.SearchColumns) > 0 {
		builder.WhereAnyLike(options.SearchColumns, options.Search)
	}

	filterable := stringSet(options.Filterable)
	for field, value := range options.Filters {
		if value == "" || !filterable[field] {
			continue
		}
		builder.WhereEq(field, value)
	}

	sortField := options.Sort
	sortable := stringSet(options.Sortable)
	if sortField == "" || !sortable[sortField] {
		sortField = "id"
	}

	order := strings.ToUpper(strings.TrimSpace(options.Order))
	if order != "DESC" {
		order = "ASC"
	}

	total, err := builder.Count(ctx)
	if err != nil {
		return ResourcePage{}, err
	}

	items, err := builder.
		OrderBy(sortField, order).
		Limit(perPage).
		Offset((page - 1) * perPage).
		AllMaps(ctx)
	if err != nil {
		return ResourcePage{}, err
	}

	totalPages := 0
	if total > 0 {
		totalPages = int((total + int64(perPage) - 1) / int64(perPage))
	}

	return ResourcePage{
		Data: items,
		Page: page,
		PerPage: perPage,
		Total: total,
		TotalPages: totalPages,
	}, nil
}

func stringSet(values []string) map[string]bool {
	set := make(map[string]bool, len(values))
	for _, value := range values {
		if safeIdentifier.MatchString(value) {
			set[value] = true
		}
	}
	return set
}

func (r *Resource) Show(id any) (map[string]any, bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return query.Table(r.DB, r.Driver, r.Table).WhereEq("id", id).FirstMap(ctx)
}

func (r *Resource) Store(values map[string]any) (map[string]any, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	id, err := query.Table(r.DB, r.Driver, r.Table).InsertID(ctx, values)
	if err != nil {
		return nil, err
	}

	item, ok, err := query.Table(r.DB, r.Driver, r.Table).WhereEq("id", id).FirstMap(ctx)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, fmt.Errorf("copytygo: inserted row %d could not be reloaded", id)
	}
	return item, nil
}

func (r *Resource) Update(id any, values map[string]any) (map[string]any, bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, err := query.Table(r.DB, r.Driver, r.Table).WhereEq("id", id).Update(ctx, values)
	if err != nil {
		return nil, false, err
	}

	if affected, rowsErr := result.RowsAffected(); rowsErr == nil && affected == 0 {
		item, ok, findErr := query.Table(r.DB, r.Driver, r.Table).WhereEq("id", id).FirstMap(ctx)
		return item, ok, findErr
	}

	item, ok, err := query.Table(r.DB, r.Driver, r.Table).WhereEq("id", id).FirstMap(ctx)
	return item, ok, err
}

func (r *Resource) Destroy(id any) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, err := query.Table(r.DB, r.Driver, r.Table).WhereEq("id", id).Delete(ctx)
	if err != nil {
		return false, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return true, nil
	}
	return affected > 0, nil
}
