package model

import "context"

func (m *Model) Related(table string) *Model {
	if m == nil {
		return &Model{}
	}
	return New(m.DB, m.Driver, table)
}

func (m *Model) HasMany(
	ctx context.Context,
	relatedTable string,
	foreignKey string,
	localValue any,
) ([]map[string]any, error) {
	return m.Related(relatedTable).
		Query().
		WhereEq(foreignKey, localValue).
		AllMaps(ctx)
}

func (m *Model) HasOne(
	ctx context.Context,
	relatedTable string,
	foreignKey string,
	localValue any,
) (map[string]any, bool, error) {
	return m.Related(relatedTable).
		Query().
		WhereEq(foreignKey, localValue).
		FirstMap(ctx)
}

func (m *Model) BelongsTo(
	ctx context.Context,
	relatedTable string,
	foreignValue any,
	ownerKey ...string,
) (map[string]any, bool, error) {
	key := "id"
	if len(ownerKey) > 0 && ownerKey[0] != "" {
		key = ownerKey[0]
	}

	return m.Related(relatedTable).
		Query().
		WhereEq(key, foreignValue).
		FirstMap(ctx)
}
