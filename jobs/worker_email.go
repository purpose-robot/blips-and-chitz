package jobs

import (
	"context"
	"fmt"

	"github.com/riverqueue/river"
)

type TokenEmailArgs struct {
	Template  string `json:"template"`
	Name      string `json:"name"`
	Plaintext string `json:"plaintext"`
	Recipient string `json:"recipient"`
}

func (TokenEmailArgs) Kind() string {
	return "emails.token"
}

func (TokenEmailArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{
		MaxAttempts: 5,
	}
}

type TokenEmailWorker struct {
	river.WorkerDefaults[TokenEmailArgs]
	mailer mailerGateway
}

func NewTokenEmailWorker(mailer mailerGateway) *TokenEmailWorker {
	return &TokenEmailWorker{
		mailer: mailer,
	}
}

func (w *TokenEmailWorker) Work(ctx context.Context, job *river.Job[TokenEmailArgs]) error {
	data := map[string]any{
		"name":      job.Args.Name,
		"plaintext": job.Args.Plaintext,
	}

	err := w.mailer.Send(ctx, job.Args.Recipient, data, job.Args.Template)
	if err != nil {
		return fmt.Errorf("jobs.emails: %w", err)
	}

	return nil
}
