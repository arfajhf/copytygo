package migration

import (
	"context"
	"database/sql"
	"fmt"
)

const repositoryTable = "_copytygo_migrations"

type Repository struct {
	db     *sql.DB
	driver string
}

type Record struct {
	ID        int64
	Migration string
	Batch     int
}

func NewRepository(
	db *sql.DB,
	driver string,
) *Repository {

	return &Repository{
		db:     db,
		driver: driver,
	}
}

func (repository *Repository) Ensure(
	ctx context.Context,
) error {

	query := fmt.Sprintf(`
CREATE TABLE IF NOT EXISTS %s (
	id BIGINT NOT NULL,
	migration VARCHAR(255) NOT NULL UNIQUE,
	batch INTEGER NOT NULL,
	executed_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	PRIMARY KEY (id)
);`, repositoryTable)

	_, err := repository.db.ExecContext(
		ctx,
		query,
	)

	if err != nil {
		return fmt.Errorf(
			"copytygo: unable to create migration repository: %w",
			err,
		)
	}

	return nil
}

func (repository *Repository) Has(
	ctx context.Context,
	name string,
) (bool, error) {

	query := fmt.Sprintf(
		"SELECT COUNT(*) FROM %s WHERE migration = %s",
		repositoryTable,
		repository.placeholder(1),
	)

	var count int

	err := repository.db.QueryRowContext(
		ctx,
		query,
		name,
	).Scan(&count)

	if err != nil {
		return false, err
	}

	return count > 0, nil
}

func (repository *Repository) NextID(
	ctx context.Context,
) (int64, error) {

	query := fmt.Sprintf(
		"SELECT COALESCE(MAX(id), 0) + 1 FROM %s",
		repositoryTable,
	)

	var id int64

	err := repository.db.QueryRowContext(
		ctx,
		query,
	).Scan(&id)

	return id, err
}

func (repository *Repository) NextBatch(
	ctx context.Context,
) (int, error) {

	query := fmt.Sprintf(
		"SELECT COALESCE(MAX(batch), 0) + 1 FROM %s",
		repositoryTable,
	)

	var batch int

	err := repository.db.QueryRowContext(
		ctx,
		query,
	).Scan(&batch)

	return batch, err
}

func (repository *Repository) Log(
	ctx context.Context,
	id int64,
	name string,
	batch int,
) error {

	query := fmt.Sprintf(
		"INSERT INTO %s (id, migration, batch) VALUES (%s, %s, %s)",
		repositoryTable,
		repository.placeholder(1),
		repository.placeholder(2),
		repository.placeholder(3),
	)

	_, err := repository.db.ExecContext(
		ctx,
		query,
		id,
		name,
		batch,
	)

	return err
}

func (repository *Repository) placeholder(
	position int,
) string {

	if repository.driver == "postgres" {
		return fmt.Sprintf(
			"$%d",
			position,
		)
	}

	return "?"
}

func (repository *Repository) LastBatch(
	ctx context.Context,
) (int, error) {

	query := fmt.Sprintf(
		"SELECT COALESCE(MAX(batch), 0) FROM %s",
		repositoryTable,
	)

	var batch int

	err := repository.db.QueryRowContext(
		ctx,
		query,
	).Scan(&batch)

	return batch, err
}

func (repository *Repository) ByBatch(
	ctx context.Context,
	batch int,
) ([]Record, error) {

	query := fmt.Sprintf(
		`SELECT id, migration, batch
		 FROM %s
		 WHERE batch = %s
		 ORDER BY id DESC`,
		repositoryTable,
		repository.placeholder(1),
	)

	rows, err := repository.db.QueryContext(
		ctx,
		query,
		batch,
	)

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	records := make([]Record, 0)

	for rows.Next() {
		var record Record

		if err := rows.Scan(
			&record.ID,
			&record.Migration,
			&record.Batch,
		); err != nil {

			return nil, err
		}

		records = append(
			records,
			record,
		)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return records, nil
}

func (repository *Repository) Delete(
	ctx context.Context,
	name string,
) error {

	query := fmt.Sprintf(
		"DELETE FROM %s WHERE migration = %s",
		repositoryTable,
		repository.placeholder(1),
	)

	_, err := repository.db.ExecContext(
		ctx,
		query,
		name,
	)

	return err
}

func (repository *Repository) AllRecords(
	ctx context.Context,
) ([]Record, error) {

	query := fmt.Sprintf(
		`SELECT id, migration, batch
		 FROM %s
		 ORDER BY id ASC`,
		repositoryTable,
	)

	rows, err := repository.db.QueryContext(
		ctx,
		query,
	)

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	records := make([]Record, 0)

	for rows.Next() {
		var record Record

		if err := rows.Scan(
			&record.ID,
			&record.Migration,
			&record.Batch,
		); err != nil {

			return nil, err
		}

		records = append(
			records,
			record,
		)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return records, nil
}

func (repository *Repository) Clear(ctx context.Context) error {
	_, err := repository.db.ExecContext(ctx, fmt.Sprintf("DELETE FROM %s", repositoryTable))
	return err
}
