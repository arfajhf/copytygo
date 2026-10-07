package database

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/arfajhf/copytygo/v4/database/drivers"
)

type TxFunc func(*sql.Tx) error

func Transaction(ctx context.Context, fn TxFunc) error {
	if fn == nil {
		return fmt.Errorf("copytygo database: transaction callback is required")
	}

	drivers.Register()
	db, err := Connect()
	if err != nil {
		return err
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("copytygo database: begin transaction: %w", err)
	}

	if err := fn(tx); err != nil {
		if rollbackErr := tx.Rollback(); rollbackErr != nil {
			return fmt.Errorf("copytygo database: transaction failed: %v; rollback failed: %w", err, rollbackErr)
		}
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("copytygo database: commit transaction: %w", err)
	}

	return nil
}
