package schema

type Operation string

const (
	OperationCreate Operation = "create"
	OperationDrop   Operation = "drop"
)

type Blueprint struct {
	Operation Operation
	Table     *Table
}

func Create(
	name string,
	callback func(table *Table),
) *Blueprint {

	table := newTable(name)

	callback(table)

	return &Blueprint{
		Operation: OperationCreate,
		Table:     table,
	}
}

func Drop(name string) *Blueprint {
	return &Blueprint{
		Operation: OperationDrop,

		Table: &Table{
			Name: name,
		},
	}
}
