package query

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

type Builder struct {
	db            *sql.DB
	driver, table string
	columns       []string
	where         []string
	args          []any
	order         string
	limit         int
	offset        int
}

func Table(db *sql.DB, driver, table string) *Builder {
	return &Builder{db: db, driver: driver, table: table, columns: []string{"*"}}
}
func (b *Builder) Select(columns ...string) *Builder {
	if len(columns) > 0 {
		b.columns = columns
	}
	return b
}
func (b *Builder) Where(column, operator string, value any) *Builder {
	b.args = append(b.args, value)
	b.where = append(b.where, fmt.Sprintf("%s %s %s", column, operator, b.placeholder(len(b.args))))
	return b
}
func (b *Builder) WhereEq(column string, value any) *Builder { return b.Where(column, "=", value) }
func (b *Builder) OrderBy(column, direction string) *Builder {
	direction = strings.ToUpper(direction)
	if direction != "DESC" {
		direction = "ASC"
	}
	b.order = column + " " + direction
	return b
}
func (b *Builder) Limit(n int) *Builder  { b.limit = n; return b }
func (b *Builder) Offset(n int) *Builder { b.offset = n; return b }
func (b *Builder) SQL() (string, []any) {
	q := "SELECT " + strings.Join(b.columns, ", ") + " FROM " + b.table
	if len(b.where) > 0 {
		q += " WHERE " + strings.Join(b.where, " AND ")
	}
	if b.order != "" {
		q += " ORDER BY " + b.order
	}
	if b.limit > 0 {
		q += fmt.Sprintf(" LIMIT %d", b.limit)
	}
	if b.offset > 0 {
		q += fmt.Sprintf(" OFFSET %d", b.offset)
	}
	return q, b.args
}
func (b *Builder) Rows(ctx context.Context) (*sql.Rows, error) {
	q, args := b.SQL()
	return b.db.QueryContext(ctx, q, args...)
}
func (b *Builder) Count(ctx context.Context) (int64, error) {
	clone := *b
	clone.columns = []string{"COUNT(*)"}
	clone.order = ""
	clone.limit = 0
	clone.offset = 0
	q, args := clone.SQL()
	var n int64
	err := b.db.QueryRowContext(ctx, q, args...).Scan(&n)
	return n, err
}
func (b *Builder) Insert(ctx context.Context, values map[string]any) (sql.Result, error) {
	cols := make([]string, 0, len(values))
	args := make([]any, 0, len(values))
	marks := make([]string, 0, len(values))
	for k, v := range values {
		cols = append(cols, k)
		args = append(args, v)
	} // deterministic order
	for i := 0; i < len(cols); i++ {
		for j := i + 1; j < len(cols); j++ {
			if cols[j] < cols[i] {
				cols[i], cols[j] = cols[j], cols[i]
			}
		}
	}
	args = args[:0]
	for i, k := range cols {
		args = append(args, values[k])
		marks = append(marks, b.placeholder(i+1))
	}
	q := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)", b.table, strings.Join(cols, ", "), strings.Join(marks, ", "))
	return b.db.ExecContext(ctx, q, args...)
}
func (b *Builder) Update(ctx context.Context, values map[string]any) (sql.Result, error) {
	cols := make([]string, 0, len(values))
	for k := range values {
		cols = append(cols, k)
	}
	for i := 0; i < len(cols); i++ {
		for j := i + 1; j < len(cols); j++ {
			if cols[j] < cols[i] {
				cols[i], cols[j] = cols[j], cols[i]
			}
		}
	}
	sets := make([]string, 0, len(cols))
	args := make([]any, 0, len(cols)+len(b.args))
	for i, k := range cols {
		args = append(args, values[k])
		sets = append(sets, fmt.Sprintf("%s = %s", k, b.placeholder(i+1)))
	}
	where := make([]string, len(b.where))
	for i, w := range b.where {
		where[i] = w
		if b.driver == "postgres" {
			for n := len(b.args); n >= 1; n-- {
				w = strings.ReplaceAll(w, fmt.Sprintf("$%d", n), fmt.Sprintf("$%d", n+len(cols)))
			}
			where[i] = w
		}
	}
	args = append(args, b.args...)
	q := "UPDATE " + b.table + " SET " + strings.Join(sets, ", ")
	if len(where) > 0 {
		q += " WHERE " + strings.Join(where, " AND ")
	}
	return b.db.ExecContext(ctx, q, args...)
}
func (b *Builder) Delete(ctx context.Context) (sql.Result, error) {
	q := "DELETE FROM " + b.table
	if len(b.where) > 0 {
		q += " WHERE " + strings.Join(b.where, " AND ")
	}
	return b.db.ExecContext(ctx, q, b.args...)
}
func (b *Builder) placeholder(n int) string {
	if b.driver == "postgres" {
		return fmt.Sprintf("$%d", n)
	}
	return "?"
}
