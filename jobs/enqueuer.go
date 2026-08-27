package jobs

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/purpose-robot/blips-and-chitz/auth"
	"github.com/riverqueue/river"
)

type emailGateway interface {
	Send(context.Context, string, any, ...string) error
}

type Enqueuer struct {
	tx     pgx.Tx
	client *river.Client[pgx.Tx]
}

func NewEnqueuer(tx pgx.Tx, client *river.Client[pgx.Tx]) *Enqueuer {
	return &Enqueuer{
		tx:     tx,
		client: client,
	}
}

func (e *Enqueuer) EnqueueActivationEmail(ctx context.Context, email auth.TokenEmail) error {
	return e.enqueueTokenEmail(ctx, email, "activate_user.tmpl")
}

func (e *Enqueuer) EnqueuePasswordResetEmail(ctx context.Context, email auth.TokenEmail) error {
	return e.enqueueTokenEmail(ctx, email, "reset_password.tmpl")
}

func (e *Enqueuer) enqueueTokenEmail(ctx context.Context, email auth.TokenEmail, template string) error {
	args := TokenEmailArgs{
		Template:  template,
		Name:      email.Name,
		Plaintext: email.Plaintext,
		Recipient: email.Recipient,
	}

	_, err := e.client.InsertTx(ctx, e.tx, args, nil)
	if err != nil {
		return fmt.Errorf("jobs.enqueuer.emails: template %s: %w", template, err)
	}

	return nil
}
