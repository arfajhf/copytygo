package query

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"sort"
	"strings"
)

var identifierPattern = regexp.MustCompile("^[A-Za-z_][A-Za-z0-9_]*$")

type Builder struct {
	db            *sql.DB
	driver, table string
	columns       []string
	where         []string
	args          []any
	order         string
	limit         int
	offset        int
	err           error
}

func Table(db *sql.DB, driver, table string) *Builder {
	builder := &Builder{
		db:      db,
		driver:  driver,
		table:   table,
		columns: []string{"*"},
	}

	if db == nil {
		builder.err = fmt.Errorf("copytygo: query database connection is nil")
	} else if !validIdentifier(table) {
		builder.err = fmt.Errorf("copytygo: invalid table name %q", table)
	}

	return builder
}

func (b *Builder) Select(columns ...string) *Builder {
	if b.err != nil {
		return b
	}

	for _, column := range columns {
		if column != "*" && !validIdentifier(column) {
			b.err = fmt.Errorf("copytygo: invalid column name %q", column)
			return b
		}
	}

	if len(columns) > 0 {
		b.columns = columns
	}

	return b
}

func (b *Builder) Where(column, operator string, value any) *Builder {
	if b.err != nil {
		return b
	}

	if !validIdentifier(column) {
		b.err = fmt.Errorf("copytygo: invalid column name %q", column)
		return b
	}

	operator = strings.ToUpper(strings.TrimSpace(operator))
	if !validOperator(operator) {
		b.err = fmt.Errorf("copytygo: unsupported query operator %q", operator)
		return b
	}

	b.args = append(b.args, value)
	b.where = append(
		b.where,
		fmt.Sprintf(
			"%s %s %s",
			column,
			operator,
			b.placeholder(len(b.args)),
		),
	)

	return b
}

func (b *Builder) WhereEq(column string, value any) *Builder {
	return b.Where(column, "=", value)
}

func (b *Builder) WhereNull(column string) *Builder {
	if b.err != nil {
		return b
	}
	if !validIdentifier(column) {
		b.err = fmt.Errorf("copytygo: invalid column name %q", column)
		return b
	}
	b.where = append(b.where, column+" IS NULL")
	return b
}

func (b *Builder) WhereNotNull(column string) *Builder {
	if b.err != nil {
		return b
	}
	if !validIdentifier(column) {
		b.err = fmt.Errorf("copytygo: invalid column name %q", column)
		return b
	}
	b.where = append(b.where, column+" IS NOT NULL")
	return b
}

func (b *Builder) WhereAnyLike(columns []string, value string) *Builder {
	if b.err != nil {
		return b
	}
	if len(columns) == 0 || value == "" {
		return b
	}

	parts := make([]string, 0, len(columns))
	for _, column := range columns {
		if !validIdentifier(column) {
			b.err = fmt.Errorf("copytygo: invalid search column %q", column)
			return b
		}
		b.args = append(b.args, "%"+value+"%")
		parts = append(parts, fmt.Sprintf("%s LIKE %s", column, b.placeholder(len(b.args))))
	}

	b.where = append(b.where, "("+strings.Join(parts, " OR ")+")")
	return b
}

func (b *Builder) OrderBy(column, direction string) *Builder {
	if b.err != nil {
		return b
	}

	if !validIdentifier(column) {
		b.err = fmt.Errorf("copytygo: invalid order column %q", column)
		return b
	}

	direction = strings.ToUpper(strings.TrimSpace(direction))
	if direction != "DESC" {
		direction = "ASC"
	}

	b.order = column + " " + direction
	return b
}

func (b *Builder) Limit(n int) *Builder {
	if n > 0 {
		b.limit = n
	}
	return b
}

func (b *Builder) Offset(n int) *Builder {
	if n > 0 {
		b.offset = n
	}
	return b
}

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

	return q, append([]any{}, b.args...)
}

func (b *Builder) Rows(ctx context.Context) (*sql.Rows, error) {
	if b.err != nil {
		return nil, b.err
	}

	q, args := b.SQL()
	return b.db.QueryContext(ctx, q, args...)
}

func (b *Builder) Count(ctx context.Context) (int64, error) {
	if b.err != nil {
		return 0, b.err
	}

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
	if err := b.validateWriteValues(values); err != nil {
		return nil, err
	}

	cols, args, marks := b.insertParts(values)
	q := fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES (%s)",
		b.table,
		strings.Join(cols, ", "),
		strings.Join(marks, ", "),
	)

	return b.db.ExecContext(ctx, q, args...)
}

func (b *Builder) InsertID(ctx context.Context, values map[string]any) (int64, error) {
	if err := b.validateWriteValues(values); err != nil {
		return 0, err
	}

	cols, args, marks := b.insertParts(values)
	q := fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES (%s)",
		b.table,
		strings.Join(cols, ", "),
		strings.Join(marks, ", "),
	)

	if b.driver == "postgres" {
		q += " RETURNING id"

		var id int64
		if err := b.db.QueryRowContext(ctx, q, args...).Scan(&id); err != nil {
			return 0, err
		}

		return id, nil
	}

	result, err := b.db.ExecContext(ctx, q, args...)
	if err != nil {
		return 0, err
	}

	return result.LastInsertId()
}

func (b *Builder) insertParts(values map[string]any) ([]string, []any, []string) {
	cols := make([]string, 0, len(values))
	for column := range values {
		cols = append(cols, column)
	}
	sort.Strings(cols)

	args := make([]any, 0, len(cols))
	marks := make([]string, 0, len(cols))

	for i, column := range cols {
		args = append(args, values[column])
		marks = append(marks, b.placeholder(i+1))
	}

	return cols, args, marks
}

func (b *Builder) Update(ctx context.Context, values map[string]any) (sql.Result, error) {
	if err := b.validateWriteValues(values); err != nil {
		return nil, err
	}

	cols := make([]string, 0, len(values))
	for column := range values {
		cols = append(cols, column)
	}
	sort.Strings(cols)

	sets := make([]string, 0, len(cols))
	args := make([]any, 0, len(cols)+len(b.args))

	for i, column := range cols {
		args = append(args, values[column])
		sets = append(
			sets,
			fmt.Sprintf("%s = %s", column, b.placeholder(i+1)),
		)
	}

	where := make([]string, len(b.where))
	copy(where, b.where)

	if b.driver == "postgres" {
		for i, clause := range where {
			for n := len(b.args); n >= 1; n-- {
				clause = strings.ReplaceAll(
					clause,
					fmt.Sprintf("$%d", n),
					fmt.Sprintf("$%d", n+len(cols)),
				)
			}
			where[i] = clause
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
	if b.err != nil {
		return nil, b.err
	}

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

func (b *Builder) AllMaps(ctx context.Context) ([]map[string]any, error) {
	rows, err := b.Rows(ctx)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanRows(rows)
}

func (b *Builder) FirstMap(ctx context.Context) (map[string]any, bool, error) {
	clone := *b
	clone.limit = 1

	rows, err := clone.Rows(ctx)
	if err != nil {
		return nil, false, err
	}
	defer rows.Close()

	items, err := scanRows(rows)
	if err != nil {
		return nil, false, err
	}
	if len(items) == 0 {
		return nil, false, nil
	}

	return items[0], true, nil
}

func scanRows(rows *sql.Rows) ([]map[string]any, error) {
	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	items := make([]map[string]any, 0)

	for rows.Next() {
		values := make([]any, len(columns))
		pointers := make([]any, len(columns))

		for i := range values {
			pointers[i] = &values[i]
		}

		if err := rows.Scan(pointers...); err != nil {
			return nil, err
		}

		item := make(map[string]any, len(columns))
		for i, column := range columns {
			value := values[i]
			if raw, ok := value.([]byte); ok {
				value = string(raw)
			}
			item[column] = value
		}

		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return items, nil
}

func (b *Builder) validateWriteValues(values map[string]any) error {
	if b.err != nil {
		return b.err
	}
	if len(values) == 0 {
		return fmt.Errorf("copytygo: write values cannot be empty")
	}

	for column := range values {
		if !validIdentifier(column) {
			return fmt.Errorf("copytygo: invalid column name %q", column)
		}
	}

	return nil
}

func validIdentifier(value string) bool {
	return identifierPattern.MatchString(value)
}

func validOperator(operator string) bool {
	switch operator {
	case "=", "!=", "<>", ">", ">=", "<", "<=", "LIKE":
		return true
	default:
		return false
	}
}
