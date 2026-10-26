//go:build integration

package inbox

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

func TestConsumer_EX05_TerminatesAfterFiveFailedDeliveries(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 70*time.Second)
	defer cancel()
	natsURL := os.Getenv("NATS_URL")
	if natsURL == "" {
		natsURL = nats.DefaultURL
	}
	connection, err := nats.Connect(natsURL, nats.Timeout(5*time.Second), nats.Name("mini-consumer-retry-test"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(connection.Close)
	js, err := jetstream.New(connection)
	if err != nil {
		t.Fatal(err)
	}
	if err := EnsureStream(ctx, js); err != nil {
		t.Fatal(err)
	}
	token := uuid.NewString()
	subject := "mini.test.retry." + token
	durable := "retry-test-" + token[:8]
	t.Cleanup(func() { _ = js.DeleteConsumer(context.Background(), "MINI", durable) })

	var deliveries atomic.Int32
	delivered := make(chan int32, 8)
	runner, err := Start(ctx, js, durable, subject, func(context.Context, []byte) error {
		count := deliveries.Add(1)
		delivered <- count
		return errors.New("intentional test handler failure")
	}, slog.New(slog.NewTextHandler(os.Stderr, nil)))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(runner.Stop)
	consumer, err := js.Consumer(ctx, "MINI", durable)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := js.Publish(ctx, subject, []byte(`{"test":true}`)); err != nil {
		t.Fatal(err)
	}

	for expected := int32(1); expected <= 5; expected++ {
		select {
		case got := <-delivered:
			if got != expected {
				t.Fatalf("delivery count = %d, want %d", got, expected)
			}
		case <-ctx.Done():
			t.Fatalf("received %d failed deliveries: %v", deliveries.Load(), ctx.Err())
		}
	}

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		info, err := consumer.Info(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if info.NumAckPending == 0 && info.NumPending == 0 {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if got := deliveries.Load(); got != 5 {
		t.Fatalf("handler deliveries = %d, want exactly 5", got)
	}
	info, err := consumer.Info(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if info.NumAckPending != 0 || info.NumPending != 0 {
		t.Fatal(fmt.Sprintf("terminal message remains pending: ack_pending=%d pending=%d", info.NumAckPending, info.NumPending))
	}
}
