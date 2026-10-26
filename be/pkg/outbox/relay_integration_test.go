//go:build integration

package outbox

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/trieu22676191/MiniProject/be/pkg/db"
	"github.com/trieu22676191/MiniProject/be/pkg/inbox"
	"github.com/trieu22676191/MiniProject/be/pkg/tenant"
)

func TestOutboxRelay_PublishesToJetStreamBeforeMarkingPublished(t *testing.T) {
	appURL := os.Getenv("MINI_TEST_APP_DATABASE_URL")
	relayURL := os.Getenv("MINI_TEST_RELAY_DATABASE_URL")
	if appURL == "" || relayURL == "" {
		t.Fatal("test app and relay database URLs are required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	appPool := openPool(t, ctx, appURL)
	relayPool := openPool(t, ctx, relayURL)
	tenantID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	eventID, err := uuid.NewV7()
	if err != nil {
		t.Fatal(err)
	}
	subject := "mini.test.relay." + eventID.String()
	payload := []byte(`{"event":"relay-test"}`)
	identityCtx := tenant.WithIdentity(ctx, tenant.Identity{TenantID: tenantID, UserID: "test"})
	if err := db.New(appPool).InTx(identityCtx, func(tx pgx.Tx) error {
		_, err := tx.Exec(identityCtx, `INSERT INTO mini_order.outbox (id, tenant_id, type, payload)
			VALUES ($1, $2, $3, $4::jsonb)`, eventID, tenantID, subject, payload)
		return err
	}); err != nil {
		t.Fatal(err)
	}

	natsURL := os.Getenv("NATS_URL")
	if natsURL == "" {
		natsURL = nats.DefaultURL
	}
	connection, err := nats.Connect(natsURL, nats.Timeout(5*time.Second), nats.Name("mini-outbox-test"))
	if err != nil {
		t.Fatal(err)
	}
	defer connection.Close()
	js, err := jetstream.New(connection)
	if err != nil {
		t.Fatal(err)
	}
	if err := inbox.EnsureStream(ctx, js); err != nil {
		t.Fatal(err)
	}
	if err := publishBatch(ctx, relayPool, unavailablePublisher{}, "mini_order"); err == nil {
		t.Fatal("relay succeeded while JetStream publisher was unavailable")
	}
	var pending bool
	if err := relayPool.QueryRow(ctx, `SELECT published_at IS NULL FROM mini_order.outbox WHERE id = $1`, eventID).Scan(&pending); err != nil {
		t.Fatal(err)
	}
	if !pending {
		t.Fatal("failed publish must leave the event pending for relay recovery")
	}

	durable := "relay-test-" + eventID.String()[:8]
	consumer, err := js.CreateOrUpdateConsumer(ctx, "MINI", jetstream.ConsumerConfig{
		Durable: durable, FilterSubject: subject,
		DeliverPolicy: jetstream.DeliverNewPolicy, AckPolicy: jetstream.AckExplicitPolicy,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = js.DeleteConsumer(context.Background(), "MINI", durable) })

	for attempts := 0; attempts < 5; attempts++ {
		if err := publishBatch(ctx, relayPool, js, "mini_order"); err != nil {
			t.Fatal(err)
		}
		var published bool
		if err := relayPool.QueryRow(ctx, `SELECT published_at IS NOT NULL FROM mini_order.outbox WHERE id = $1`, eventID).Scan(&published); err != nil {
			t.Fatal(err)
		}
		if published {
			break
		}
	}
	var published bool
	if err := relayPool.QueryRow(ctx, `SELECT published_at IS NOT NULL FROM mini_order.outbox WHERE id = $1`, eventID).Scan(&published); err != nil {
		t.Fatal(err)
	}
	if !published {
		t.Fatal("relay did not mark the target event as published")
	}

	batch, err := consumer.Fetch(1, jetstream.FetchMaxWait(5*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	received := false
	for message := range batch.Messages() {
		received = true
		metadata, err := message.Metadata()
		if err != nil {
			t.Fatal(err)
		}
		var receivedPayload struct {
			Event string `json:"event"`
		}
		if err := json.Unmarshal(message.Data(), &receivedPayload); err != nil {
			t.Fatal(err)
		}
		if receivedPayload.Event != "relay-test" {
			t.Fatalf("received payload = %s, want relay-test event", message.Data())
		}
		if metadata.NumDelivered != 1 {
			t.Fatalf("delivery count = %d, want 1", metadata.NumDelivered)
		}
		if err := message.Ack(); err != nil {
			t.Fatal(err)
		}
	}
	if err := batch.Error(); err != nil {
		t.Fatal(err)
	}
	if !received {
		t.Fatal(fmt.Sprintf("JetStream did not deliver subject %s", subject))
	}
}

type unavailablePublisher struct{}

func (unavailablePublisher) PublishMsg(context.Context, *nats.Msg, ...jetstream.PublishOpt) (*jetstream.PubAck, error) {
	return nil, errors.New("simulated JetStream outage")
}

var _ publisher = unavailablePublisher{}

func openPool(t *testing.T, ctx context.Context, url string) *pgxpool.Pool {
	t.Helper()
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	return pool
}
