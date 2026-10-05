package migration

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/arfajhf/copytygo/database/schema"
	"github.com/arfajhf/copytygo/database/schema/dialect"
)

type Migrator struct {
	db         *sql.DB
	driver     string
	repository *Repository
}

func NewMigrator(
	db *sql.DB,
	driver string,
) *Migrator {

	return &Migrator{
		db:         db,
		driver:     driver,
		repository: NewRepository(db, driver),
	}
}

func (migrator *Migrator) Migrate() error {
	ctx, cancel := context.WithTimeout(
		context.Background(),
		30*time.Second,
	)

	defer cancel()

	if err := migrator.repository.Ensure(ctx); err != nil {
		return err
	}

	batch, err :=
		migrator.repository.NextBatch(ctx)

	if err != nil {
		return fmt.Errorf(
			"copytygo: unable to determine migration batch: %w",
			err,
		)
	}

	migrations := All()

	executed := 0

	for _, item := range migrations {

		hasRun, err :=
			migrator.repository.Has(
				ctx,
				item.Name,
			)

		if err != nil {
			return fmt.Errorf(
				"copytygo: unable to check migration %q: %w",
				item.Name,
				err,
			)
		}

		if hasRun {
			fmt.Printf(
				"SKIP  %s\n",
				item.Name,
			)

			continue
		}

		blueprint := item.Up()

		if blueprint == nil {
			return fmt.Errorf(
				"copytygo: migration %q returned an empty blueprint",
				item.Name,
			)
		}

		query, err :=
			migrator.compile(blueprint)

		if err != nil {
			return err
		}

		fmt.Printf(
			"RUN   %s\n",
			item.Name,
		)

		if _, err := migrator.db.ExecContext(
			ctx,
			query,
		); err != nil {

			return fmt.Errorf(
				"copytygo: migration %q failed: %w",
				item.Name,
				err,
			)
		}

		id, err :=
			migrator.repository.NextID(ctx)

		if err != nil {
			return err
		}

		if err := migrator.repository.Log(
			ctx,
			id,
			item.Name,
			batch,
		); err != nil {

			return fmt.Errorf(
				"copytygo: unable to record migration %q: %w",
				item.Name,
				err,
			)
		}

		fmt.Printf(
			"DONE  %s\n",
			item.Name,
		)

		executed++
	}

	if executed == 0 {
		fmt.Println(
			"Nothing to migrate.",
		)

		return nil
	}

	fmt.Printf(
		"\n%d migration(s) completed. Batch %d.\n",
		executed,
		batch,
	)

	return nil
}

func (migrator *Migrator) compile(
	blueprint *schema.Blueprint,
) (string, error) {

	switch migrator.driver {

	case "mysql":

		compiler := dialect.MySQL{}

		return compiler.Compile(
			blueprint,
		)

	case "postgres":

		compiler := dialect.PostgreSQL{}

		return compiler.Compile(
			blueprint,
		)

	default:

		return "", fmt.Errorf(
			"copytygo: migration dialect %q is not supported",
			migrator.driver,
		)
	}
}

func (migrator *Migrator) Rollback() error {
	ctx, cancel := context.WithTimeout(
		context.Background(),
		30*time.Second,
	)

	defer cancel()

	if err := migrator.repository.Ensure(ctx); err != nil {
		return err
	}

	batch, err :=
		migrator.repository.LastBatch(ctx)

	if err != nil {
		return fmt.Errorf(
			"copytygo: unable to determine last migration batch: %w",
			err,
		)
	}

	if batch == 0 {
		fmt.Println(
			"Nothing to rollback.",
		)

		return nil
	}

	records, err :=
		migrator.repository.ByBatch(
			ctx,
			batch,
		)

	if err != nil {
		return fmt.Errorf(
			"copytygo: unable to read migration batch %d: %w",
			batch,
			err,
		)
	}

	for _, record := range records {

		item, exists := Find(
			record.Migration,
		)

		if !exists {
			return fmt.Errorf(
				"copytygo: migration %q is recorded in database but not registered",
				record.Migration,
			)
		}

		if item.Down == nil {
			return fmt.Errorf(
				"copytygo: migration %q does not define a Down operation",
				item.Name,
			)
		}

		blueprint := item.Down()

		if blueprint == nil {
			return fmt.Errorf(
				"copytygo: migration %q returned an empty Down blueprint",
				item.Name,
			)
		}

		query, err :=
			migrator.compile(
				blueprint,
			)

		if err != nil {
			return err
		}

		fmt.Printf(
			"ROLLBACK  %s\n",
			item.Name,
		)

		if _, err := migrator.db.ExecContext(
			ctx,
			query,
		); err != nil {

			return fmt.Errorf(
				"copytygo: rollback %q failed: %w",
				item.Name,
				err,
			)
		}

		if err :=
			migrator.repository.Delete(
				ctx,
				item.Name,
			); err != nil {

			return fmt.Errorf(
				"copytygo: unable to remove migration record %q: %w",
				item.Name,
				err,
			)
		}

		fmt.Printf(
			"DONE      %s\n",
			item.Name,
		)
	}

	fmt.Printf(
		"\nBatch %d rolled back.\n",
		batch,
	)

	return nil
}

func (migrator *Migrator) Status() error {
	ctx, cancel := context.WithTimeout(
		context.Background(),
		10*time.Second,
	)

	defer cancel()

	if err := migrator.repository.Ensure(ctx); err != nil {
		return err
	}

	records, err :=
		migrator.repository.AllRecords(ctx)

	if err != nil {
		return fmt.Errorf(
			"copytygo: unable to read migration status: %w",
			err,
		)
	}

	executed := make(
		map[string]Record,
	)

	for _, record := range records {
		executed[record.Migration] = record
	}

	fmt.Println()
	fmt.Println("Migration Status")
	fmt.Println("---------------------------------------------------------------")
	fmt.Printf(
		"%-8s %-8s %s\n",
		"STATUS",
		"BATCH",
		"MIGRATION",
	)
	fmt.Println("---------------------------------------------------------------")

	for _, item := range All() {

		record, exists :=
			executed[item.Name]

		if exists {
			fmt.Printf(
				"%-8s %-8d %s\n",
				"Ran",
				record.Batch,
				item.Name,
			)

			continue
		}

		fmt.Printf(
			"%-8s %-8s %s\n",
			"Pending",
			"-",
			item.Name,
		)
	}

	fmt.Println()

	return nil
}

func (migrator *Migrator) Reset() error {
	for {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		if err := migrator.repository.Ensure(ctx); err != nil {
			cancel()
			return err
		}
		batch, err := migrator.repository.LastBatch(ctx)
		cancel()
		if err != nil {
			return err
		}
		if batch == 0 {
			return nil
		}
		if err := migrator.Rollback(); err != nil {
			return err
		}
	}
}

func (migrator *Migrator) Fresh() error {
	if err := migrator.Reset(); err != nil {
		return err
	}
	return migrator.Migrate()
}
