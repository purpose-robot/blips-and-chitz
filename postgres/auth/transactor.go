package auth

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/purpose-robot/blips-and-chitz/auth"
	"github.com/purpose-robot/blips-and-chitz/jobs"
	"github.com/riverqueue/river"
)

type Transactor struct {
	conn   *pgxpool.Pool
	client *river.Client[pgx.Tx]
}

func NewTransactor(conn *pgxpool.Pool, client *river.Client[pgx.Tx]) *Transactor {
	return &Transactor{
		conn:   conn,
		client: client,
	}
}

func (t *Transactor) Run(ctx context.Context, fn func(tx auth.Stores) error) error {
	tx, err := t.conn.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	stores := auth.Stores{
		Jobs:        jobs.NewEnqueuer(tx, t.client),
		Users:       NewUserStore(tx),
		Tokens:      NewTokenStore(tx),
		Permissions: NewPermissionStore(tx),
	}

	if err := fn(stores); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}
