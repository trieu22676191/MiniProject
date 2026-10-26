package outbox

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

type Event struct {
	ID      string
	Subject string
	Payload []byte
}

func Run(ctx context.Context, pool *pgxpool.Pool, stream jetstream.Publisher, schema string, logger *slog.Logger) error {
	if schema != "mini_order" && schema != "mini_stock" {
		return fmt.Errorf("unsupported outbox schema %q", schema)
	}
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	for {
		if err := publishBatch(ctx, pool, stream, schema); err != nil && ctx.Err() == nil {
			logger.Error("publish outbox batch", "schema", schema, "error", err)
		}
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		}
	}
}

func publishBatch(ctx context.Context, pool *pgxpool.Pool, stream jetstream.Publisher, schema string) error {
	batchCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	tx, err := pool.Begin(batchCtx)
	if err != nil {
		return fmt.Errorf("begin outbox transaction: %w", err)
	}
	defer tx.Rollback(batchCtx)
	query := fmt.Sprintf(`SELECT id::text, type, payload
FROM %s.outbox
WHERE published_at IS NULL
ORDER BY id
LIMIT 100
FOR UPDATE SKIP LOCKED`, schema)
	rows, err := tx.Query(batchCtx, query)
	if err != nil {
		return fmt.Errorf("claim outbox events: %w", err)
	}
	events := make([]Event, 0, 100)
	for rows.Next() {
		var event Event
		if err := rows.Scan(&event.ID, &event.Subject, &event.Payload); err != nil {
			rows.Close()
			return fmt.Errorf("read outbox event: %w", err)
		}
		events = append(events, event)
	}
	rowsErr := rows.Err()
	rows.Close()
	if rowsErr != nil {
		return fmt.Errorf("iterate outbox events: %w", rowsErr)
	}
	for _, event := range events {
		publishCtx, publishCancel := context.WithTimeout(batchCtx, 5*time.Second)
		message := nats.NewMsg(event.Subject)
		message.Header.Set("Nats-Msg-Id", event.ID)
		message.Data = event.Payload
		_, err := stream.PublishMsg(publishCtx, message, jetstream.WithMsgID(event.ID))
		publishCancel()
		if err != nil {
			return fmt.Errorf("publish event %s: %w", event.ID, err)
		}
		if _, err := tx.Exec(batchCtx, fmt.Sprintf("UPDATE %s.outbox SET published_at = now() WHERE id = $1", schema), event.ID); err != nil {
			return fmt.Errorf("mark event %s published: %w", event.ID, err)
		}
	}
	if err := tx.Commit(batchCtx); err != nil {
		return fmt.Errorf("commit outbox transaction: %w", err)
	}
	return nil
}
