package model

import (
	"context"
	"database/sql"
	"github.com/arfajhf/copytygo/database/query"
)

type Model struct {
	DB     *sql.DB
	Driver string
	Table  string
}

func New(db *sql.DB, driver, table string) *Model {
	return &Model{DB: db, Driver: driver, Table: table}
}
func (m *Model) Query() *query.Builder { return query.Table(m.DB, m.Driver, m.Table) }
func (m *Model) Find(ctx context.Context, id any) (*sql.Rows, error) {
	return m.Query().WhereEq("id", id).Limit(1).Rows(ctx)
}
func (m *Model) Create(ctx context.Context, values map[string]any) (sql.Result, error) {
	return m.Query().Insert(ctx, values)
}
