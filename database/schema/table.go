package schema

type Table struct {
	Name    string
	Columns []*Column
}

func newTable(name string) *Table {
	return &Table{
		Name:    name,
		Columns: make([]*Column, 0),
	}
}

func (table *Table) addColumn(
	name string,
	columnType ColumnType,
) *Column {

	column := &Column{
		Name: name,
		Type: columnType,
	}

	table.Columns = append(
		table.Columns,
		column,
	)

	return column
}

func (table *Table) ID() *Column {
	return table.addColumn(
		"id",
		TypeID,
	)
}

func (table *Table) String(
	name string,
	length ...int,
) *Column {

	size := 255

	if len(length) > 0 {
		size = length[0]
	}

	column := table.addColumn(
		name,
		TypeString,
	)

	column.Length = size

	return column
}

func (table *Table) Text(name string) *Column {
	return table.addColumn(
		name,
		TypeText,
	)
}

func (table *Table) Integer(name string) *Column {
	return table.addColumn(
		name,
		TypeInteger,
	)
}

func (table *Table) BigInteger(name string) *Column {
	return table.addColumn(
		name,
		TypeBigInt,
	)
}

func (table *Table) Boolean(name string) *Column {
	return table.addColumn(
		name,
		TypeBoolean,
	)
}

func (table *Table) Decimal(
	name string,
	precision int,
	scale int,
) *Column {

	column := table.addColumn(
		name,
		TypeDecimal,
	)

	column.Precision = precision
	column.Scale = scale

	return column
}

func (table *Table) Timestamp(name string) *Column {
	return table.addColumn(
		name,
		TypeTimestamp,
	)
}

func (table *Table) Timestamps() {
	table.Timestamp("created_at").DefaultValue(Raw("CURRENT_TIMESTAMP"))
	table.Timestamp("updated_at").DefaultValue(Raw("CURRENT_TIMESTAMP"))
}

func (table *Table) SoftDeletes() {
	table.Timestamp("deleted_at").Nullable()
}

func (table *Table) DateTime(name string) *Column  { return table.addColumn(name, TypeDateTime) }
func (table *Table) JSON(name string) *Column      { return table.addColumn(name, TypeJSON) }
func (table *Table) UUID(name string) *Column      { return table.addColumn(name, TypeUUID) }
func (table *Table) ForeignID(name string) *Column { return table.addColumn(name, TypeBigInt) }
