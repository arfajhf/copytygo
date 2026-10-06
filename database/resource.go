package database

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"time"

	"github.com/arfajhf/copytygo/database/query"
)

var safeIdentifier = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

type Resource struct {
	DB     *sql.DB
	Driver string
	Table  string
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
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return query.Table(r.DB, r.Driver, r.Table).OrderBy("id", "ASC").AllMaps(ctx)
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

	affected, err := result.RowsAffected()
	if err == nil && affected == 0 {
		return nil, false, nil
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
