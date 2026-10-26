//go:build integration

package repo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/trieu22676191/MiniProject/be/pkg/db"
	"github.com/trieu22676191/MiniProject/be/pkg/events"
	"github.com/trieu22676191/MiniProject/be/pkg/inbox"
	"github.com/trieu22676191/MiniProject/be/pkg/outbox"
	"github.com/trieu22676191/MiniProject/be/pkg/tenant"
	"github.com/trieu22676191/MiniProject/be/services/stock/internal/adapter/consumer"
	"github.com/trieu22676191/MiniProject/be/services/stock/internal/domain"
	"github.com/trieu22676191/MiniProject/be/services/stock/internal/usecase"
)

var reservationTenant = uuid.MustParse("11111111-1111-1111-1111-111111111111")

func TestReservation_AC05_ApprovedOrderReservesAllItems(t *testing.T) {
	database, ctx := reservationTestDatabase(t)
	sku := uniqueReservationSKU(t, "AC05")
	insertReservationStock(t, database, ctx, sku, 4)
	orderID := uuid.Must(uuid.NewV7())
	approved := approvedEvent(t, orderID, map[string]int64{sku: 2})
	if err := reservationHandler(database).Handle(ctx, approved); err != nil {
		t.Fatal(err)
	}
	if got := readReserved(t, database, ctx, sku); got != 2 {
		t.Fatalf("reserved = %d, want 2", got)
	}
	var held events.ReservationHeld
	decodeStockResult(t, database, ctx, orderID, "mini.stk.reservation.held", &held)
	if held.OrderID != orderID.String() || held.ReservationID == "" {
		t.Fatalf("held event = %+v, want order ID and reservation ID", held)
	}
}

func TestReservation_AC06_ShortageDoesNotPartiallyHold(t *testing.T) {
	database, ctx := reservationTestDatabase(t)
	availableSKU, shortSKU := uniqueReservationSKU(t, "AC06A"), uniqueReservationSKU(t, "AC06B")
	insertReservationStock(t, database, ctx, availableSKU, 5)
	insertReservationStock(t, database, ctx, shortSKU, 1)
	orderID := uuid.Must(uuid.NewV7())
	approved := approvedEvent(t, orderID, map[string]int64{availableSKU: 2, shortSKU: 2})
	if err := reservationHandler(database).Handle(ctx, approved); err != nil {
		t.Fatal(err)
	}
	for _, sku := range []string{availableSKU, shortSKU} {
		if got := readReserved(t, database, ctx, sku); got != 0 {
			t.Fatalf("SKU %s reserved = %d, want 0", sku, got)
		}
	}
	var failed events.ReservationFailed
	decodeStockResult(t, database, ctx, orderID, "mini.stk.reservation.failed", &failed)
	if len(failed.ShortSKUs) != 1 || failed.ShortSKUs[0] != shortSKU {
		t.Fatalf("short SKUs = %v, want [%s]", failed.ShortSKUs, shortSKU)
	}
}

func TestReservation_AC07_DuplicateApprovedEventReservesOnce(t *testing.T) {
	database, ctx := reservationTestDatabase(t)
	sku := uniqueReservationSKU(t, "AC07")
	insertReservationStock(t, database, ctx, sku, 3)
	orderID := uuid.Must(uuid.NewV7())
	approved := approvedEvent(t, orderID, map[string]int64{sku: 1})
	handler := reservationHandler(database)
	for range 2 {
		if err := handler.Handle(ctx, approved); err != nil {
			t.Fatal(err)
		}
	}
	if got := readReserved(t, database, ctx, sku); got != 1 {
		t.Fatalf("reserved after duplicate delivery = %d, want 1", got)
	}
	if got := readReservationCount(t, database, ctx, orderID); got != 1 {
		t.Fatalf("reservation count = %d, want 1", got)
	}
	if got := readStockResultCount(t, database, ctx, orderID); got != 1 {
		t.Fatalf("reservation result event count = %d, want 1", got)
	}
}

func TestReservation_EX01_CancellationReleasesStockOnce(t *testing.T) {
	database, ctx := reservationTestDatabase(t)
	sku, secondSKU := uniqueReservationSKU(t, "EX01A"), uniqueReservationSKU(t, "EX01B")
	insertReservationStock(t, database, ctx, sku, 5)
	insertReservationStock(t, database, ctx, secondSKU, 4)
	orderID := uuid.Must(uuid.NewV7())
	if err := reservationHandler(database).Handle(ctx, approvedEvent(t, orderID, map[string]int64{sku: 3, secondSKU: 2})); err != nil {
		t.Fatal(err)
	}
	if got := readReserved(t, database, ctx, sku); got != 3 {
		t.Fatalf("reserved before cancellation = %d, want 3", got)
	}
	if got := readReserved(t, database, ctx, secondSKU); got != 2 {
		t.Fatalf("second SKU reserved before cancellation = %d, want 2", got)
	}

	payload := cancelledEvent(t, orderID)
	repository := NewReservations(db.New(database))
	cancelHandler := consumer.NewOrderCancelled(usecase.NewReservations(repository))
	var envelope events.Envelope
	if err := json.Unmarshal(payload, &envelope); err != nil {
		t.Fatal(err)
	}
	if err := db.New(database).InTx(ctx, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `INSERT INTO mini_order.outbox (id, tenant_id, type, payload)
			VALUES ($1, $2, $3, $4::jsonb)`, envelope.ID, reservationTenant, envelope.Type, payload)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	natsURL := os.Getenv("NATS_URL")
	if natsURL == "" {
		natsURL = nats.DefaultURL
	}
	connection, err := nats.Connect(natsURL, nats.Timeout(5*time.Second), nats.Name("mini-stock-cancel-integration"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(connection.Close)
	js, err := jetstream.New(connection)
	if err != nil {
		t.Fatal(err)
	}
	if err := inbox.EnsureStream(ctx, js); err != nil {
		t.Fatal(err)
	}
	durable := "cancel-stock-" + orderID.String()[:8]
	runner, err := inbox.Start(ctx, js, durable, envelope.Type, cancelHandler.Handle, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(runner.Stop)
	relayURL := os.Getenv("MINI_TEST_RELAY_DATABASE_URL")
	if relayURL == "" {
		t.Fatal("MINI_TEST_RELAY_DATABASE_URL is required")
	}
	relayPool, err := pgxpool.New(ctx, relayURL)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(relayPool.Close)
	relayCtx, stopRelay := context.WithCancel(ctx)
	relayDone := make(chan struct{})
	t.Cleanup(func() {
		stopRelay()
		select {
		case <-relayDone:
		case <-time.After(5 * time.Second):
			t.Error("order outbox relay did not stop")
		}
	})
	go func() {
		defer close(relayDone)
		_ = outbox.Run(relayCtx, relayPool, js, "mini_order", slog.New(slog.NewTextHandler(io.Discard, nil)))
	}()
	waitForStockRelease(t, database, ctx, sku, secondSKU)
	if err := cancelHandler.Handle(ctx, payload); err != nil {
		t.Fatal(err)
	}
	if err := cancelHandler.Handle(ctx, cancelledEvent(t, orderID)); err != nil {
		t.Fatal(err)
	}
	for _, stockSKU := range []string{sku, secondSKU} {
		if got := readReserved(t, database, ctx, stockSKU); got != 0 {
			t.Fatalf("reserved for %s after duplicate cancellation = %d, want 0", stockSKU, got)
		}
	}
	var reservationStatus string
	if err := db.New(database).InTx(ctx, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, `SELECT status FROM mini_stock.reservations
			WHERE tenant_id = $1 AND order_id = $2`, reservationTenant, orderID).Scan(&reservationStatus)
	}); err != nil {
		t.Fatal(err)
	}
	if reservationStatus != "RELEASED" {
		t.Fatalf("reservation status = %s, want RELEASED", reservationStatus)
	}
}

func waitForStockRelease(t *testing.T, database *pgxpool.Pool, ctx context.Context, skus ...string) {
	t.Helper()
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()
	for {
		allReleased := true
		for _, sku := range skus {
			if got := readReserved(t, database, ctx, sku); got != 0 {
				allReleased = false
			}
		}
		if allReleased {
			return
		}
		select {
		case <-ctx.Done():
			t.Fatalf("reservation was not released through JetStream before timeout: %v", ctx.Err())
		case <-ticker.C:
		}
	}
}

func TestReservation_AC08_ConcurrentOrdersCannotOversell(t *testing.T) {
	database, ctx := reservationTestDatabase(t)
	const rounds = 20
	type attempt struct {
		result domain.ReservationResult
		err    error
	}
	type contest struct {
		sku     string
		ids     [2]uuid.UUID
		results [2]attempt
	}
	contests := make([]contest, rounds)
	for i := range contests {
		contests[i].sku = uniqueReservationSKU(t, fmt.Sprintf("AC08-%02d", i))
		insertReservationStock(t, database, ctx, contests[i].sku, 1)
		for j := range contests[i].ids {
			contests[i].ids[j] = uuid.Must(uuid.NewV7())
		}
	}
	repository := NewReservations(db.New(database))
	start := make(chan struct{})
	var workers sync.WaitGroup
	for i := range contests {
		for competitor := range contests[i].ids {
			workers.Add(1)
			go func(i, competitor int) {
				defer workers.Done()
				<-start
				orderID := contests[i].ids[competitor]
				identity := tenant.Identity{TenantID: reservationTenant, UserID: "system"}
				callCtx := tenant.WithIdentity(ctx, identity)
				result, err := repository.ReserveOrder(callCtx, uuid.Must(uuid.NewV7()), orderID.String(), []domain.ReservationLine{{SKU: contests[i].sku, Quantity: 1}})
				contests[i].results[competitor] = attempt{result: result, err: err}
			}(i, competitor)
		}
	}
	close(start)
	workers.Wait()
	for _, contest := range contests {
		held, short := 0, 0
		for _, result := range contest.results {
			if result.err != nil {
				t.Fatal(result.err)
			}
			if len(result.result.ShortSKUs) == 0 {
				held++
			} else {
				short++
			}
		}
		if held != 1 || short != 1 {
			t.Fatalf("SKU %s: held=%d short=%d, want one each", contest.sku, held, short)
		}
		if got := readReserved(t, database, ctx, contest.sku); got != 1 {
			t.Fatalf("SKU %s reserved = %d, want 1", contest.sku, got)
		}
	}
}

func TestStockAdjustment_EX03_RejectsAmountBelowReserved(t *testing.T) {
	database, ctx := reservationTestDatabase(t)
	sku := uniqueReservationSKU(t, "EX03")
	insertReservationStock(t, database, ctx, sku, 5)
	if err := db.New(database).InTx(ctx, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `UPDATE mini_stock.stock_items SET reserved = 2
			WHERE tenant_id = $1 AND sku = $2`, reservationTenant, sku)
		return err
	}); err != nil {
		t.Fatal(err)
	}

	repository := NewItems(db.New(database))
	if _, err := repository.AdjustStock(ctx, sku, -4); !errors.Is(err, domain.ErrBelowReserved) {
		t.Fatalf("adjust below reserved returned %v, want ErrBelowReserved", err)
	}
	adjusted, err := repository.AdjustStock(ctx, sku, -3)
	if err != nil {
		t.Fatal(err)
	}
	if adjusted.OnHand != 2 || adjusted.Reserved != 2 || adjusted.Available() != 0 {
		t.Fatalf("adjusted stock = %+v, want on_hand=2 reserved=2 available=0", adjusted)
	}
}

func TestReservation_AC08_WithoutLock_ConcurrentWritesHitConstraint(t *testing.T) {
	database, ctx := reservationTestDatabase(t)
	sku := uniqueReservationSKU(t, "AC08-NOLOCK")
	insertReservationStock(t, database, ctx, sku, 1)
	ready := make(chan struct{}, 2)
	release := make(chan struct{})
	errs := make(chan error, 2)
	for range 2 {
		go func() {
			err := db.New(database).InTx(ctx, func(tx pgx.Tx) error {
				var onHand, reserved int64
				if err := tx.QueryRow(ctx, `SELECT on_hand, reserved FROM mini_stock.stock_items
					WHERE tenant_id = $1 AND sku = $2`, reservationTenant, sku).Scan(&onHand, &reserved); err != nil {
					return err
				}
				if onHand-reserved != 1 {
					return fmt.Errorf("fixture available = %d, want 1", onHand-reserved)
				}
				ready <- struct{}{}
				<-release
				_, err := tx.Exec(ctx, `UPDATE mini_stock.stock_items SET reserved = reserved + 1
					WHERE tenant_id = $1 AND sku = $2`, reservationTenant, sku)
				return err
			})
			errs <- err
		}()
	}
	for range 2 {
		select {
		case <-ready:
		case <-ctx.Done():
			t.Fatal("transactions did not both read available stock")
		}
	}
	close(release)
	firstErr, secondErr := <-errs, <-errs
	if (firstErr == nil) == (secondErr == nil) {
		t.Fatalf("unlocked updates returned errors (%v, %v), want exactly one constraint error", firstErr, secondErr)
	}
	if got := readReserved(t, database, ctx, sku); got != 1 {
		t.Fatalf("database constraint allowed reserved = %d, want 1", got)
	}
}

func reservationTestDatabase(t *testing.T) (*pgxpool.Pool, context.Context) {
	t.Helper()
	url := os.Getenv("MINI_TEST_APP_DATABASE_URL")
	if url == "" {
		t.Fatal("MINI_TEST_APP_DATABASE_URL is required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)
	database, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(database.Close)
	identity := tenant.Identity{TenantID: reservationTenant, UserID: "chi", Capabilities: map[string]bool{"order.approve": true}}
	return database, tenant.WithIdentity(ctx, identity)
}

func insertReservationStock(t *testing.T, database *pgxpool.Pool, ctx context.Context, sku string, onHand int64) {
	t.Helper()
	err := db.New(database).InTx(ctx, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `INSERT INTO mini_stock.stock_items
			(tenant_id, sku, name, unit_price, currency_code, on_hand, reserved)
			VALUES ($1, $2, $2, 100.00, 'VND', $3, 0)`, reservationTenant, sku, onHand)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
}

func reservationHandler(database *pgxpool.Pool) *consumer.OrderApproved {
	repository := NewReservations(db.New(database))
	return consumer.NewOrderApproved(usecase.NewReservations(repository))
}

func approvedEvent(t *testing.T, orderID uuid.UUID, lines map[string]int64) []byte {
	t.Helper()
	data := events.OrderApproved{OrderID: orderID.String()}
	for sku, quantity := range lines {
		data.Lines = append(data.Lines, struct {
			SKU      string `json:"sku"`
			Quantity int64  `json:"quantity"`
		}{SKU: sku, Quantity: quantity})
	}
	encodedData, err := json.Marshal(data)
	if err != nil {
		t.Fatal(err)
	}
	eventID := uuid.Must(uuid.NewV7())
	payload, err := json.Marshal(events.Envelope{
		ID: eventID, Type: "mini.ord.order.approved", Source: "order",
		TenantID: reservationTenant, CorrelationID: orderID.String(),
		OccurredAt: time.Now().UTC().Format(time.RFC3339Nano), SchemaVersion: 1, Data: encodedData,
	})
	if err != nil {
		t.Fatal(err)
	}
	return payload
}

func cancelledEvent(t *testing.T, orderID uuid.UUID) []byte {
	t.Helper()
	eventID := uuid.Must(uuid.NewV7())
	data, err := json.Marshal(events.OrderCancelled{OrderID: orderID.String()})
	if err != nil {
		t.Fatal(err)
	}
	payload, err := json.Marshal(events.Envelope{
		ID: eventID, Type: "mini.ord.order.cancelled", Source: "order",
		TenantID: reservationTenant, CorrelationID: orderID.String(),
		OccurredAt: time.Now().UTC().Format(time.RFC3339Nano), SchemaVersion: 1, Data: data,
	})
	if err != nil {
		t.Fatal(err)
	}
	return payload
}

func decodeStockResult(t *testing.T, database *pgxpool.Pool, ctx context.Context, orderID uuid.UUID, expectedType string, target any) {
	t.Helper()
	payload := readStockResultPayload(t, database, ctx, orderID)
	var envelope events.Envelope
	if err := json.Unmarshal(payload, &envelope); err != nil {
		t.Fatal(err)
	}
	if envelope.Type != expectedType {
		t.Fatalf("result type = %s, want %s", envelope.Type, expectedType)
	}
	if err := json.Unmarshal(envelope.Data, target); err != nil {
		t.Fatal(err)
	}
}

func readStockResultPayload(t *testing.T, database *pgxpool.Pool, ctx context.Context, orderID uuid.UUID) []byte {
	t.Helper()
	var payload []byte
	err := db.New(database).InTx(ctx, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, `SELECT payload FROM mini_stock.outbox
			WHERE tenant_id = $1 AND payload->>'correlationId' = $2`, reservationTenant, orderID.String()).Scan(&payload)
	})
	if err != nil {
		t.Fatal(err)
	}
	return payload
}

func readReserved(t *testing.T, database *pgxpool.Pool, ctx context.Context, sku string) int64 {
	t.Helper()
	var result int64
	err := db.New(database).InTx(ctx, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, `SELECT reserved FROM mini_stock.stock_items WHERE tenant_id = $1 AND sku = $2`, reservationTenant, sku).Scan(&result)
	})
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func readReservationCount(t *testing.T, database *pgxpool.Pool, ctx context.Context, orderID uuid.UUID) int64 {
	t.Helper()
	var result int64
	err := db.New(database).InTx(ctx, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, `SELECT count(*) FROM mini_stock.reservations WHERE tenant_id = $1 AND order_id = $2`, reservationTenant, orderID).Scan(&result)
	})
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func readStockResultCount(t *testing.T, database *pgxpool.Pool, ctx context.Context, orderID uuid.UUID) int64 {
	t.Helper()
	var result int64
	err := db.New(database).InTx(ctx, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, `SELECT count(*) FROM mini_stock.outbox WHERE tenant_id = $1 AND payload->>'correlationId' = $2`, reservationTenant, orderID.String()).Scan(&result)
	})
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func uniqueReservationSKU(t *testing.T, prefix string) string {
	t.Helper()
	id, err := uuid.NewV7()
	if err != nil {
		t.Fatal(err)
	}
	return prefix + "-" + id.String()
}
