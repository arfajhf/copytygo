package dialect

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/arfajhf/copytygo/v4/database/schema"
)

type MySQL struct{}

func (MySQL) Name() string {
	return "mysql"
}

func (MySQL) Compile(
	blueprint *schema.Blueprint,
) (string, error) {

	switch blueprint.Operation {

	case schema.OperationCreate:
		return compileMySQLCreate(
			blueprint.Table,
		)

	case schema.OperationDrop:
		return fmt.Sprintf(
			"DROP TABLE `%s`;",
			blueprint.Table.Name,
		), nil

	default:
		return "", fmt.Errorf(
			"copytygo: unsupported schema operation %q",
			blueprint.Operation,
		)
	}
}

func compileMySQLCreate(
	table *schema.Table,
) (string, error) {

	columns := make(
		[]string,
		0,
		len(table.Columns),
	)

	for _, column := range table.Columns {

		compiled, err := compileMySQLColumn(
			column,
		)

		if err != nil {
			return "", err
		}

		columns = append(
			columns,
			compiled,
		)
	}

	return fmt.Sprintf(
		"CREATE TABLE `%s` (\n  %s\n);",
		table.Name,
		strings.Join(
			columns,
			",\n  ",
		),
	), nil
}

func compileMySQLColumn(
	column *schema.Column,
) (string, error) {

	var sqlType string

	switch column.Type {

	case schema.TypeID:
		sqlType = "BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY"

	case schema.TypeString:
		sqlType = fmt.Sprintf(
			"VARCHAR(%d)",
			column.Length,
		)

	case schema.TypeText:
		sqlType = "TEXT"

	case schema.TypeInteger:
		sqlType = "INT"

	case schema.TypeBigInt:
		sqlType = "BIGINT"

	case schema.TypeBoolean:
		sqlType = "BOOLEAN"

	case schema.TypeDecimal:
		sqlType = fmt.Sprintf(
			"DECIMAL(%d,%d)",
			column.Precision,
			column.Scale,
		)

	case schema.TypeTimestamp:
		sqlType = "TIMESTAMP"

	case schema.TypeDateTime:
		sqlType = "DATETIME"

	case schema.TypeJSON:
		sqlType = "JSON"

	case schema.TypeUUID:
		sqlType = "CHAR(36)"

	default:
		return "", fmt.Errorf(
			"copytygo: unsupported MySQL column type %q",
			column.Type,
		)
	}

	result := fmt.Sprintf(
		"`%s` %s",
		column.Name,
		sqlType,
	)

	if !column.IsNullable &&
		column.Type != schema.TypeID {

		result += " NOT NULL"
	}

	if column.IsNullable {
		result += " NULL"
	}

	if column.IsUnique {
		result += " UNIQUE"
	}

	if column.HasDefault {
		result += " DEFAULT " +
			mysqlDefault(column.Default)
	}

	return result, nil
}

func mysqlDefault(value any) string {

	switch typed := value.(type) {

	case schema.Expression:
		return string(typed)

	case string:
		return "'" +
			strings.ReplaceAll(
				typed,
				"'",
				"''",
			) +
			"'"

	case bool:
		if typed {
			return "TRUE"
		}

		return "FALSE"

	case int:
		return strconv.Itoa(typed)

	case int64:
		return strconv.FormatInt(
			typed,
			10,
		)

	case float64:
		return strconv.FormatFloat(
			typed,
			'f',
			-1,
			64,
		)

	case nil:
		return "NULL"

	default:
		return fmt.Sprintf(
			"'%v'",
			typed,
		)
	}
}
