package store

import (
	"context"
	"database/sql"

	integrationstore "github.com/bcp-technology-ug/lobster/gen/sqlc/integrations"
	planstore "github.com/bcp-technology-ug/lobster/gen/sqlc/plan"
	runstore "github.com/bcp-technology-ug/lobster/gen/sqlc/run"
	stackstore "github.com/bcp-technology-ug/lobster/gen/sqlc/stack"
)

// TxStore exposes domain query sets bound to a single SQL transaction.
type TxStore struct {
	tx *sql.Tx

	Run          *runstore.Queries
	Plan         *planstore.Queries
	Stack        *stackstore.Queries
	Integrations *integrationstore.Queries
}

// Tx returns the backing SQL transaction for advanced call sites.
func (t *TxStore) Tx() *sql.Tx {
	if t == nil {
		return nil
	}
	return t.tx
}

// WithTx executes fn inside a transaction and commits only on nil error.
func (s *Store) WithTx(ctx context.Context, fn func(*TxStore) error) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	txStore := &TxStore{
		tx:           tx,
		Run:          s.Run.WithTx(tx),
		Plan:         s.Plan.WithTx(tx),
		Stack:        s.Stack.WithTx(tx),
		Integrations: s.Integrations.WithTx(tx),
	}

	if err := fn(txStore); err != nil {
		_ = tx.Rollback()
		return err
	}

	return tx.Commit()
}
