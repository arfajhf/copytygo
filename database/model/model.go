package model

import (
	"context"
	"database/sql"

	"github.com/arfajhf/copytygo/v2/database/query"
)

type Model struct {
	DB     *sql.DB
	Driver string
	Table  string
}

func New(db *sql.DB, driver, table string) *Model {
	return &Model{DB: db, Driver: driver, Table: table}
}

func (m *Model) Query() *query.Builder {
	return query.Table(m.DB, m.Driver, m.Table)
}

func (m *Model) All(ctx context.Context) ([]map[string]any, error) {
	return m.Query().AllMaps(ctx)
}

func (m *Model) Find(ctx context.Context, id any) (map[string]any, bool, error) {
	return m.Query().WhereEq("id", id).FirstMap(ctx)
}

func (m *Model) Create(ctx context.Context, values map[string]any) (map[string]any, error) {
	id, err := m.Query().InsertID(ctx, values)
	if err != nil {
		return nil, err
	}

	item, found, err := m.Find(ctx, id)
	if err != nil {
		return nil, err
	}
	if !found {
		return map[string]any{"id": id}, nil
	}
	return item, nil
}

func (m *Model) Update(ctx context.Context, id any, values map[string]any) (map[string]any, bool, error) {
	result, err := m.Query().WhereEq("id", id).Update(ctx, values)
	if err != nil {
		return nil, false, err
	}
	if affected, rowsErr := result.RowsAffected(); rowsErr == nil && affected == 0 {
		return m.Find(ctx, id)
	}
	return m.Find(ctx, id)
}

func (m *Model) Delete(ctx context.Context, id any) (bool, error) {
	result, err := m.Query().WhereEq("id", id).Delete(ctx)
	if err != nil {
		return false, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return true, nil
	}
	return affected > 0, nil
}
