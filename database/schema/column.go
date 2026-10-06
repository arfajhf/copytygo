package schema

type ColumnType string

type Expression string

const (
	TypeID        ColumnType = "id"
	TypeString    ColumnType = "string"
	TypeText      ColumnType = "text"
	TypeInteger   ColumnType = "integer"
	TypeBigInt    ColumnType = "bigint"
	TypeBoolean   ColumnType = "boolean"
	TypeDecimal   ColumnType = "decimal"
	TypeTimestamp ColumnType = "timestamp"
	TypeDateTime  ColumnType = "datetime"
	TypeJSON      ColumnType = "json"
	TypeUUID      ColumnType = "uuid"
)

type Column struct {
	Name string
	Type ColumnType

	Length    int
	Precision int
	Scale     int

	IsNullable bool
	IsUnique   bool

	HasDefault bool
	Default    any
}

func (column *Column) Nullable() *Column {
	column.IsNullable = true
	return column
}

func (column *Column) Unique() *Column {
	column.IsUnique = true
	return column
}

func (column *Column) DefaultValue(value any) *Column {
	column.HasDefault = true
	column.Default = value

	return column
}


func Raw(value string) Expression {
	return Expression(value)
}
